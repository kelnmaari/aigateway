package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"aigateway/internal/inference"
	"aigateway/internal/models"
	"aigateway/internal/storage"

	"github.com/sirupsen/logrus"
)

// Manager tracks all registered agent nodes, their health, and running models.
// It runs on the main server and communicates with remote agents via HTTP.
type Manager struct {
	mu      sync.RWMutex
	nodes   map[string]*WorkerNode      // nodeID -> live state
	clients map[string]*Client          // nodeID -> HTTP client
	models  map[string]*RemoteModel     // alias -> running model info

	db     storage.Database
	logger *logrus.Logger

	// Per-node locks to prevent concurrent health checks for the same node
	nodeLocksMu         sync.Mutex
	nodeLocks           map[string]*sync.Mutex
	consecutiveFailures map[string]int // nodeID -> failure count (guarded by nodeLock)

	// Lifecycle context for background goroutines
	globalCtx    context.Context
	globalCancel context.CancelFunc

	// Config
	healthInterval       time.Duration
	healthTimeout        time.Duration
	maxConsecFailures    int
	tlsSkipVerify        bool
	caCertPath           string
}

// RemoteModel tracks a model running on a remote agent.
type RemoteModel struct {
	Alias    string
	Provider inference.ProviderKind
	HFRepo   string // populated from agent's model list for correct name rewriting
	NodeID   string
	NodeName string
	NodeAddr string // https://agent:9090
	Status   inference.ModelStatus
}

// ManagerConfig holds configuration for the agent manager.
type ManagerConfig struct {
	HealthCheckInterval  time.Duration
	HealthCheckTimeout   time.Duration
	MaxConsecFailures    int
	TLSSkipVerify        bool
	CACertPath           string
}

// NewManager creates an agent manager.
func NewManager(db storage.Database, logger *logrus.Logger, cfg ManagerConfig) *Manager {
	if cfg.HealthCheckInterval <= 0 {
		cfg.HealthCheckInterval = 30 * time.Second
	}
	if cfg.HealthCheckTimeout <= 0 {
		cfg.HealthCheckTimeout = 10 * time.Second
	}
	if cfg.MaxConsecFailures <= 0 {
		cfg.MaxConsecFailures = 3
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		nodes:               make(map[string]*WorkerNode),
		clients:             make(map[string]*Client),
		models:              make(map[string]*RemoteModel),
		nodeLocks:           make(map[string]*sync.Mutex),
		consecutiveFailures: make(map[string]int),
		db:                  db,
		logger:              logger,
		globalCtx:           ctx,
		globalCancel:        cancel,
		healthInterval:      cfg.HealthCheckInterval,
		healthTimeout:       cfg.HealthCheckTimeout,
		maxConsecFailures:   cfg.MaxConsecFailures,
		tlsSkipVerify:       cfg.TLSSkipVerify,
		caCertPath:          cfg.CACertPath,
	}
}

// ──────────────────────────────────────────────────────────────
// Node Registration
// ──────────────────────────────────────────────────────────────

// RegisterNode adds a worker node and persists it to DB.
func (m *Manager) RegisterNode(ctx context.Context, node *models.WorkerNode) error {
	// Save to DB
	if err := m.db.CreateWorkerNode(ctx, node); err != nil {
		return fmt.Errorf("register worker node: %w", err)
	}

	// Create client and cache in memory
	client := m.createClient(node)

	m.mu.Lock()
	m.nodes[node.ID] = m.dbNodeToLive(node)
	m.clients[node.ID] = client
	m.mu.Unlock()

	m.logger.WithFields(logrus.Fields{
		"node_id":   node.ID,
		"node_name": node.Name,
		"address":   node.Address,
		"node_type": node.NodeType,
	}).Info("Worker node registered")

	// Run initial health check in background (use manager's lifecycle context)
	go m.checkNodeHealth(m.globalCtx, node.ID)

	return nil
}

// getNodeLock returns or creates a per-node mutex.
func (m *Manager) getNodeLock(nodeID string) *sync.Mutex {
	m.nodeLocksMu.Lock()
	defer m.nodeLocksMu.Unlock()
	if lock, ok := m.nodeLocks[nodeID]; ok {
		return lock
	}
	lock := &sync.Mutex{}
	m.nodeLocks[nodeID] = lock
	return lock
}

// RemoveNode removes a worker node from DB and memory.
func (m *Manager) RemoveNode(ctx context.Context, nodeID string) error {
	// Remove models from this node
	m.mu.Lock()
	for alias, rm := range m.models {
		if rm.NodeID == nodeID {
			delete(m.models, alias)
		}
	}
	delete(m.nodes, nodeID)
	delete(m.clients, nodeID)
	m.mu.Unlock()

	if err := m.db.DeleteWorkerNode(ctx, nodeID); err != nil {
		return fmt.Errorf("remove worker node: %w", err)
	}

	m.logger.WithField("node_id", nodeID).Info("Worker node removed")
	return nil
}

// UpdateNode updates worker node config in DB and refreshes client.
func (m *Manager) UpdateNode(ctx context.Context, node *models.WorkerNode) error {
	if err := m.db.UpdateWorkerNode(ctx, node); err != nil {
		return err
	}

	client := m.createClient(node)

	m.mu.Lock()
	m.nodes[node.ID] = m.dbNodeToLive(node)
	m.clients[node.ID] = client
	m.mu.Unlock()

	return nil
}

// ListNodes returns all registered worker nodes from DB.
func (m *Manager) ListNodes(ctx context.Context) ([]*models.WorkerNode, error) {
	return m.db.ListWorkerNodes(ctx)
}

// GetNode returns a single node from DB.
func (m *Manager) GetNode(ctx context.Context, nodeID string) (*models.WorkerNode, error) {
	return m.db.GetWorkerNode(ctx, nodeID)
}

// GetClient returns the HTTP client for a node.
func (m *Manager) GetClient(nodeID string) (*Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.clients[nodeID]
	return c, ok
}

// ──────────────────────────────────────────────────────────────
// Model Discovery & Resolution
// ──────────────────────────────────────────────────────────────

// FindRunningModel looks for a model running on any agent.
// Returns the RemoteModel and the node address if found.
func (m *Manager) FindRunningModel(ctx context.Context, alias string) (*RemoteModel, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	rm, ok := m.models[alias]
	if !ok || rm.Status != inference.StatusRunning {
		return nil, false
	}
	return rm, true
}

// ListAllModels returns all models tracked across all agents.
func (m *Manager) ListAllModels() []*RemoteModel {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*RemoteModel, 0, len(m.models))
	for _, rm := range m.models {
		result = append(result, rm)
	}
	return result
}

// LoadModelOnNode sends a model load request to a specific agent node.
func (m *Manager) LoadModelOnNode(ctx context.Context, nodeID string, spec inference.ModelSpec) (*LoadModelResponse, error) {
	client, ok := m.GetClient(nodeID)
	if !ok {
		return nil, fmt.Errorf("worker node not found: %s", nodeID)
	}

	m.mu.RLock()
	node, nodeOk := m.nodes[nodeID]
	m.mu.RUnlock()
	if !nodeOk {
		return nil, fmt.Errorf("worker node not in memory: %s", nodeID)
	}

	resp, err := client.LoadModel(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("load model on %s: %w", node.Name, err)
	}

	// Track the model if it started successfully
	if resp.Status == inference.StatusRunning || resp.Status == inference.StatusStarting {
		m.mu.Lock()
		m.models[spec.Alias] = &RemoteModel{
			Alias:    spec.Alias,
			Provider: spec.Provider,
			NodeID:   nodeID,
			NodeName: node.Name,
			NodeAddr: node.Address,
			Status:   resp.Status,
		}
		m.mu.Unlock()
	}

	// Refresh node's running models
	go m.syncNodeModels(context.Background(), nodeID)

	return resp, nil
}

// StopModelOnNode sends a model stop request to a specific agent node.
func (m *Manager) StopModelOnNode(ctx context.Context, nodeID string, alias string) error {
	client, ok := m.GetClient(nodeID)
	if !ok {
		return fmt.Errorf("worker node not found: %s", nodeID)
	}

	if err := client.StopModel(ctx, alias); err != nil {
		return err
	}

	m.mu.Lock()
	delete(m.models, alias)
	m.mu.Unlock()

	go m.syncNodeModels(context.Background(), nodeID)
	return nil
}

// EvictModelOnNode evicts a model from a specific agent node.
func (m *Manager) EvictModelOnNode(ctx context.Context, nodeID string, alias string) error {
	client, ok := m.GetClient(nodeID)
	if !ok {
		return fmt.Errorf("worker node not found: %s", nodeID)
	}

	if err := client.EvictModel(ctx, alias); err != nil {
		return err
	}

	m.mu.Lock()
	delete(m.models, alias)
	m.mu.Unlock()

	go m.syncNodeModels(context.Background(), nodeID)
	return nil
}

// ──────────────────────────────────────────────────────────────
// Health Check Loop
// ──────────────────────────────────────────────────────────────

// StartHealthChecks begins periodic health checks for all registered nodes.
func (m *Manager) StartHealthChecks(ctx context.Context) {
	go func() {
		// Load nodes from DB on startup
		m.loadNodesFromDB(ctx)

		ticker := time.NewTicker(m.healthInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.checkAllNodes(ctx)
			}
		}
	}()
}

// loadNodesFromDB loads all worker nodes from database and creates clients.
func (m *Manager) loadNodesFromDB(ctx context.Context) {
	nodes, err := m.db.ListWorkerNodes(ctx)
	if err != nil {
		m.logger.WithError(err).Error("Failed to load worker nodes from DB")
		return
	}

	m.mu.Lock()
	for _, node := range nodes {
		m.nodes[node.ID] = m.dbNodeToLive(node)
		m.clients[node.ID] = m.createClient(node)
	}
	m.mu.Unlock()

	m.logger.WithField("count", len(nodes)).Info("Loaded worker nodes from database")

	// Initial health check + model sync for all nodes
	for _, node := range nodes {
		go m.checkNodeHealth(ctx, node.ID)
	}
}

// checkAllNodes runs health checks for every registered node.
func (m *Manager) checkAllNodes(ctx context.Context) {
	m.mu.RLock()
	nodeIDs := make([]string, 0, len(m.nodes))
	for id := range m.nodes {
		nodeIDs = append(nodeIDs, id)
	}
	m.mu.RUnlock()

	for _, id := range nodeIDs {
		m.checkNodeHealth(ctx, id)
	}
}

// checkNodeHealth performs health check for a single node.
// Uses per-node lock to prevent concurrent checks for the same node.
func (m *Manager) checkNodeHealth(ctx context.Context, nodeID string) {
	// Per-node lock prevents concurrent health checks racing
	nodeLock := m.getNodeLock(nodeID)
	nodeLock.Lock()
	defer nodeLock.Unlock()

	m.mu.RLock()
	client, clientOk := m.clients[nodeID]
	node, nodeOk := m.nodes[nodeID]
	m.mu.RUnlock()
	if !clientOk || !nodeOk {
		return
	}

	healthCtx, cancel := context.WithTimeout(ctx, m.healthTimeout)
	defer cancel()

	_, err := client.Health(healthCtx)
	if err != nil {
		// Increment failure count (safe — guarded by nodeLock)
		m.consecutiveFailures[nodeID]++
		failures := m.consecutiveFailures[nodeID]

		m.logger.WithFields(logrus.Fields{
			"node_id":            nodeID,
			"node_name":          node.Name,
			"failures":           failures,
			"max_before_offline": m.maxConsecFailures,
			"error":              err.Error(),
		}).Warn("Worker node health check failed")

		if failures >= m.maxConsecFailures {
			m.setNodeOffline(ctx, nodeID, err.Error())
		}
		return
	}

	// Health check succeeded — reset failure counter
	m.consecutiveFailures[nodeID] = 0

	// Update node status
	m.mu.Lock()
	node.Status = WorkerStatusOnline
	now := time.Now()
	node.LastHealthCheck = &now
	node.LastError = ""
	m.mu.Unlock()

	// Sync models from agent (use independent timeout)
	syncCtx, syncCancel := context.WithTimeout(ctx, m.healthTimeout)
	m.syncNodeModels(syncCtx, nodeID)
	syncCancel()

	// Fetch system info (GPU/CPU/memory) with independent timeout
	sysCtx, sysCancel := context.WithTimeout(ctx, m.healthTimeout)
	sysInfo, sysErr := client.SystemInfo(sysCtx)
	sysCancel()

	// Update DB with health data
	var gpuJSON, cpuJSON, memJSON, modelsJSON []byte

	if sysErr == nil && sysInfo != nil {
		gpuJSON, _ = json.Marshal(sysInfo.GPUDevices)
		if sysInfo.CPU != nil {
			cpuJSON, _ = json.Marshal(sysInfo.CPU)
		}
		if sysInfo.Memory != nil {
			memJSON, _ = json.Marshal(sysInfo.Memory)
		}
	}

	// Collect running model aliases for this node
	m.mu.RLock()
	var runningAliases []string
	for alias, rm := range m.models {
		if rm.NodeID == nodeID && rm.Status == inference.StatusRunning {
			runningAliases = append(runningAliases, alias)
		}
	}
	m.mu.RUnlock()
	modelsJSON, _ = json.Marshal(runningAliases)

	_ = m.db.UpdateWorkerNodeHealth(ctx, nodeID, string(WorkerStatusOnline),
		gpuJSON, cpuJSON, memJSON, modelsJSON, "")
}

// syncNodeModels fetches model list from agent and updates local tracking.
func (m *Manager) syncNodeModels(ctx context.Context, nodeID string) {
	m.mu.RLock()
	client, ok := m.clients[nodeID]
	node, nodeOk := m.nodes[nodeID]
	m.mu.RUnlock()
	if !ok || !nodeOk {
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, m.healthTimeout)
	defer cancel()

	agentModels, err := client.ListModels(listCtx)
	if err != nil {
		m.logger.WithFields(logrus.Fields{
			"node_id": nodeID,
			"error":   err.Error(),
		}).Debug("Failed to list models from agent")
		return
	}

	m.mu.Lock()
	// Remove old models for this node
	for alias, rm := range m.models {
		if rm.NodeID == nodeID {
			delete(m.models, alias)
		}
	}
	// Add current models
	for _, am := range agentModels {
		if am.Status == inference.StatusRunning || am.Status == inference.StatusStarting {
			m.models[am.Alias] = &RemoteModel{
				Alias:    am.Alias,
				Provider: am.Provider,
				NodeID:   nodeID,
				NodeName: node.Name,
				NodeAddr: node.Address,
				Status:   am.Status,
			}
		}
	}
	m.mu.Unlock()
}

// setNodeOffline marks a node as offline and removes its models from tracking.
func (m *Manager) setNodeOffline(ctx context.Context, nodeID string, errMsg string) {
	m.mu.Lock()
	if node, ok := m.nodes[nodeID]; ok {
		node.Status = WorkerStatusOffline
		node.LastError = errMsg
	}
	// Remove all models from this offline node
	for alias, rm := range m.models {
		if rm.NodeID == nodeID {
			delete(m.models, alias)
		}
	}
	m.mu.Unlock()

	_ = m.db.UpdateWorkerNodeHealth(ctx, nodeID, string(WorkerStatusOffline),
		nil, nil, nil, nil, errMsg)

	m.logger.WithFields(logrus.Fields{
		"node_id": nodeID,
		"error":   errMsg,
	}).Warn("Worker node marked offline — models removed from routing")
}

// ──────────────────────────────────────────────────────────────
// RemoteModelResolver interface (for inference.Router)
// ──────────────────────────────────────────────────────────────

// FindRemoteModel implements inference.RemoteModelResolver.
// Called by inference.Router.EnsureByAlias when local resolution fails.
// Returns the agent base URL, provider kind, and HF repo name for correct model name rewriting.
func (m *Manager) FindRemoteModel(ctx context.Context, alias string) (nodeAddr string, provider inference.ProviderKind, found bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	rm, ok := m.models[alias]
	if !ok || rm.Status != inference.StatusRunning {
		return "", "", false
	}
	return rm.NodeAddr, rm.Provider, true
}

// ──────────────────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────────────────

func (m *Manager) createClient(node *models.WorkerNode) *Client {
	return NewClient(ClientConfig{
		BaseURL:       node.Address,
		APIKey:        node.APIKey,
		TLSSkipVerify: m.tlsSkipVerify,
		CACertPath:    m.caCertPath,
		Timeout:       m.healthTimeout,
	})
}

func (m *Manager) dbNodeToLive(node *models.WorkerNode) *WorkerNode {
	live := &WorkerNode{
		ID:              node.ID,
		Name:            node.Name,
		Address:         node.Address,
		Status:          WorkerNodeStatus(node.Status),
		NodeType:        NodeType(node.NodeType),
		ModelsRunning:   nil,
		LastHealthCheck: node.LastHealthCheck,
		LastError:       node.LastError,
		CreatedAt:       node.CreatedAt,
		UpdatedAt:       node.UpdatedAt,
	}

	// Parse GPU devices from JSONB
	if node.GPUInfo != nil {
		_ = json.Unmarshal(node.GPUInfo, &live.GPUDevices)
	}
	// Parse CPU info
	if node.CPUInfo != nil {
		var cpu CPUInfo
		if json.Unmarshal(node.CPUInfo, &cpu) == nil && cpu.Cores > 0 {
			live.CPUInfo = &cpu
		}
	}
	// Parse memory info
	if node.MemoryInfo != nil {
		var mem MemoryInfo
		if json.Unmarshal(node.MemoryInfo, &mem) == nil && mem.TotalMB > 0 {
			live.MemoryInfo = &mem
		}
	}
	// Parse running models list
	if node.ModelsRunning != nil {
		_ = json.Unmarshal(node.ModelsRunning, &live.ModelsRunning)
	}

	return live
}
