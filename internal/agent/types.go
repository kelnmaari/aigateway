package agent

import (
	"time"

	"aigateway/internal/inference"
)

// WorkerNodeStatus represents the health state of a worker node.
type WorkerNodeStatus string

const (
	WorkerStatusOnline   WorkerNodeStatus = "online"
	WorkerStatusOffline  WorkerNodeStatus = "offline"
	WorkerStatusPending  WorkerNodeStatus = "pending"  // registered but never connected
	WorkerStatusDraining WorkerNodeStatus = "draining" // no new models, finishing existing
)

// NodeType describes what kind of hardware the worker has.
type NodeType string

const (
	NodeTypeGPU   NodeType = "gpu"
	NodeTypeCPU   NodeType = "cpu"
	NodeTypeMixed NodeType = "mixed"
)

// WorkerNode represents a remote inference worker.
type WorkerNode struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Address         string           `json:"address"`      // https://host:port
	APIKey          string           `json:"-"`             // never expose in API responses
	Status          WorkerNodeStatus `json:"status"`
	NodeType        NodeType         `json:"node_type"`
	GPUDevices      []GPUDevice      `json:"gpu_devices"`
	CPUInfo         *CPUInfo         `json:"cpu_info,omitempty"`
	MemoryInfo      *MemoryInfo      `json:"memory_info,omitempty"`
	ModelsRunning   []string         `json:"models_running"`
	LastHealthCheck *time.Time       `json:"last_health_check,omitempty"`
	LastError       string           `json:"last_error,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// GPUDevice describes a single GPU on a worker.
type GPUDevice struct {
	Index          int     `json:"index"`
	Name           string  `json:"name"`
	MemoryTotalMB  float64 `json:"memory_total_mb"`
	MemoryUsedMB   float64 `json:"memory_used_mb"`
	MemoryFreeMB   float64 `json:"memory_free_mb"`
	UtilizationGPU uint32  `json:"utilization_gpu"` // 0-100 %
	TemperatureC   uint32  `json:"temperature_c"`
	PowerDrawW     float64 `json:"power_draw_w,omitempty"`
	PowerLimitW    float64 `json:"power_limit_w,omitempty"`
}

// CPUInfo describes the CPU on a worker.
type CPUInfo struct {
	Model   string `json:"model"`
	Cores   int    `json:"cores"`
	Threads int    `json:"threads"`
}

// MemoryInfo describes system RAM on a worker.
type MemoryInfo struct {
	TotalMB float64 `json:"total_mb"`
	UsedMB  float64 `json:"used_mb"`
	FreeMB  float64 `json:"free_mb"`
}

// AgentModelStatus is returned by agents for each running model.
type AgentModelStatus struct {
	Alias     string                `json:"alias"`
	Status    inference.ModelStatus `json:"status"`
	Provider  inference.ProviderKind `json:"provider"`
	Endpoint  string                `json:"endpoint"`   // agent-local endpoint (e.g. http://localhost:8001)
	StartedAt *time.Time            `json:"started_at,omitempty"`
	Error     string                `json:"error,omitempty"`
	VRAMMB    float64               `json:"vram_mb,omitempty"`
}

// AgentHealthResponse is the health check response from an agent.
type AgentHealthResponse struct {
	Status        string   `json:"status"` // "ok" or "error"
	NodeName      string   `json:"node_name"`
	NodeType      NodeType `json:"node_type"`
	UptimeSeconds int64    `json:"uptime_seconds"`
	ModelsRunning int      `json:"models_running"`
	Version       string   `json:"version"`
}

// AgentSystemInfo is the system information response from an agent.
type AgentSystemInfo struct {
	GPUDevices []GPUDevice `json:"gpu_devices"`
	CPU        *CPUInfo    `json:"cpu,omitempty"`
	Memory     *MemoryInfo `json:"memory,omitempty"`
}

// LoadModelRequest is sent to an agent to start a model.
type LoadModelRequest struct {
	inference.ModelSpec
}

// LoadModelResponse is returned by the agent after model load.
type LoadModelResponse struct {
	Alias    string                `json:"alias"`
	Status   inference.ModelStatus `json:"status"`
	Endpoint string                `json:"endpoint,omitempty"`
	Error    string                `json:"error,omitempty"`
}

// StopModelRequest is sent to an agent to stop a model.
type StopModelRequest struct {
	Alias string `json:"alias"`
}

// GenerateConfigRequest is the admin API request for generating agent config.
type GenerateConfigRequest struct {
	Name         string   `json:"name" binding:"required"`
	Address      string   `json:"address" binding:"required"` // URL where main server reaches agent (e.g. http://192.168.1.50:9090)
	NodeType     NodeType `json:"node_type"`
	ListenAddr   string   `json:"listen_addr"`
	HFCacheDir   string   `json:"hf_cache_dir"`
	GGUFCacheDir string   `json:"gguf_cache_dir"`
	HFToken      string   `json:"hf_token,omitempty"`
	GenerateTLS  bool     `json:"generate_tls"`
}

// GenerateConfigResponse is the admin API response with generated config.
type GenerateConfigResponse struct {
	ConfigYAML     string `json:"config_yaml"`
	APIKey         string `json:"api_key"`
	InstallCommand string `json:"install_command"`
	PostInstall    string `json:"post_install"`
	WorkerID       string `json:"worker_id"` // ID of pre-registered worker node
}
