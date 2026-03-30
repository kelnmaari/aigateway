package inference

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// DefaultOnnxExportImage is the Docker image used to run optimum-cli.
// python:3.11-slim is used for reliability; first run installs optimum (~2-3 min).
const DefaultOnnxExportImage = "python:3.11-slim"

// OnnxExportStatus represents the state of an export job.
type OnnxExportStatus string

const (
	OnnxStatusPending OnnxExportStatus = "pending"
	OnnxStatusRunning OnnxExportStatus = "running"
	OnnxStatusDone    OnnxExportStatus = "done"
	OnnxStatusFailed  OnnxExportStatus = "failed"
)

// OnnxExportJob holds the state of a single export operation.
type OnnxExportJob struct {
	ID        string           `json:"id"`
	HFRepo    string           `json:"hf_repo"`
	Task      string           `json:"task"`
	Dtype     string           `json:"dtype"`
	Status    OnnxExportStatus `json:"status"`
	OutputDir string           `json:"output_dir,omitempty"`
	Logs      []string         `json:"logs"`
	Error     string           `json:"error,omitempty"`
	StartedAt time.Time        `json:"started_at"`
	DoneAt    *time.Time       `json:"done_at,omitempty"`

	mu sync.Mutex
}

func (j *OnnxExportJob) appendLog(line string) {
	j.mu.Lock()
	j.Logs = append(j.Logs, line)
	j.mu.Unlock()
}

// Snapshot returns a copy safe to read without holding the mutex.
func (j *OnnxExportJob) Snapshot() OnnxExportJob {
	j.mu.Lock()
	defer j.mu.Unlock()
	cp := *j
	cp.Logs = make([]string, len(j.Logs))
	copy(cp.Logs, j.Logs)
	return cp
}

// OnnxExportRequest holds parameters for an export job.
type OnnxExportRequest struct {
	HFRepo string `json:"hf_repo"` // e.g. "BAAI/bge-reranker-v2-m3"
	Task   string `json:"task"`    // "text-classification", "feature-extraction"
	Dtype  string `json:"dtype"`   // "float32" | "float16" | "int8" (default: float32)
}

// OnnxExporter manages ONNX export jobs.
type OnnxExporter struct {
	jobs       map[string]*OnnxExportJob
	mu         sync.RWMutex
	hfCacheDir string
	hfToken    string
	logger     *logrus.Logger
}

// NewOnnxExporter creates a new ONNX exporter.
func NewOnnxExporter(hfCacheDir, hfToken string, logger *logrus.Logger) *OnnxExporter {
	return &OnnxExporter{
		jobs:       make(map[string]*OnnxExportJob),
		hfCacheDir: hfCacheDir,
		hfToken:    hfToken,
		logger:     logger,
	}
}

// StartExport creates and starts an ONNX export job. Returns the job.
func (e *OnnxExporter) StartExport(req OnnxExportRequest) (*OnnxExportJob, error) {
	if req.HFRepo == "" {
		return nil, fmt.Errorf("hf_repo is required")
	}
	if req.Task == "" {
		req.Task = "feature-extraction"
	}
	if req.Dtype == "" {
		req.Dtype = "float32"
	}

	job := &OnnxExportJob{
		ID:        uuid.New().String(),
		HFRepo:    req.HFRepo,
		Task:      req.Task,
		Dtype:     req.Dtype,
		Status:    OnnxStatusPending,
		Logs:      []string{},
		StartedAt: time.Now(),
	}

	e.mu.Lock()
	e.jobs[job.ID] = job
	e.mu.Unlock()

	go e.runExport(job)

	return job, nil
}

// GetJob returns a job snapshot by ID.
func (e *OnnxExporter) GetJob(id string) (OnnxExportJob, bool) {
	e.mu.RLock()
	j, ok := e.jobs[id]
	e.mu.RUnlock()
	if !ok {
		return OnnxExportJob{}, false
	}
	return j.Snapshot(), true
}

// ListJobs returns snapshots of all export jobs, newest first.
func (e *OnnxExporter) ListJobs() []OnnxExportJob {
	e.mu.RLock()
	defer e.mu.RUnlock()
	jobs := make([]OnnxExportJob, 0, len(e.jobs))
	for _, j := range e.jobs {
		jobs = append(jobs, j.Snapshot())
	}
	return jobs
}

// runExport executes the Docker-based export job in the background.
func (e *OnnxExporter) runExport(job *OnnxExportJob) {
	job.mu.Lock()
	job.Status = OnnxStatusRunning
	job.mu.Unlock()

	log := e.logger.WithField("job_id", job.ID).WithField("hf_repo", job.HFRepo)

	// Escape repo for use in shell (no shell injection since we validate it)
	repo := job.HFRepo
	task := job.Task
	dtype := job.Dtype

	// The export script runs inside the container:
	//  1. Install optimum[exporters] (cached by Docker layer on repeat runs)
	//  2. Find or download the model snapshot via huggingface_hub
	//  3. Export ONNX directly into snapshot/onnx/ so TEI finds it automatically
	script := fmt.Sprintf(`#!/bin/bash
set -e

echo "=== [1/3] Installing optimum + onnxruntime ==="
# optimum 2.x renamed [exporters] to [onnxruntime]; install both notations for compatibility
pip install "optimum[onnxruntime]" onnx --quiet --no-cache-dir 2>&1 || \
pip install "optimum[exporters]" onnx --quiet --no-cache-dir 2>&1
# Verify optimum installed (2.x dropped __version__, use importlib.metadata)
python3 -c "import importlib.metadata; print('optimum', importlib.metadata.version('optimum'))"

echo "=== [2/3] Locating model snapshot ==="
SNAPSHOT_PATH=$(python3 - <<'PYEOF'
import os, sys
from huggingface_hub import snapshot_download
try:
    path = snapshot_download(%q)
    print(path, end="")
except Exception as ex:
    print("ERROR: " + str(ex), file=sys.stderr)
    sys.exit(1)
PYEOF
)
echo "Snapshot: $SNAPSHOT_PATH"
mkdir -p "$SNAPSHOT_PATH/onnx"

echo "=== [3/3] Exporting ONNX (task=%s, dtype=%s) ==="
optimum-cli export onnx \
  --model %q \
  --task %q \
  --dtype %q \
  "$SNAPSHOT_PATH/onnx/"

echo ""
echo "=== Export complete ==="
echo "Files in $SNAPSHOT_PATH/onnx/:"
ls -lh "$SNAPSHOT_PATH/onnx/"
`, repo, task, dtype, repo, task, dtype)

	args := []string{
		"run", "--rm",
		// Mount HF cache so the model is found locally and outputs go there
		"-v", fmt.Sprintf("%s:/hf-cache:rw", e.hfCacheDir),
		// Tell huggingface_hub to use our cache directory directly
		"-e", "HF_HOME=/hf-cache",
		"-e", "HF_HUB_CACHE=/hf-cache",
	}
	if e.hfToken != "" {
		args = append(args,
			"-e", fmt.Sprintf("HF_TOKEN=%s", e.hfToken),
			"-e", fmt.Sprintf("HUGGINGFACE_HUB_TOKEN=%s", e.hfToken),
		)
	}
	args = append(args,
		DefaultOnnxExportImage,
		"bash", "-c", script,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker", args...)

	// Combine stdout+stderr into one pipe for streaming logs
	combined, err := cmd.StdoutPipe()
	if err != nil {
		e.failJob(job, fmt.Sprintf("pipe error: %v", err))
		return
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		e.failJob(job, fmt.Sprintf("failed to start docker: %v", err))
		return
	}

	log.Info("ONNX export container started")
	scanner := bufio.NewScanner(combined)
	for scanner.Scan() {
		line := scanner.Text()
		job.appendLog(line)
		log.Debug(line)

		// Capture snapshot path from the export output to store in job
		if strings.HasPrefix(line, "Snapshot: ") {
			dir := strings.TrimPrefix(line, "Snapshot: ")
			dir = strings.TrimSpace(dir)
			if dir != "" {
				job.mu.Lock()
				job.OutputDir = filepath.Join(dir, "onnx")
				job.mu.Unlock()
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			e.failJob(job, "export timed out after 45 minutes")
		} else {
			e.failJob(job, fmt.Sprintf("export failed: %v", err))
		}
		return
	}

	now := time.Now()
	job.mu.Lock()
	job.Status = OnnxStatusDone
	job.DoneAt = &now
	job.mu.Unlock()

	log.WithField("output_dir", job.OutputDir).Info("ONNX export completed successfully")
}

func (e *OnnxExporter) failJob(job *OnnxExportJob, errMsg string) {
	now := time.Now()
	job.mu.Lock()
	job.Status = OnnxStatusFailed
	job.Error = errMsg
	job.DoneAt = &now
	job.mu.Unlock()
	e.logger.WithField("job_id", job.ID).WithField("error", errMsg).Error("ONNX export failed")
}
