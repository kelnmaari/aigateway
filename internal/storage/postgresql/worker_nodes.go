package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"aigateway/internal/models"
)

// ========================================
// Worker Nodes CRUD Operations (Agent Mode)
// ========================================

// CreateWorkerNode registers a new remote inference worker node.
func (db *PostgreSQLDB) CreateWorkerNode(ctx context.Context, node *models.WorkerNode) error {
	if node.ID == "" {
		node.ID = generateID()
	}

	now := time.Now()
	node.CreatedAt = now
	node.UpdatedAt = now

	if node.Status == "" {
		node.Status = "pending"
	}
	if node.NodeType == "" {
		node.NodeType = "gpu"
	}
	if node.MaxRunningModels <= 0 {
		node.MaxRunningModels = 2
	}

	// Ensure JSON fields are not nil
	if node.GPUInfo == nil {
		node.GPUInfo = json.RawMessage("[]")
	}
	if node.CPUInfo == nil {
		node.CPUInfo = json.RawMessage("{}")
	}
	if node.MemoryInfo == nil {
		node.MemoryInfo = json.RawMessage("{}")
	}
	if node.ModelsRunning == nil {
		node.ModelsRunning = json.RawMessage("[]")
	}

	query := `
		INSERT INTO worker_nodes (
			id, name, address, api_key, status, node_type,
			gpu_info, cpu_info, memory_info, models_running,
			max_running_models, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err := db.db.ExecContext(ctx, query,
		node.ID, node.Name, node.Address, node.APIKey, node.Status, node.NodeType,
		node.GPUInfo, node.CPUInfo, node.MemoryInfo, node.ModelsRunning,
		node.MaxRunningModels, node.CreatedAt, node.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create worker node: %w", err)
	}
	return nil
}

// GetWorkerNode retrieves a worker node by ID.
func (db *PostgreSQLDB) GetWorkerNode(ctx context.Context, id string) (*models.WorkerNode, error) {
	query := `
		SELECT id, name, address, api_key, status, node_type,
			   gpu_info, cpu_info, memory_info, models_running,
			   max_running_models, created_at, updated_at, last_health_check, last_error
		FROM worker_nodes WHERE id = $1
	`
	return db.scanWorkerNode(ctx, query, id)
}

// GetWorkerNodeByName retrieves a worker node by its unique name.
func (db *PostgreSQLDB) GetWorkerNodeByName(ctx context.Context, name string) (*models.WorkerNode, error) {
	query := `
		SELECT id, name, address, api_key, status, node_type,
			   gpu_info, cpu_info, memory_info, models_running,
			   max_running_models, created_at, updated_at, last_health_check, last_error
		FROM worker_nodes WHERE name = $1
	`
	return db.scanWorkerNode(ctx, query, name)
}

// UpdateWorkerNode updates worker node configuration.
func (db *PostgreSQLDB) UpdateWorkerNode(ctx context.Context, node *models.WorkerNode) error {
	node.UpdatedAt = time.Now()

	query := `
		UPDATE worker_nodes SET
			name = $2, address = $3, api_key = $4, status = $5, node_type = $6,
			max_running_models = $7, updated_at = $8
		WHERE id = $1
	`
	result, err := db.db.ExecContext(ctx, query,
		node.ID, node.Name, node.Address, node.APIKey, node.Status, node.NodeType,
		node.MaxRunningModels, node.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("update worker node: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("worker node not found: %s", node.ID)
	}
	return nil
}

// DeleteWorkerNode removes a worker node by ID.
func (db *PostgreSQLDB) DeleteWorkerNode(ctx context.Context, id string) error {
	result, err := db.db.ExecContext(ctx, "DELETE FROM worker_nodes WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete worker node: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("worker node not found: %s", id)
	}
	return nil
}

// ListWorkerNodes returns all registered worker nodes.
func (db *PostgreSQLDB) ListWorkerNodes(ctx context.Context) ([]*models.WorkerNode, error) {
	query := `
		SELECT id, name, address, api_key, status, node_type,
			   gpu_info, cpu_info, memory_info, models_running,
			   max_running_models, created_at, updated_at, last_health_check, last_error
		FROM worker_nodes ORDER BY name ASC
	`
	rows, err := db.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list worker nodes: %w", err)
	}
	defer rows.Close()

	var nodes []*models.WorkerNode
	for rows.Next() {
		node, err := scanWorkerNodeRow(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, rows.Err()
}

// UpdateWorkerNodeHealth updates health check timestamp, status, GPU/CPU/memory info, and running models.
func (db *PostgreSQLDB) UpdateWorkerNodeHealth(ctx context.Context, id string, status string, gpuInfo, cpuInfo, memoryInfo, modelsRunning []byte, lastError string) error {
	now := time.Now()

	// Default JSON values if nil
	if gpuInfo == nil {
		gpuInfo = []byte("[]")
	}
	if cpuInfo == nil {
		cpuInfo = []byte("{}")
	}
	if memoryInfo == nil {
		memoryInfo = []byte("{}")
	}
	if modelsRunning == nil {
		modelsRunning = []byte("[]")
	}

	query := `
		UPDATE worker_nodes SET
			status = $2, gpu_info = $3, cpu_info = $4, memory_info = $5,
			models_running = $6, last_health_check = $7, last_error = $8, updated_at = $7
		WHERE id = $1
	`
	_, err := db.db.ExecContext(ctx, query,
		id, status, gpuInfo, cpuInfo, memoryInfo,
		modelsRunning, now, lastError,
	)
	if err != nil {
		return fmt.Errorf("update worker node health: %w", err)
	}
	return nil
}

// ──────────────────────────────────────────────────────────────
// Internal helpers
// ──────────────────────────────────────────────────────────────

func (db *PostgreSQLDB) scanWorkerNode(ctx context.Context, query string, arg interface{}) (*models.WorkerNode, error) {
	row := db.db.QueryRowContext(ctx, query, arg)
	var node models.WorkerNode
	var lastHealthCheck sql.NullTime
	var lastError sql.NullString

	err := row.Scan(
		&node.ID, &node.Name, &node.Address, &node.APIKey, &node.Status, &node.NodeType,
		&node.GPUInfo, &node.CPUInfo, &node.MemoryInfo, &node.ModelsRunning,
		&node.MaxRunningModels, &node.CreatedAt, &node.UpdatedAt, &lastHealthCheck, &lastError,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("worker node not found")
		}
		return nil, fmt.Errorf("scan worker node: %w", err)
	}

	if lastHealthCheck.Valid {
		node.LastHealthCheck = &lastHealthCheck.Time
	}
	if lastError.Valid {
		node.LastError = lastError.String
	}
	return &node, nil
}

// scanWorkerNodeRow scans a single row from a rows iterator.
func scanWorkerNodeRow(rows *sql.Rows) (*models.WorkerNode, error) {
	var node models.WorkerNode
	var lastHealthCheck sql.NullTime
	var lastError sql.NullString

	err := rows.Scan(
		&node.ID, &node.Name, &node.Address, &node.APIKey, &node.Status, &node.NodeType,
		&node.GPUInfo, &node.CPUInfo, &node.MemoryInfo, &node.ModelsRunning,
		&node.MaxRunningModels, &node.CreatedAt, &node.UpdatedAt, &lastHealthCheck, &lastError,
	)
	if err != nil {
		return nil, fmt.Errorf("scan worker node row: %w", err)
	}

	if lastHealthCheck.Valid {
		node.LastHealthCheck = &lastHealthCheck.Time
	}
	if lastError.Valid {
		node.LastError = lastError.String
	}
	return &node, nil
}

// ──────────────────────────────────────────────────────────────
// Transaction delegation (postgresqlTx → PostgreSQLDB)
// ──────────────────────────────────────────────────────────────

func (tx *postgresqlTx) CreateWorkerNode(ctx context.Context, node *models.WorkerNode) error {
	return tx.db.CreateWorkerNode(ctx, node)
}

func (tx *postgresqlTx) GetWorkerNode(ctx context.Context, id string) (*models.WorkerNode, error) {
	return tx.db.GetWorkerNode(ctx, id)
}

func (tx *postgresqlTx) GetWorkerNodeByName(ctx context.Context, name string) (*models.WorkerNode, error) {
	return tx.db.GetWorkerNodeByName(ctx, name)
}

func (tx *postgresqlTx) UpdateWorkerNode(ctx context.Context, node *models.WorkerNode) error {
	return tx.db.UpdateWorkerNode(ctx, node)
}

func (tx *postgresqlTx) DeleteWorkerNode(ctx context.Context, id string) error {
	return tx.db.DeleteWorkerNode(ctx, id)
}

func (tx *postgresqlTx) ListWorkerNodes(ctx context.Context) ([]*models.WorkerNode, error) {
	return tx.db.ListWorkerNodes(ctx)
}

func (tx *postgresqlTx) UpdateWorkerNodeHealth(ctx context.Context, id string, status string, gpuInfo, cpuInfo, memoryInfo, modelsRunning []byte, lastError string) error {
	return tx.db.UpdateWorkerNodeHealth(ctx, id, status, gpuInfo, cpuInfo, memoryInfo, modelsRunning, lastError)
}
