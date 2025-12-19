package inference

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// ProviderLogger handles detailed logging for provider container launches.
type ProviderLogger struct {
	dir    string
	mu     sync.Mutex
	logger *logrus.Logger
}

// NewProviderLogger creates a new provider logger.
func NewProviderLogger(logsDir string, logger *logrus.Logger) *ProviderLogger {
	dir := filepath.Join(logsDir, "providers")
	if err := os.MkdirAll(dir, 0755); err != nil && logger != nil {
		logger.WithError(err).Warn("Failed to create providers log directory")
	}
	return &ProviderLogger{
		dir:    dir,
		logger: logger,
	}
}

// ProviderLaunchLog contains all details about a provider launch.
type ProviderLaunchLog struct {
	Timestamp   string            `json:"timestamp"`
	ModelAlias  string            `json:"model_alias"`
	Provider    string            `json:"provider"`
	Image       string            `json:"image"`
	Command     []string          `json:"command"`
	CommandStr  string            `json:"command_string"`
	Environment map[string]string `json:"environment"`
	Mounts      []MountLog        `json:"mounts"`
	Ports       map[string]int    `json:"ports"`
	GPUDevice   string            `json:"gpu_device,omitempty"`
	GPUMode     string            `json:"gpu_mode"`
	ModelSpec   ModelSpecLog      `json:"model_spec"`
}

// MountLog represents a volume mount in logs.
type MountLog struct {
	HostPath      string `json:"host_path"`
	ContainerPath string `json:"container_path"`
	ReadOnly      bool   `json:"read_only"`
}

// ModelSpecLog contains model spec details for logging.
type ModelSpecLog struct {
	HFRepo    string `json:"hf_repo,omitempty"`
	LocalPath string `json:"local_path,omitempty"`
	Format    string `json:"format"`

	// vLLM specific
	VLLMTensorParallel  int     `json:"vllm_tensor_parallel,omitempty"`
	VLLMMaxModelLen     int     `json:"vllm_max_model_len,omitempty"`
	VLLMGPUUtilization  float64 `json:"vllm_gpu_utilization,omitempty"`

	// SGLang specific
	SGLangTensorParallel int     `json:"sglang_tensor_parallel,omitempty"`
	SGLangDataParallel   int     `json:"sglang_data_parallel,omitempty"`
	SGLangMemFraction    float64 `json:"sglang_mem_fraction,omitempty"`
	SGLangContextLen     int     `json:"sglang_context_len,omitempty"`
	SGLangChunkedPrefill bool    `json:"sglang_chunked_prefill,omitempty"`

	// TGI specific
	TGINumShard           int `json:"tgi_num_shard,omitempty"`
	TGIMaxConcurrentReqs  int `json:"tgi_max_concurrent_reqs,omitempty"`
	TGIMaxInputLen        int `json:"tgi_max_input_len,omitempty"`
	TGIMaxTotalTokens     int `json:"tgi_max_total_tokens,omitempty"`

	// llama.cpp specific
	LlamaNGPULayers  int    `json:"llama_n_gpu_layers,omitempty"`
	LlamaMainGPU     int    `json:"llama_main_gpu,omitempty"`
	LlamaTensorSplit string `json:"llama_tensor_split,omitempty"`
}

// LogLaunch logs a container launch request.
func (pl *ProviderLogger) LogLaunch(spec ModelSpec, req ContainerStartRequest) {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	// Determine GPU mode description
	gpuMode := "all GPUs"
	if req.GPUDevice != "" {
		gpuMode = fmt.Sprintf("specific GPU(s): %s", req.GPUDevice)
	}

	// Build mounts log
	mounts := make([]MountLog, len(req.Mounts))
	for i, m := range req.Mounts {
		mounts[i] = MountLog{
			HostPath:      m.HostPath,
			ContainerPath: m.ContainerPath,
			ReadOnly:      m.ReadOnly,
		}
	}

	// Sanitize environment (hide tokens)
	safeEnv := make(map[string]string)
	for k, v := range req.Env {
		if strings.Contains(strings.ToLower(k), "token") || strings.Contains(strings.ToLower(k), "key") {
			safeEnv[k] = "[REDACTED]"
		} else {
			safeEnv[k] = v
		}
	}

	log := ProviderLaunchLog{
		Timestamp:   time.Now().Format(time.RFC3339),
		ModelAlias:  req.ModelAlias,
		Provider:    string(req.Provider),
		Image:       req.Image,
		Command:     req.Command,
		CommandStr:  strings.Join(req.Command, " "),
		Environment: safeEnv,
		Mounts:      mounts,
		Ports:       req.Ports,
		GPUDevice:   req.GPUDevice,
		GPUMode:     gpuMode,
		ModelSpec: ModelSpecLog{
			HFRepo:               spec.HFRepo,
			LocalPath:            spec.LocalPath,
			Format:               string(spec.Format),
			VLLMTensorParallel:   spec.VLLMTensorParallel,
			VLLMMaxModelLen:      spec.VLLMMaxModelLen,
			VLLMGPUUtilization:   spec.VLLMGPUUtilization,
			SGLangTensorParallel: spec.SGLangTensorParallel,
			SGLangDataParallel:   spec.SGLangDataParallel,
			SGLangMemFraction:    spec.SGLangMemFraction,
			SGLangContextLen:     spec.SGLangContextLen,
			SGLangChunkedPrefill: spec.SGLangChunkedPrefill,
			TGINumShard:          spec.TGINumShard,
			TGIMaxConcurrentReqs: spec.TGIMaxConcurrentReqs,
			TGIMaxInputLen:       spec.TGIMaxInputLen,
			TGIMaxTotalTokens:    spec.TGIMaxTotalTokens,
			LlamaNGPULayers:      spec.LlamaNGPULayers,
			LlamaMainGPU:         spec.LlamaMainGPU,
			LlamaTensorSplit:     spec.LlamaTensorSplit,
		},
	}

	// Log to main logger
	if pl.logger != nil {
		pl.logger.WithFields(logrus.Fields{
			"provider":    log.Provider,
			"alias":       log.ModelAlias,
			"image":       log.Image,
			"command":     log.CommandStr,
			"gpu_mode":    log.GPUMode,
			"gpu_device":  log.GPUDevice,
			"mounts":      len(log.Mounts),
		}).Info("Starting inference container")
	}

	// Write to provider-specific log file
	filename := filepath.Join(pl.dir, fmt.Sprintf("%s.log", sanitizeFilename(req.ModelAlias)))
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		if pl.logger != nil {
			pl.logger.WithError(err).Warn("Failed to open provider log file")
		}
		return
	}
	defer f.Close()

	// Write human-readable header
	fmt.Fprintf(f, "\n%s\n", strings.Repeat("=", 80))
	fmt.Fprintf(f, "CONTAINER LAUNCH: %s\n", log.Timestamp)
	fmt.Fprintf(f, "%s\n", strings.Repeat("=", 80))
	fmt.Fprintf(f, "Model Alias:  %s\n", log.ModelAlias)
	fmt.Fprintf(f, "Provider:     %s\n", log.Provider)
	fmt.Fprintf(f, "Image:        %s\n", log.Image)
	fmt.Fprintf(f, "GPU Mode:     %s\n", log.GPUMode)
	fmt.Fprintf(f, "\n--- Command ---\n")
	fmt.Fprintf(f, "%s\n", log.CommandStr)
	fmt.Fprintf(f, "\n--- Full Docker Command ---\n")
	fmt.Fprintf(f, "%s\n", pl.buildDockerCommand(req))
	fmt.Fprintf(f, "\n--- Environment Variables ---\n")
	for k, v := range safeEnv {
		fmt.Fprintf(f, "%s=%s\n", k, v)
	}
	fmt.Fprintf(f, "\n--- Volume Mounts ---\n")
	for _, m := range log.Mounts {
		ro := ""
		if m.ReadOnly {
			ro = " (readonly)"
		}
		fmt.Fprintf(f, "%s -> %s%s\n", m.HostPath, m.ContainerPath, ro)
	}
	fmt.Fprintf(f, "\n--- Ports ---\n")
	for name, port := range log.Ports {
		fmt.Fprintf(f, "%s: %d\n", name, port)
	}
	fmt.Fprintf(f, "\n--- Model Spec (JSON) ---\n")
	specJSON, _ := json.MarshalIndent(log.ModelSpec, "", "  ")
	fmt.Fprintf(f, "%s\n", specJSON)
	fmt.Fprintf(f, "\n--- Full Log Entry (JSON) ---\n")
	fullJSON, _ := json.MarshalIndent(log, "", "  ")
	fmt.Fprintf(f, "%s\n", fullJSON)
}

// buildDockerCommand reconstructs the equivalent docker run command.
func (pl *ProviderLogger) buildDockerCommand(req ContainerStartRequest) string {
	var parts []string
	parts = append(parts, "docker run -d --rm")

	// GPU access via --gpus all + NVIDIA_VISIBLE_DEVICES
	isMultiGPU := strings.Contains(req.GPUDevice, ",") || req.GPUDevice == ""
	parts = append(parts, "--gpus all")
	if req.GPUDevice != "" {
		// Shown separately via -e NVIDIA_VISIBLE_DEVICES
	}

	// Multi-GPU NCCL flags
	if isMultiGPU {
		parts = append(parts, "--ipc=host")
		parts = append(parts, "--shm-size=16g")
		parts = append(parts, "--ulimit memlock=-1:-1")
	}

	// Ports (show as <host>:<container> with placeholder for dynamic host port)
	for name, cport := range req.Ports {
		parts = append(parts, fmt.Sprintf("-p <DYNAMIC>:%d", cport))
		_ = name // suppress unused warning
	}

	// Environment
	for k, v := range req.Env {
		if strings.Contains(strings.ToLower(k), "token") || strings.Contains(strings.ToLower(k), "key") {
			parts = append(parts, fmt.Sprintf("-e %s=[REDACTED]", k))
		} else {
			parts = append(parts, fmt.Sprintf("-e %s=%s", k, v))
		}
	}

	// Mounts
	for _, m := range req.Mounts {
		mode := "rw"
		if m.ReadOnly {
			mode = "ro"
		}
		parts = append(parts, fmt.Sprintf("-v %s:%s:%s", m.HostPath, m.ContainerPath, mode))
	}

	// Image
	parts = append(parts, req.Image)

	// Command
	if len(req.Command) > 0 {
		parts = append(parts, strings.Join(req.Command, " "))
	}

	return strings.Join(parts, " \\\n  ")
}

// LogError logs a container launch error.
func (pl *ProviderLogger) LogError(alias, provider string, err error) {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	if pl.logger != nil {
		pl.logger.WithFields(logrus.Fields{
			"provider": provider,
			"alias":    alias,
			"error":    err.Error(),
		}).Error("Failed to start inference container")
	}

	filename := filepath.Join(pl.dir, fmt.Sprintf("%s.log", sanitizeFilename(alias)))
	f, ferr := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if ferr != nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "\n%s ERROR: Container launch failed\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(f, "Error: %s\n", err.Error())
}

// LogSuccess logs a successful container start.
func (pl *ProviderLogger) LogSuccess(alias, provider, containerID, endpoint string) {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	if pl.logger != nil {
		pl.logger.WithFields(logrus.Fields{
			"provider":     provider,
			"alias":        alias,
			"container_id": containerID[:12],
			"endpoint":     endpoint,
		}).Info("Inference container started successfully")
	}

	filename := filepath.Join(pl.dir, fmt.Sprintf("%s.log", sanitizeFilename(alias)))
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Fprintf(f, "\n%s SUCCESS: Container started\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(f, "Container ID: %s\n", containerID)
	fmt.Fprintf(f, "Endpoint:     %s\n", endpoint)
}

func sanitizeFilename(name string) string {
	// Replace characters that are problematic for filenames
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	return replacer.Replace(name)
}

