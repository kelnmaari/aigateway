package inference

import (
	"testing"
)

func TestBuildVLLMRequest(t *testing.T) {
	spec := ModelSpec{
		Alias:              "test-llama",
		HFRepo:             "meta-llama/Llama-3.1-8B",
		VLLMTensorParallel: 2,
		VLLMMaxModelLen:    4096,
		VLLMGPUUtilization: 0.9,
	}

	req := BuildVLLMRequest(spec, "/data/models", "")

	if req.ModelAlias != "test-llama" {
		t.Errorf("ModelAlias = %q, want %q", req.ModelAlias, "test-llama")
	}
	if req.Provider != ProviderVLLM {
		t.Errorf("Provider = %v, want %v", req.Provider, ProviderVLLM)
	}
	if req.Image != DefaultVLLMImage {
		t.Errorf("Image = %q, want %q", req.Image, DefaultVLLMImage)
	}

	// Check command contains expected flags
	cmdStr := ""
	for _, c := range req.Command {
		cmdStr += c + " "
	}
	if cmdStr == "" {
		t.Error("Command is empty")
	}

	// Should contain tensor-parallel-size 2
	found := false
	for i, c := range req.Command {
		if c == "--tensor-parallel-size" && i+1 < len(req.Command) && req.Command[i+1] == "2" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Command missing --tensor-parallel-size 2")
	}
}

func TestBuildVLLMRequest_UsesLocalPath(t *testing.T) {
	spec := ModelSpec{
		Alias:     "local-model",
		HFRepo:    "meta-llama/Llama-3.1-8B",
		LocalPath: "/data/models/meta-llama/Llama-3.1-8B",
	}

	req := BuildVLLMRequest(spec, "/data/models", "")

	// Should use LocalPath for --model
	found := false
	for i, c := range req.Command {
		if c == "--model" && i+1 < len(req.Command) {
			if req.Command[i+1] == spec.LocalPath {
				found = true
			}
			break
		}
	}
	if !found {
		t.Error("Command should use LocalPath for --model")
	}
}

func TestBuildSGLangRequest_HFTokenIsolation(t *testing.T) {
	// When LocalPath is empty, HF_TOKEN should be passed
	specWithoutLocal := ModelSpec{
		Alias:  "sglang-model",
		HFRepo: "Qwen/Qwen2-7B-Instruct",
	}

	req := BuildSGLangRequest(specWithoutLocal, "/data/models", "hf_secret_token")
	if req.Env["HF_TOKEN"] != "hf_secret_token" {
		t.Error("HF_TOKEN should be passed when LocalPath is empty")
	}

	// When LocalPath is set, HF_TOKEN should NOT be passed (security isolation)
	specWithLocal := ModelSpec{
		Alias:     "sglang-model",
		HFRepo:    "Qwen/Qwen2-7B-Instruct",
		LocalPath: "/data/models/Qwen/Qwen2-7B-Instruct",
	}

	req = BuildSGLangRequest(specWithLocal, "/data/models", "hf_secret_token")
	if _, exists := req.Env["HF_TOKEN"]; exists {
		t.Error("HF_TOKEN should NOT be passed when LocalPath is set (security isolation)")
	}
}

func TestBuildTGIRequest_HFTokenIsolation(t *testing.T) {
	// When LocalPath is empty, token should be passed
	specWithoutLocal := ModelSpec{
		Alias:  "tgi-model",
		HFRepo: "mistralai/Mistral-7B-Instruct-v0.3",
	}

	req := BuildTGIRequest(specWithoutLocal, "/data/models", "hf_secret")
	if req.Env["HUGGINGFACE_HUB_TOKEN"] != "hf_secret" {
		t.Error("HUGGINGFACE_HUB_TOKEN should be passed when LocalPath is empty")
	}

	// When LocalPath is set, token should NOT be passed
	specWithLocal := ModelSpec{
		Alias:     "tgi-model",
		HFRepo:    "mistralai/Mistral-7B-Instruct-v0.3",
		LocalPath: "/data/models/mistralai/Mistral-7B-Instruct-v0.3",
	}

	req = BuildTGIRequest(specWithLocal, "/data/models", "hf_secret")
	if _, exists := req.Env["HUGGINGFACE_HUB_TOKEN"]; exists {
		t.Error("HUGGINGFACE_HUB_TOKEN should NOT be passed when LocalPath is set")
	}
}

func TestBuildTRTLLMRequest_NoHFToken(t *testing.T) {
	spec := ModelSpec{
		Alias:     "trt-model",
		LocalPath: "/data/engines/trt/llama3-8b",
	}

	req, err := BuildTRTLLMRequest(spec, "/data/engines/trt", "hf_secret_token")
	if err != nil {
		t.Fatalf("BuildTRTLLMRequest failed: %v", err)
	}

	// TRT-LLM should NEVER have HF token (requires pre-converted engines)
	if _, exists := req.Env["HUGGINGFACE_HUB_TOKEN"]; exists {
		t.Error("TRT-LLM should never receive HF_TOKEN")
	}
	if _, exists := req.Env["HF_TOKEN"]; exists {
		t.Error("TRT-LLM should never receive HF_TOKEN")
	}
}

func TestBuildTRTLLMRequest_RequiresLocalPath(t *testing.T) {
	spec := ModelSpec{
		Alias:  "trt-model",
		HFRepo: "meta-llama/Llama-3.1-8B",
		// LocalPath is empty
	}

	_, err := BuildTRTLLMRequest(spec, "/data/engines/trt", "")
	if err == nil {
		t.Error("BuildTRTLLMRequest should fail when LocalPath is empty")
	}
}

func TestBuildLlamaCPPRequest(t *testing.T) {
	spec := ModelSpec{
		Alias:           "llama-q4",
		LocalPath:       "/data/gguf/llama-3.1-8b.Q4_K_M.gguf",
		LlamaNGPULayers: 99,
		LlamaMainGPU:    0,
	}

	req, err := BuildLlamaCPPRequest(spec)
	if err != nil {
		t.Fatalf("BuildLlamaCPPRequest failed: %v", err)
	}

	if req.Provider != ProviderLlamaCPP {
		t.Errorf("Provider = %v, want %v", req.Provider, ProviderLlamaCPP)
	}

	// Check n-gpu-layers
	found := false
	for i, c := range req.Command {
		if c == "--n-gpu-layers" && i+1 < len(req.Command) && req.Command[i+1] == "99" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Command missing --n-gpu-layers 99")
	}

	// Check mount exists
	if len(req.Mounts) == 0 {
		t.Error("Mounts should not be empty")
	}
}

func TestBuildLlamaCPPRequest_RequiresLocalPath(t *testing.T) {
	spec := ModelSpec{
		Alias: "llama-q4",
		// LocalPath is empty
	}

	_, err := BuildLlamaCPPRequest(spec)
	if err == nil {
		t.Error("BuildLlamaCPPRequest should fail when LocalPath is empty")
	}
}

func TestBuildLlamaCPPRequest_TensorSplit(t *testing.T) {
	spec := ModelSpec{
		Alias:            "llama-70b-q4",
		LocalPath:        "/data/gguf/llama-70b.Q4_K_M.gguf",
		LlamaNGPULayers:  99,
		LlamaTensorSplit: "0.5,0.5", // Split across 2 GPUs
	}

	req, err := BuildLlamaCPPRequest(spec)
	if err != nil {
		t.Fatalf("BuildLlamaCPPRequest failed: %v", err)
	}

	// Check tensor-split
	found := false
	for i, c := range req.Command {
		if c == "--tensor-split" && i+1 < len(req.Command) && req.Command[i+1] == "0.5,0.5" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Command missing --tensor-split 0.5,0.5")
	}
}

func TestBuildLlamaCPPRequest_CacheReuse(t *testing.T) {
	tests := []struct {
		name        string
		cacheReuse  int
		expectFlag  bool
		expectValue string
	}{
		{"zero omits flag", 0, false, ""},
		{"positive value sets flag", 128, true, "128"},
		{"negative one disables", -1, true, "-1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := ModelSpec{
				Alias:           "test-llama",
				LocalPath:       "/data/gguf/test.gguf",
				LlamaCacheReuse: tt.cacheReuse,
			}

			req, err := BuildLlamaCPPRequest(spec)
			if err != nil {
				t.Fatalf("BuildLlamaCPPRequest failed: %v", err)
			}

			found := false
			for i, c := range req.Command {
				if c == "--cache-reuse" && i+1 < len(req.Command) {
					found = true
					if req.Command[i+1] != tt.expectValue {
						t.Errorf("--cache-reuse value = %q, want %q", req.Command[i+1], tt.expectValue)
					}
					break
				}
			}
			if tt.expectFlag && !found {
				t.Errorf("expected --cache-reuse flag but not found in %v", req.Command)
			}
			if !tt.expectFlag && found {
				t.Errorf("did not expect --cache-reuse flag but found it in %v", req.Command)
			}
		})
	}
}

func TestBuildLlamaCPPRequest_ExtraArgs(t *testing.T) {
	spec := ModelSpec{
		Alias:          "test-llama",
		LocalPath:      "/data/gguf/test.gguf",
		LlamaExtraArgs: "--no-mmap --verbose",
	}

	req, err := BuildLlamaCPPRequest(spec)
	if err != nil {
		t.Fatalf("BuildLlamaCPPRequest failed: %v", err)
	}

	foundNoMmap := false
	foundVerbose := false
	for _, c := range req.Command {
		if c == "--no-mmap" {
			foundNoMmap = true
		}
		if c == "--verbose" {
			foundVerbose = true
		}
	}
	if !foundNoMmap {
		t.Error("Command missing --no-mmap from extra_args")
	}
	if !foundVerbose {
		t.Error("Command missing --verbose from extra_args")
	}
}

