package inference

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// TRTEngineMetadata stores version info for TensorRT engine compatibility validation.
type TRTEngineMetadata struct {
	ModelID       string    `json:"model_id"`        // HF repo or alias
	CUDAVersion   string    `json:"cuda_version"`    // e.g., "12.4"
	TRTVersion    string    `json:"trt_version"`     // e.g., "10.0.1"
	DriverVersion string    `json:"driver_version"`  // e.g., "550.54.14"
	GPUSMVersion  string    `json:"gpu_sm_version"`  // e.g., "89" (Ada Lovelace)
	Dtype         string    `json:"dtype"`           // e.g., "float16", "bfloat16"
	MaxBatchSize  int       `json:"max_batch_size"`  // Compiled batch size
	MaxSeqLen     int       `json:"max_seq_len"`     // Compiled sequence length
	CreatedAt     time.Time `json:"created_at"`
	SourceHash    string    `json:"source_hash"` // Hash of source model for change detection
}

// TRTConverter handles TensorRT-LLM model conversion and caching.
type TRTConverter struct {
	enginesDir     string // /data/engines/trt
	hfCacheDir     string // /data/models
	converterImage string // nvcr.io/nvidia/tensorrt-llm:latest
	logger         *logrus.Logger
}

// TRTConverterConfig holds configuration for TRT converter.
type TRTConverterConfig struct {
	EnginesDir     string
	HFCacheDir     string
	ConverterImage string
	Logger         *logrus.Logger
}

// NewTRTConverter creates a new TensorRT-LLM converter.
func NewTRTConverter(cfg TRTConverterConfig) (*TRTConverter, error) {
	if cfg.Logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	if cfg.EnginesDir == "" {
		return nil, fmt.Errorf("EnginesDir is required")
	}
	if cfg.HFCacheDir == "" {
		return nil, fmt.Errorf("HFCacheDir is required")
	}
	if cfg.ConverterImage == "" {
		cfg.ConverterImage = "nvcr.io/nvidia/tensorrt-llm:latest"
	}

	if err := os.MkdirAll(cfg.EnginesDir, 0755); err != nil {
		return nil, fmt.Errorf("create engines dir: %w", err)
	}

	return &TRTConverter{
		enginesDir:     cfg.EnginesDir,
		hfCacheDir:     cfg.HFCacheDir,
		converterImage: cfg.ConverterImage,
		logger:         cfg.Logger,
	}, nil
}

// ConvertRequest specifies conversion parameters.
type ConvertRequest struct {
	ModelID      string // HF repo ID or local path
	Alias        string // Output engine alias
	Dtype        string // float16, bfloat16, int8, int4
	MaxBatchSize int
	MaxSeqLen    int
	TensorParallel int // Number of GPUs
	Force        bool // Force reconversion even if cached
}

// EnginePath returns the expected path for a converted engine.
func (c *TRTConverter) EnginePath(alias string) string {
	return filepath.Join(c.enginesDir, alias)
}

// MetadataPath returns the path to engine metadata file.
func (c *TRTConverter) MetadataPath(alias string) string {
	return filepath.Join(c.enginesDir, alias, "metadata.json")
}

// GetCachedEngine returns engine path if valid cached engine exists, empty string otherwise.
func (c *TRTConverter) GetCachedEngine(ctx context.Context, alias string) (string, *TRTEngineMetadata, error) {
	enginePath := c.EnginePath(alias)
	metaPath := c.MetadataPath(alias)

	// Check if engine directory exists
	if _, err := os.Stat(enginePath); os.IsNotExist(err) {
		return "", nil, nil
	}

	// Load and validate metadata
	meta, err := c.loadMetadata(metaPath)
	if err != nil {
		c.logger.WithError(err).Warn("failed to load TRT engine metadata, reconversion needed")
		return "", nil, nil
	}

	// Validate compatibility with current system
	sysInfo, err := c.getSystemInfo(ctx)
	if err != nil {
		c.logger.WithError(err).Warn("failed to get system info for TRT validation")
		return enginePath, meta, nil // Return cached engine anyway
	}

	if !c.isCompatible(meta, sysInfo) {
		c.logger.WithFields(logrus.Fields{
			"cached_cuda":    meta.CUDAVersion,
			"cached_trt":     meta.TRTVersion,
			"cached_driver":  meta.DriverVersion,
			"cached_sm":      meta.GPUSMVersion,
			"current_cuda":   sysInfo.CUDAVersion,
			"current_trt":    sysInfo.TRTVersion,
			"current_driver": sysInfo.DriverVersion,
			"current_sm":     sysInfo.GPUSMVersion,
		}).Warn("TRT engine version mismatch, reconversion recommended")
		return "", meta, fmt.Errorf("version mismatch: engine built with CUDA %s/TRT %s/SM %s, current CUDA %s/TRT %s/SM %s",
			meta.CUDAVersion, meta.TRTVersion, meta.GPUSMVersion,
			sysInfo.CUDAVersion, sysInfo.TRTVersion, sysInfo.GPUSMVersion)
	}

	return enginePath, meta, nil
}

// Convert performs HF model to TensorRT-LLM engine conversion.
func (c *TRTConverter) Convert(ctx context.Context, req ConvertRequest) (string, error) {
	if req.Alias == "" {
		req.Alias = sanitizeAlias(req.ModelID)
	}
	if req.Dtype == "" {
		req.Dtype = "float16"
	}
	if req.MaxBatchSize <= 0 {
		req.MaxBatchSize = 8
	}
	if req.MaxSeqLen <= 0 {
		req.MaxSeqLen = 4096
	}
	if req.TensorParallel <= 0 {
		req.TensorParallel = 1
	}

	enginePath := c.EnginePath(req.Alias)

	// Check cache unless force reconversion
	if !req.Force {
		cached, _, err := c.GetCachedEngine(ctx, req.Alias)
		if err == nil && cached != "" {
			c.logger.WithField("alias", req.Alias).Info("using cached TRT engine")
			return cached, nil
		}
	}

	c.logger.WithFields(logrus.Fields{
		"model_id":        req.ModelID,
		"alias":           req.Alias,
		"dtype":           req.Dtype,
		"max_batch_size":  req.MaxBatchSize,
		"max_seq_len":     req.MaxSeqLen,
		"tensor_parallel": req.TensorParallel,
	}).Info("starting TRT-LLM conversion")

	// Clean up existing engine
	if err := os.RemoveAll(enginePath); err != nil {
		c.logger.WithError(err).Warn("failed to clean existing engine dir")
	}
	if err := os.MkdirAll(enginePath, 0755); err != nil {
		return "", fmt.Errorf("create engine dir: %w", err)
	}

	// Determine source model path
	modelPath := req.ModelID
	if !filepath.IsAbs(modelPath) && !strings.HasPrefix(modelPath, "/") {
		// Assume HF repo, look in cache
		modelPath = filepath.Join(c.hfCacheDir, req.ModelID)
		if _, err := os.Stat(modelPath); os.IsNotExist(err) {
			// Try hub cache structure
			modelPath = filepath.Join(c.hfCacheDir, "hub", "models--"+strings.ReplaceAll(req.ModelID, "/", "--"))
		}
	}

	// Run conversion in Docker container
	if err := c.runConversion(ctx, modelPath, enginePath, req); err != nil {
		return "", fmt.Errorf("conversion failed: %w", err)
	}

	// Get system info and save metadata
	sysInfo, err := c.getSystemInfo(ctx)
	if err != nil {
		c.logger.WithError(err).Warn("failed to get system info for metadata")
		sysInfo = &TRTEngineMetadata{} // Use empty metadata
	}

	meta := &TRTEngineMetadata{
		ModelID:       req.ModelID,
		CUDAVersion:   sysInfo.CUDAVersion,
		TRTVersion:    sysInfo.TRTVersion,
		DriverVersion: sysInfo.DriverVersion,
		GPUSMVersion:  sysInfo.GPUSMVersion,
		Dtype:         req.Dtype,
		MaxBatchSize:  req.MaxBatchSize,
		MaxSeqLen:     req.MaxSeqLen,
		CreatedAt:     time.Now(),
	}

	if err := c.saveMetadata(c.MetadataPath(req.Alias), meta); err != nil {
		c.logger.WithError(err).Warn("failed to save TRT engine metadata")
	}

	c.logger.WithFields(logrus.Fields{
		"alias":       req.Alias,
		"engine_path": enginePath,
	}).Info("TRT-LLM conversion completed")

	return enginePath, nil
}

// runConversion executes the actual conversion using Docker.
func (c *TRTConverter) runConversion(ctx context.Context, modelPath, enginePath string, req ConvertRequest) error {
	// Build trtllm-build command
	// See: https://nvidia.github.io/TensorRT-LLM/commands/trtllm-build.html
	args := []string{
		"run", "--rm",
		"--gpus", "all",
		"-v", modelPath + ":/model:ro",
		"-v", enginePath + ":/engine",
		c.converterImage,
		"trtllm-build",
		"--checkpoint_dir", "/model",
		"--output_dir", "/engine",
		"--dtype", req.Dtype,
		"--max_batch_size", fmt.Sprintf("%d", req.MaxBatchSize),
		"--max_seq_len", fmt.Sprintf("%d", req.MaxSeqLen),
	}

	if req.TensorParallel > 1 {
		args = append(args, "--tp_size", fmt.Sprintf("%d", req.TensorParallel))
	}

	c.logger.WithField("args", args).Debug("running TRT conversion")

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker trtllm-build: %w", err)
	}

	// Verify engine was created
	engineFiles, err := filepath.Glob(filepath.Join(enginePath, "*.engine"))
	if err != nil || len(engineFiles) == 0 {
		return fmt.Errorf("no engine files produced")
	}

	return nil
}

// getSystemInfo retrieves current CUDA/TRT/Driver versions.
func (c *TRTConverter) getSystemInfo(ctx context.Context) (*TRTEngineMetadata, error) {
	info := &TRTEngineMetadata{}

	// Get NVIDIA driver version from nvidia-smi
	cmd := exec.CommandContext(ctx, "nvidia-smi", "--query-gpu=driver_version,compute_cap", "--format=csv,noheader,nounits")
	out, err := cmd.Output()
	if err == nil {
		parts := strings.Split(strings.TrimSpace(string(out)), ", ")
		if len(parts) >= 2 {
			info.DriverVersion = strings.TrimSpace(parts[0])
			// Compute capability to SM version (e.g., 8.9 -> 89)
			sm := strings.ReplaceAll(strings.TrimSpace(parts[1]), ".", "")
			info.GPUSMVersion = sm
		}
	}

	// Get CUDA version from nvcc or nvidia-smi
	cmd = exec.CommandContext(ctx, "nvcc", "--version")
	out, err = cmd.Output()
	if err == nil {
		// Parse "Cuda compilation tools, release 12.4, V12.4.131"
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, "release") {
				parts := strings.Split(line, "release ")
				if len(parts) >= 2 {
					ver := strings.Split(parts[1], ",")[0]
					info.CUDAVersion = strings.TrimSpace(ver)
					break
				}
			}
		}
	}

	// Get TensorRT version (from docker image label or trtexec)
	cmd = exec.CommandContext(ctx, "docker", "run", "--rm", c.converterImage, "trtexec", "--version")
	out, err = cmd.Output()
	if err == nil {
		// Parse TensorRT version from output
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if strings.Contains(line, "TensorRT") {
				parts := strings.Fields(line)
				for i, p := range parts {
					if p == "TensorRT" && i+1 < len(parts) {
						info.TRTVersion = parts[i+1]
						break
					}
				}
			}
		}
	}

	return info, nil
}

// isCompatible checks if cached engine is compatible with current system.
func (c *TRTConverter) isCompatible(cached, current *TRTEngineMetadata) bool {
	if cached == nil || current == nil {
		return false
	}

	// Critical: SM version must match (GPU architecture)
	if cached.GPUSMVersion != current.GPUSMVersion && cached.GPUSMVersion != "" && current.GPUSMVersion != "" {
		return false
	}

	// Major CUDA version should match
	cachedCUDAMajor := strings.Split(cached.CUDAVersion, ".")[0]
	currentCUDAMajor := strings.Split(current.CUDAVersion, ".")[0]
	if cachedCUDAMajor != currentCUDAMajor && cachedCUDAMajor != "" && currentCUDAMajor != "" {
		return false
	}

	// TensorRT major version should match
	cachedTRTMajor := strings.Split(cached.TRTVersion, ".")[0]
	currentTRTMajor := strings.Split(current.TRTVersion, ".")[0]
	if cachedTRTMajor != currentTRTMajor && cachedTRTMajor != "" && currentTRTMajor != "" {
		return false
	}

	return true
}

func (c *TRTConverter) loadMetadata(path string) (*TRTEngineMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var meta TRTEngineMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (c *TRTConverter) saveMetadata(path string, meta *TRTEngineMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func sanitizeAlias(s string) string {
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, ":", "-")
	return s
}

// ListEngines returns all cached TRT engines with metadata.
func (c *TRTConverter) ListEngines() ([]TRTEngineMetadata, error) {
	entries, err := os.ReadDir(c.enginesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var engines []TRTEngineMetadata
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		metaPath := filepath.Join(c.enginesDir, entry.Name(), "metadata.json")
		meta, err := c.loadMetadata(metaPath)
		if err != nil {
			continue
		}
		engines = append(engines, *meta)
	}
	return engines, nil
}

// DeleteEngine removes a cached TRT engine.
func (c *TRTConverter) DeleteEngine(alias string) error {
	return os.RemoveAll(c.EnginePath(alias))
}

