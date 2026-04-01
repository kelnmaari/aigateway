package models

import (
	"encoding/json"
	"time"
)

// WorkerNode represents a remote inference worker node in the database.
type WorkerNode struct {
	ID               string          `json:"id"`
	Name             string          `json:"name"`
	Address          string          `json:"address"`            // https://host:port
	APIKey           string          `json:"-"`                  // never expose in API
	Status           string          `json:"status"`             // pending | online | offline | draining
	NodeType         string          `json:"node_type"`          // gpu | cpu | mixed
	GPUInfo          json.RawMessage `json:"gpu_info"`           // [{index,name,memory_total_mb,...}]
	CPUInfo          json.RawMessage `json:"cpu_info"`           // {model,cores,threads}
	MemoryInfo       json.RawMessage `json:"memory_info"`        // {total_mb,used_mb,free_mb}
	ModelsRunning    json.RawMessage `json:"models_running"`     // ["alias1","alias2"]
	MaxRunningModels int             `json:"max_running_models"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	LastHealthCheck  *time.Time      `json:"last_health_check,omitempty"`
	LastError        string          `json:"last_error,omitempty"`
}
