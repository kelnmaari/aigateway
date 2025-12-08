//go:build docker_integration
// +build docker_integration

// Docker integration tests for inference providers.
// Run with: go test -tags=docker_integration -v ./internal/inference/...
//
// Prerequisites:
// - Docker Desktop with NVIDIA Container Toolkit (WSL2 backend on Windows)
// - NVIDIA GPU available
// - Network access to pull Docker images
//
// Environment variables:
// - DOCKER_TEST_TIMEOUT: timeout for container operations (default: 5m)
// - DOCKER_TEST_GPU: set to "false" to skip GPU tests (default: "true")
// - HF_TOKEN: HuggingFace token for gated models (optional)

package inference

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"
)

func getTestTimeout() time.Duration {
	if v := os.Getenv("DOCKER_TEST_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return 5 * time.Minute
}

func skipIfNoGPU(t *testing.T) {
	if v := os.Getenv("DOCKER_TEST_GPU"); v == "false" {
		t.Skip("Skipping GPU test (DOCKER_TEST_GPU=false)")
	}
}

func TestDockerIntegration_RuntimeAvailable(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	runtime := NewDockerRuntime()

	// Test Docker connection
	containers, err := runtime.ListContainers(ctx)
	if err != nil {
		t.Fatalf("Docker not available: %v", err)
	}

	t.Logf("Docker runtime available, found %d containers", len(containers))
}

func TestDockerIntegration_PullImage(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), getTestTimeout())
	defer cancel()

	runtime := NewDockerRuntime()

	// Pull a small test image
	testImage := "alpine:latest"

	t.Logf("Pulling image: %s", testImage)
	if err := runtime.PullImage(ctx, testImage); err != nil {
		t.Fatalf("Failed to pull image: %v", err)
	}

	t.Log("Image pulled successfully")
}

func TestDockerIntegration_LlamaCPP_StartStop(t *testing.T) {
	skipIfNoGPU(t)

	ctx, cancel := context.WithTimeout(context.Background(), getTestTimeout())
	defer cancel()

	runtime := NewDockerRuntime()

	// Use a small GGUF model for testing
	// This test assumes you have a small GGUF model available
	// You can download one for testing:
	// wget https://huggingface.co/TheBloke/TinyLlama-1.1B-Chat-v1.0-GGUF/resolve/main/tinyllama-1.1b-chat-v1.0.Q4_K_M.gguf

	modelPath := os.Getenv("DOCKER_TEST_GGUF_PATH")
	if modelPath == "" {
		t.Skip("DOCKER_TEST_GGUF_PATH not set, skipping llama.cpp test")
	}

	req := &ContainerRequest{
		Image:      "ghcr.io/ggerganov/llama.cpp:server-cuda",
		Name:       "test-llamacpp-" + strconv.FormatInt(time.Now().UnixNano(), 36),
		Port:       8081,
		GPUEnabled: true,
		Mounts: []Mount{
			{Source: modelPath, Target: "/models/model.gguf", ReadOnly: true},
		},
		Cmd: []string{
			"--model", "/models/model.gguf",
			"--port", "8081",
			"--host", "0.0.0.0",
			"-ngl", "99", // GPU layers
		},
	}

	t.Logf("Starting llama.cpp container: %s", req.Name)
	containerID, err := runtime.StartContainer(ctx, req)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}

	t.Logf("Container started: %s", containerID)

	// Cleanup
	defer func() {
		t.Log("Stopping container...")
		if err := runtime.StopContainer(ctx, containerID); err != nil {
			t.Errorf("Failed to stop container: %v", err)
		}
		t.Log("Container stopped")
	}()

	// Wait for container to be ready
	t.Log("Waiting for container to be healthy...")
	for i := 0; i < 60; i++ {
		status, err := runtime.ContainerStatus(ctx, containerID)
		if err != nil {
			t.Logf("Status check %d: error: %v", i, err)
		} else {
			t.Logf("Status check %d: %s", i, status)
			if status == "running" {
				// Give the server a moment to start accepting connections
				time.Sleep(5 * time.Second)
				break
			}
		}
		time.Sleep(2 * time.Second)
	}

	// Get logs
	logs, err := runtime.ContainerLogs(ctx, containerID, 50)
	if err != nil {
		t.Errorf("Failed to get logs: %v", err)
	} else {
		t.Logf("Container logs:\n%s", logs)
	}

	t.Log("llama.cpp container test passed")
}

func TestDockerIntegration_VLLM_StartStop(t *testing.T) {
	skipIfNoGPU(t)

	ctx, cancel := context.WithTimeout(context.Background(), getTestTimeout())
	defer cancel()

	runtime := NewDockerRuntime()

	// vLLM requires significant GPU memory
	// Use a small model for testing
	hfModel := os.Getenv("DOCKER_TEST_HF_MODEL")
	if hfModel == "" {
		hfModel = "facebook/opt-125m" // Small model for testing
	}

	modelsDir := os.Getenv("DOCKER_TEST_MODELS_DIR")
	if modelsDir == "" {
		t.Skip("DOCKER_TEST_MODELS_DIR not set, skipping vLLM test")
	}

	hfToken := os.Getenv("HF_TOKEN")

	env := []string{}
	if hfToken != "" {
		env = append(env, "HF_TOKEN="+hfToken)
	}

	req := &ContainerRequest{
		Image:      "vllm/vllm-openai:latest",
		Name:       "test-vllm-" + strconv.FormatInt(time.Now().UnixNano(), 36),
		Port:       8000,
		GPUEnabled: true,
		Env:        env,
		Mounts: []Mount{
			{Source: modelsDir, Target: "/root/.cache/huggingface", ReadOnly: false},
		},
		Cmd: []string{
			"--model", hfModel,
			"--port", "8000",
			"--host", "0.0.0.0",
			"--gpu-memory-utilization", "0.5", // Conservative for testing
		},
	}

	t.Logf("Starting vLLM container: %s with model %s", req.Name, hfModel)
	containerID, err := runtime.StartContainer(ctx, req)
	if err != nil {
		t.Fatalf("Failed to start container: %v", err)
	}

	t.Logf("Container started: %s", containerID)

	// Cleanup
	defer func() {
		t.Log("Stopping container...")
		if err := runtime.StopContainer(ctx, containerID); err != nil {
			t.Errorf("Failed to stop container: %v", err)
		}
		t.Log("Container stopped")
	}()

	// Wait for container to be ready (vLLM takes longer to start)
	t.Log("Waiting for vLLM container to be healthy (this may take a few minutes)...")
	for i := 0; i < 120; i++ { // 4 minutes timeout
		status, err := runtime.ContainerStatus(ctx, containerID)
		if err != nil {
			t.Logf("Status check %d: error: %v", i, err)
		} else {
			t.Logf("Status check %d: %s", i, status)
			if status == "running" {
				// vLLM needs more time to load model
				time.Sleep(10 * time.Second)
				break
			}
			if status == "exited" {
				logs, _ := runtime.ContainerLogs(ctx, containerID, 100)
				t.Fatalf("Container exited. Logs:\n%s", logs)
			}
		}
		time.Sleep(2 * time.Second)
	}

	// Get logs
	logs, err := runtime.ContainerLogs(ctx, containerID, 50)
	if err != nil {
		t.Errorf("Failed to get logs: %v", err)
	} else {
		t.Logf("Container logs:\n%s", logs)
	}

	t.Log("vLLM container test passed")
}

func TestDockerIntegration_FullFlow_LlamaCPP(t *testing.T) {
	skipIfNoGPU(t)

	modelPath := os.Getenv("DOCKER_TEST_GGUF_PATH")
	if modelPath == "" {
		t.Skip("DOCKER_TEST_GGUF_PATH not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), getTestTimeout())
	defer cancel()

	cfg := &ServiceConfig{
		GGUFDir:            "/tmp/inference-test-gguf",
		HFCacheDir:         "/tmp/inference-test-hf",
		TRTEngineDir:       "/tmp/inference-test-trt",
		MaxRunningModels:   1,
		HealthCheckTimeout: 30 * time.Second,
		StartupTimeout:     2 * time.Minute,
	}

	runtime := NewDockerRuntime()
	svc := NewService(cfg, runtime, nil)

	// Load model
	req := &LoadRequest{
		Alias:    "test-tiny",
		Provider: ProviderLlamaCPP,
		Format:   FormatGGUF,
		// Assuming modelPath is already downloaded
		Capabilities:      []string{"chat"},
		LlamaNGPULayers:   99,
	}

	// Note: This test assumes the model is already at modelPath
	// In a real scenario, you'd use EnsureGGUF or prepare the model first

	t.Logf("Loading model with alias: %s", req.Alias)

	// Start the model
	if err := svc.Load(ctx, req); err != nil {
		t.Fatalf("Failed to load model: %v", err)
	}

	t.Log("Model loaded, checking status...")

	// Get model status
	models := svc.ListModels()
	found := false
	for _, m := range models {
		if m.Alias == req.Alias {
			found = true
			t.Logf("Model status: %s, endpoint: %s", m.Status, m.Endpoint)
		}
	}
	if !found {
		t.Fatal("Model not found in list")
	}

	// Stop the model
	t.Log("Stopping model...")
	if err := svc.Stop(ctx, req.Alias); err != nil {
		t.Errorf("Failed to stop model: %v", err)
	}

	t.Log("Full flow test passed")
}

// BenchmarkDockerIntegration_ContainerStart measures container startup time
func BenchmarkDockerIntegration_ContainerStart(b *testing.B) {
	if os.Getenv("DOCKER_TEST_GPU") == "false" {
		b.Skip("Skipping GPU benchmark")
	}

	ctx := context.Background()
	runtime := NewDockerRuntime()

	// Pre-pull the image
	if err := runtime.PullImage(ctx, "alpine:latest"); err != nil {
		b.Fatalf("Failed to pull image: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := &ContainerRequest{
			Image: "alpine:latest",
			Name:  "bench-" + strconv.Itoa(i) + "-" + strconv.FormatInt(time.Now().UnixNano(), 36),
			Cmd:   []string{"sleep", "1"},
		}

		containerID, err := runtime.StartContainer(ctx, req)
		if err != nil {
			b.Fatalf("Failed to start container: %v", err)
		}

		if err := runtime.StopContainer(ctx, containerID); err != nil {
			b.Errorf("Failed to stop container: %v", err)
		}
	}
}

