package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"

	"aigateway/internal/agent"
	"aigateway/internal/inference"
	"aigateway/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// WorkerHandler handles admin API for managing remote inference workers.
type WorkerHandler struct {
	manager         *agent.Manager
	inferenceRouter *inference.Router // for Docker image access
	logger          *logrus.Logger
}

// NewWorkerHandler creates a new WorkerHandler.
func NewWorkerHandler(manager *agent.Manager, logger *logrus.Logger) *WorkerHandler {
	return &WorkerHandler{
		manager: manager,
		logger:  logger,
	}
}

// SetInferenceRouter sets the inference router for Docker image operations.
func (h *WorkerHandler) SetInferenceRouter(router *inference.Router) {
	h.inferenceRouter = router
}

// ListWorkers returns all registered worker nodes.
// GET /api/admin/workers
func (h *WorkerHandler) ListWorkers(c *gin.Context) {
	nodes, err := h.manager.ListNodes(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if nodes == nil {
		nodes = []*models.WorkerNode{}
	}
	c.JSON(http.StatusOK, gin.H{"workers": nodes})
}

// GetWorker returns a single worker node.
// GET /api/admin/workers/:id
func (h *WorkerHandler) GetWorker(c *gin.Context) {
	id := c.Param("id")
	node, err := h.manager.GetNode(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, node)
}

// CreateWorker registers a new worker node.
// POST /api/admin/workers
func (h *WorkerHandler) CreateWorker(c *gin.Context) {
	var req struct {
		Name             string `json:"name" binding:"required"`
		Address          string `json:"address" binding:"required"`
		APIKey           string `json:"api_key"` // optional — auto-generated if empty
		NodeType         string `json:"node_type"`
		MaxRunningModels int    `json:"max_running_models"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	// Validate address is a proper URL
	if u, parseErr := url.Parse(req.Address); parseErr != nil || u.Host == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid address: must be a valid URL (e.g. https://gpu-server:9090)"})
		return
	}

	// Validate node type
	if req.NodeType != "" && req.NodeType != "gpu" && req.NodeType != "cpu" && req.NodeType != "mixed" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "node_type must be gpu, cpu, or mixed"})
		return
	}

	// Generate API key if not provided
	apiKey := req.APIKey
	if apiKey == "" {
		keyBytes := make([]byte, 32)
		if _, err := rand.Read(keyBytes); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate API key"})
			return
		}
		apiKey = "agent-" + hex.EncodeToString(keyBytes)
	}

	if req.NodeType == "" {
		req.NodeType = "gpu"
	}
	if req.MaxRunningModels <= 0 {
		req.MaxRunningModels = 2
	}

	node := &models.WorkerNode{
		Name:             req.Name,
		Address:          req.Address,
		APIKey:           apiKey,
		NodeType:         req.NodeType,
		Status:           "pending",
		MaxRunningModels: req.MaxRunningModels,
	}

	if err := h.manager.RegisterNode(c.Request.Context(), node); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"node_id":   node.ID,
		"node_name": node.Name,
		"address":   node.Address,
	}).Info("Worker node registered via admin API")

	// Return node with API key visible (only on creation)
	c.JSON(http.StatusCreated, gin.H{
		"worker":  node,
		"api_key": apiKey, // Show API key only once
	})
}

// UpdateWorker updates worker node configuration.
// PUT /api/admin/workers/:id
func (h *WorkerHandler) UpdateWorker(c *gin.Context) {
	id := c.Param("id")

	existing, err := h.manager.GetNode(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	var req struct {
		Name             *string `json:"name"`
		Address          *string `json:"address"`
		NodeType         *string `json:"node_type"`
		MaxRunningModels *int    `json:"max_running_models"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Address != nil {
		existing.Address = *req.Address
	}
	if req.NodeType != nil {
		existing.NodeType = *req.NodeType
	}
	if req.MaxRunningModels != nil {
		existing.MaxRunningModels = *req.MaxRunningModels
	}

	if err := h.manager.UpdateNode(c.Request.Context(), existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// DeleteWorker removes a worker node.
// DELETE /api/admin/workers/:id
func (h *WorkerHandler) DeleteWorker(c *gin.Context) {
	id := c.Param("id")
	if err := h.manager.RemoveNode(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// SetWorkerStatus updates the status of a worker node (online/draining).
// PUT /api/admin/workers/:id/status
func (h *WorkerHandler) SetWorkerStatus(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	if req.Status != "online" && req.Status != "draining" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status must be 'online' or 'draining'"})
		return
	}

	node, err := h.manager.GetNode(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	node.Status = req.Status
	if err := h.manager.UpdateNode(c.Request.Context(), node); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": node.Status})
}

// GetWorkerGPU returns live GPU metrics from a worker.
// GET /api/admin/workers/:id/gpu
func (h *WorkerHandler) GetWorkerGPU(c *gin.Context) {
	id := c.Param("id")

	client, ok := h.manager.GetClient(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "worker not found"})
		return
	}

	sysInfo, err := client.SystemInfo(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("agent unreachable: %v", err)})
		return
	}

	c.JSON(http.StatusOK, sysInfo)
}

// ListWorkerModels returns models running on a specific worker.
// GET /api/admin/workers/:id/models
func (h *WorkerHandler) ListWorkerModels(c *gin.Context) {
	id := c.Param("id")

	client, ok := h.manager.GetClient(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "worker not found"})
		return
	}

	models, err := client.ListModels(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("agent unreachable: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"models": models})
}

// LoadModelOnWorker loads a model on a specific worker.
// POST /api/admin/workers/:id/models/load
func (h *WorkerHandler) LoadModelOnWorker(c *gin.Context) {
	id := c.Param("id")

	var spec inference.ModelSpec
	if err := c.ShouldBindJSON(&spec); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid model spec: %v", err)})
		return
	}
	if spec.Alias == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "alias is required"})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"node_id": id,
		"alias":   spec.Alias,
	}).Info("Loading model on worker via admin API")

	resp, err := h.manager.LoadModelOnNode(c.Request.Context(), id, spec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// StopModelOnWorker stops a model on a specific worker.
// POST /api/admin/workers/:id/models/stop
func (h *WorkerHandler) StopModelOnWorker(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		Alias string `json:"alias" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	if err := h.manager.StopModelOnNode(c.Request.Context(), id, req.Alias); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "stopped", "alias": req.Alias})
}

// GenerateConfig generates an agent config YAML and API key for a new worker.
// POST /api/admin/workers/generate-config
func (h *WorkerHandler) GenerateConfig(c *gin.Context) {
	var req agent.GenerateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	// Generate API key
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate API key"})
		return
	}
	apiKey := "agent-" + hex.EncodeToString(keyBytes)

	listenAddr := req.ListenAddr
	if listenAddr == "" {
		listenAddr = "0.0.0.0:9090"
	}
	hfCacheDir := req.HFCacheDir
	if hfCacheDir == "" {
		hfCacheDir = "./data/models/hf"
	}
	ggufCacheDir := req.GGUFCacheDir
	if ggufCacheDir == "" {
		ggufCacheDir = "./data/models/gguf"
	}
	nodeType := string(req.NodeType)
	if nodeType == "" {
		nodeType = "gpu"
	}

	// Generate YAML config
	configYAML := fmt.Sprintf(`# AIGateway Agent Configuration — %s
# Generated automatically. Do not edit the api_key.

agent:
  listen_addr: "%s"
  api_key: "%s"
  node_name: "%s"
  node_type: "%s"
  hf_cache_dir: "%s"
  gguf_cache_dir: "%s"
  hf_token: "%s"
  docker_bin: "docker"
  max_running_models: 2
  default_provider: "vllm"
  health_check_timeout: "60s"
  startup_timeout: "10m"
  tls:
    enabled: false
    cert_file: "certs/agent.crt"
    key_file: "certs/agent.key"
`, req.Name, listenAddr, apiKey, req.Name, nodeType, hfCacheDir, ggufCacheDir, req.HFToken)

	installCmd := "curl -fsSL https://gitlab.alexue4.dev/api/v4/projects/146/packages/generic/aigateway-agent/latest/install.sh | sudo bash"
	postInstall := fmt.Sprintf("sudo cp agent.yaml /opt/aigateway-agent/configs/agent.yaml && sudo systemctl restart aigateway-agent")

	c.JSON(http.StatusOK, agent.GenerateConfigResponse{
		ConfigYAML:     configYAML,
		APIKey:         apiKey,
		InstallCommand: installCmd,
		PostInstall:    postInstall,
	})
}

// PushImageToWorker pushes a local Docker image to a specific worker.
// Uses docker save on main server → stream → docker load on agent.
// POST /api/admin/workers/:id/images/push
func (h *WorkerHandler) PushImageToWorker(c *gin.Context) {
	nodeID := c.Param("id")

	var req struct {
		Image string `json:"image" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("invalid request: %v", err)})
		return
	}

	// Get agent client
	agentClient, ok := h.manager.GetClient(nodeID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "worker not found"})
		return
	}

	// Get Docker runtime for local image save
	if h.inferenceRouter == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "inference system not available"})
		return
	}
	runtime := h.inferenceRouter.GetRuntime()
	if runtime == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Docker runtime not available"})
		return
	}

	// Check image exists locally
	exists, size := runtime.ImageExists(req.Image)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("image %q not found locally", req.Image)})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"node_id": nodeID,
		"image":   req.Image,
		"size":    size,
	}).Info("Pushing Docker image to worker (docker save → stream → docker load)")

	// docker save → stream → agent docker load
	imageReader, err := runtime.SaveImage(c.Request.Context(), req.Image)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("docker save failed: %v", err)})
		return
	}
	defer imageReader.Close()

	if err := agentClient.PushImage(c.Request.Context(), req.Image, imageReader); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("image push to agent failed: %v", err)})
		return
	}

	h.logger.WithFields(logrus.Fields{
		"node_id": nodeID,
		"image":   req.Image,
	}).Info("Docker image pushed to worker successfully")

	c.JSON(http.StatusOK, gin.H{
		"status": "pushed",
		"image":  req.Image,
		"size":   size,
	})
}

// ListWorkerImages returns Docker images available on a specific worker.
// GET /api/admin/workers/:id/images
func (h *WorkerHandler) ListWorkerImages(c *gin.Context) {
	nodeID := c.Param("id")

	client, ok := h.manager.GetClient(nodeID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "worker not found"})
		return
	}

	images, err := client.ListImages(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": fmt.Sprintf("agent unreachable: %v", err)})
		return
	}

	c.JSON(http.StatusOK, gin.H{"images": images})
}

// GetWorkerModelsForUI returns all models from all workers (for Chat UI /v1/models merge).
func (h *WorkerHandler) GetAllRemoteModels() []*agent.RemoteModel {
	if h.manager == nil {
		return nil
	}
	return h.manager.ListAllModels()
}
