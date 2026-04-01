package agent

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"aigateway/internal/inference"
	"aigateway/internal/version"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// rewriteModelField replaces the "model" value in a JSON body without touching other fields.
// Uses map[string]json.RawMessage to preserve all original fields.
func rewriteModelField(body []byte, newModel string) []byte {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return body
	}
	modelJSON, err := json.Marshal(newModel)
	if err != nil {
		return body
	}
	raw["model"] = modelJSON
	out, err := json.Marshal(raw)
	if err != nil {
		return body
	}
	return out
}

// ──────────────────────────────────────────────────────────────
// Health & System Info
// ──────────────────────────────────────────────────────────────

func (s *Server) handleHealth(c *gin.Context) {
	models := s.infRouter.ListModels()
	running := 0
	for _, m := range models {
		if m.Status == inference.StatusRunning {
			running++
		}
	}
	c.JSON(http.StatusOK, AgentHealthResponse{
		Status:        "ok",
		NodeName:      s.config.NodeName,
		NodeType:      s.config.NodeType,
		UptimeSeconds: s.uptimeSeconds(),
		ModelsRunning: running,
		Version:       version.Version,
	})
}

func (s *Server) handleSystemInfo(c *gin.Context) {
	// GPU info from nvidia-smi (reuse metrics package if available)
	info := AgentSystemInfo{
		GPUDevices: collectGPUDevices(s.logger),
		CPU:        collectCPUInfo(),
		Memory:     collectMemoryInfo(),
	}
	c.JSON(http.StatusOK, info)
}

// ──────────────────────────────────────────────────────────────
// Model Lifecycle
// ──────────────────────────────────────────────────────────────

func (s *Server) handleListModels(c *gin.Context) {
	models := s.infRouter.ListModels()
	result := make([]AgentModelStatus, 0, len(models))
	for _, m := range models {
		st := AgentModelStatus{
			Alias:    m.Spec.Alias,
			Status:   m.Status,
			Provider: m.Spec.Provider,
			Error:    m.Error,
		}
		if m.Handle != nil {
			st.Endpoint = m.Handle.Endpoint
		} else if m.Endpoint != "" {
			st.Endpoint = m.Endpoint
		}
		if !m.StartedAt.IsZero() {
			t := m.StartedAt
			st.StartedAt = &t
		}
		result = append(result, st)
	}
	c.JSON(http.StatusOK, result)
}

func (s *Server) handleLoadModel(c *gin.Context) {
	var req LoadModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	spec := req.ModelSpec
	if spec.Alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}
	if spec.Provider == "" {
		spec.Provider = inference.ProviderKind(s.config.DefaultProvider)
	}

	s.logger.WithFields(logrus.Fields{
		"alias":    spec.Alias,
		"provider": spec.Provider,
		"hf_repo":  spec.HFRepo,
	}).Info("Loading model on agent")

	// Register spec and start via the inference router.
	// EnsureBySpec calls service.LoadAndStart internally, which builds
	// the container start request using the same provider builders.
	s.infRouter.RegisterSpec(spec)
	inst, err := s.infRouter.EnsureBySpec(c.Request.Context(), spec)
	if err != nil {
		s.logger.WithError(err).WithField("alias", spec.Alias).Error("Failed to load model")
		c.JSON(http.StatusInternalServerError, LoadModelResponse{
			Alias:  spec.Alias,
			Status: inference.StatusFailed,
			Error:  err.Error(),
		})
		return
	}

	endpoint := ""
	if inst.Handle != nil {
		endpoint = inst.Handle.Endpoint
	}

	c.JSON(http.StatusOK, LoadModelResponse{
		Alias:    spec.Alias,
		Status:   inst.Status,
		Endpoint: endpoint,
	})
}

func (s *Server) handleStopModel(c *gin.Context) {
	var req StopModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}
	if req.Alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}

	s.logger.WithField("alias", req.Alias).Info("Stopping model on agent")
	if err := s.infRouter.Stop(c.Request.Context(), req.Alias); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "stopped", "alias": req.Alias})
}

func (s *Server) handleEvictModel(c *gin.Context) {
	alias := c.Param("alias")
	if alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}

	s.logger.WithField("alias", alias).Info("Evicting model on agent")
	if err := s.infRouter.Evict(c.Request.Context(), alias); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "evicted", "alias": alias})
}

func (s *Server) handleModelLogs(c *gin.Context) {
	alias := c.Param("alias")
	tailStr := c.DefaultQuery("tail", "100")
	tail, _ := strconv.Atoi(tailStr)
	if tail <= 0 {
		tail = 100
	}

	logs, err := s.infRouter.ContainerLogs(c.Request.Context(), alias, tail)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"alias": alias, "logs": logs})
}

// ──────────────────────────────────────────────────────────────
// Inference Proxy — forwards to local container
// ──────────────────────────────────────────────────────────────

func (s *Server) handleInferenceProxy(c *gin.Context) {
	// Read raw body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}

	// Extract model alias from body
	var partial struct {
		Model  string `json:"model"`
		Stream *bool  `json:"stream,omitempty"`
	}
	if err := json.Unmarshal(body, &partial); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
		return
	}
	if partial.Model == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "model field is required"})
		return
	}

	// Resolve model to local endpoint
	endpoint, running := s.infRouter.GetModel(partial.Model)
	if !running || endpoint == "" {
		// Try to ensure model is running
		inst, ensureErr := s.infRouter.EnsureByAlias(c.Request.Context(), partial.Model)
		if ensureErr != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("model not available: %s: %v", partial.Model, ensureErr)})
			return
		}
		if inst.Handle != nil {
			endpoint = inst.Handle.Endpoint
		}
		if endpoint == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": fmt.Sprintf("model %s has no endpoint", partial.Model)})
			return
		}
	}

	// Rewrite model field to provider-expected name (e.g. alias → HFRepo)
	inst := s.infRouter.GetModelInstance(partial.Model)
	if inst != nil {
		providerModel := resolveProviderModelName(inst.Spec)
		if providerModel != partial.Model {
			body = rewriteModelField(body, providerModel)
		}
	}

	// Note: Anthropic message normalization is NOT needed here —
	// the main server already normalizes before forwarding to the agent.

	// Validate endpoint URL
	if endpoint == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "model endpoint not available"})
		return
	}
	if _, parseErr := url.Parse(endpoint); parseErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("invalid model endpoint: %v", parseErr)})
		return
	}

	// Determine target URL
	path := c.Request.URL.Path // e.g. /v1/chat/completions
	targetURL := endpoint + path

	// Forward request to container
	proxyReq, err := http.NewRequestWithContext(c.Request.Context(), http.MethodPost, targetURL, io.NopCloser(bytes.NewReader(body)))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create proxy request"})
		return
	}
	proxyReq.ContentLength = int64(len(body))
	proxyReq.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{Timeout: 10 * time.Minute}
	resp, err := httpClient.Do(proxyReq)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("proxy to container failed: %v", err)})
		return
	}
	defer resp.Body.Close()

	// Stream or forward response — only stream on HTTP 200 success
	isStream := partial.Stream != nil && *partial.Stream
	if isStream && resp.StatusCode == http.StatusOK {
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Status(http.StatusOK)

		flusher, ok := c.Writer.(http.Flusher)
		buf := make([]byte, 4096)
		for {
			n, readErr := resp.Body.Read(buf)
			if n > 0 {
				if _, writeErr := c.Writer.Write(buf[:n]); writeErr != nil {
					// Client disconnected
					s.logger.WithField("alias", partial.Model).Debug("Client disconnected during streaming")
					return
				}
				if ok {
					flusher.Flush()
				}
			}
			if readErr != nil {
				break
			}
		}
		return
	}

	// Non-streaming or error: forward as-is
	c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
}

// ──────────────────────────────────────────────────────────────
// Docker Image Management
// ──────────────────────────────────────────────────────────────

// handleImagePush receives a Docker image tar stream from the main server
// and loads it into the local Docker daemon.
// POST /api/agent/images/push?image=<image:tag>
func (s *Server) handleImagePush(c *gin.Context) {
	imageName := c.Query("image")
	if imageName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "image query parameter is required"})
		return
	}

	runtime := s.infRouter.GetRuntime()
	if runtime == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Docker runtime not available"})
		return
	}

	s.logger.WithField("image", imageName).Info("Receiving Docker image push from main server")

	// Load image from request body (tar stream)
	if err := runtime.LoadImage(c.Request.Context(), c.Request.Body); err != nil {
		s.logger.WithError(err).WithField("image", imageName).Error("Failed to load Docker image")
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("docker load failed: %v", err)})
		return
	}

	// Verify the image exists after loading
	exists, size := runtime.ImageExists(imageName)
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "image loaded but not found — tag mismatch?"})
		return
	}

	s.logger.WithFields(logrus.Fields{
		"image": imageName,
		"size":  size,
	}).Info("Docker image received and loaded successfully")

	c.JSON(http.StatusOK, gin.H{
		"status": "loaded",
		"image":  imageName,
		"size":   size,
	})
}

// handleListImages returns all Docker images on the agent that match inference providers.
// GET /api/agent/images
func (s *Server) handleListImages(c *gin.Context) {
	runtime := s.infRouter.GetRuntime()
	if runtime == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Docker runtime not available"})
		return
	}

	// Check each default provider image + any custom images from running models
	defaultImages := []struct {
		Provider string
		Image    string
	}{
		{"vllm", inference.DefaultVLLMImage},
		{"sglang", inference.DefaultSGLangImage},
		{"tgi", inference.DefaultTGIImage},
		{"tei", inference.DefaultTEIImage},
		{"tei-cpu", inference.DefaultTEICPUImage},
		{"llama.cpp", inference.DefaultLlamaImage},
		{"tensorrt-llm", inference.DefaultTRTLLMImage},
	}

	type imageInfo struct {
		Provider string `json:"provider"`
		Image    string `json:"image"`
		Exists   bool   `json:"exists"`
		Size     string `json:"size,omitempty"`
	}

	var images []imageInfo
	for _, di := range defaultImages {
		exists, size := runtime.ImageExists(di.Image)
		images = append(images, imageInfo{
			Provider: di.Provider,
			Image:    di.Image,
			Exists:   exists,
			Size:     size,
		})
	}

	// Also check custom images from running models
	for _, m := range s.infRouter.ListModels() {
		if m.Spec.DockerImage != "" {
			exists, size := runtime.ImageExists(m.Spec.DockerImage)
			images = append(images, imageInfo{
				Provider: string(m.Spec.Provider) + " (custom)",
				Image:    m.Spec.DockerImage,
				Exists:   exists,
				Size:     size,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"images": images})
}
