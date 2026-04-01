package sqlite

import (
	"context"
	"fmt"

	"aigateway/internal/models"
)

// Worker nodes are only supported in PostgreSQL (Agent Mode).
// These stubs satisfy the Database interface for SQLite.

func (db *SQLiteDB) CreateWorkerNode(ctx context.Context, node *models.WorkerNode) error {
	return fmt.Errorf("worker nodes are not supported in SQLite — use PostgreSQL")
}

func (db *SQLiteDB) GetWorkerNode(ctx context.Context, id string) (*models.WorkerNode, error) {
	return nil, fmt.Errorf("worker nodes are not supported in SQLite — use PostgreSQL")
}

func (db *SQLiteDB) GetWorkerNodeByName(ctx context.Context, name string) (*models.WorkerNode, error) {
	return nil, fmt.Errorf("worker nodes are not supported in SQLite — use PostgreSQL")
}

func (db *SQLiteDB) UpdateWorkerNode(ctx context.Context, node *models.WorkerNode) error {
	return fmt.Errorf("worker nodes are not supported in SQLite — use PostgreSQL")
}

func (db *SQLiteDB) DeleteWorkerNode(ctx context.Context, id string) error {
	return fmt.Errorf("worker nodes are not supported in SQLite — use PostgreSQL")
}

func (db *SQLiteDB) ListWorkerNodes(ctx context.Context) ([]*models.WorkerNode, error) {
	return nil, nil // Return empty list — workers simply not available on SQLite
}

func (db *SQLiteDB) UpdateWorkerNodeHealth(ctx context.Context, id string, status string, gpuInfo, cpuInfo, memoryInfo, modelsRunning []byte, lastError string) error {
	return fmt.Errorf("worker nodes are not supported in SQLite — use PostgreSQL")
}

// Transaction delegations for sqliteTx

func (tx *sqliteTx) CreateWorkerNode(ctx context.Context, node *models.WorkerNode) error {
	return tx.db.CreateWorkerNode(ctx, node)
}

func (tx *sqliteTx) GetWorkerNode(ctx context.Context, id string) (*models.WorkerNode, error) {
	return tx.db.GetWorkerNode(ctx, id)
}

func (tx *sqliteTx) GetWorkerNodeByName(ctx context.Context, name string) (*models.WorkerNode, error) {
	return tx.db.GetWorkerNodeByName(ctx, name)
}

func (tx *sqliteTx) UpdateWorkerNode(ctx context.Context, node *models.WorkerNode) error {
	return tx.db.UpdateWorkerNode(ctx, node)
}

func (tx *sqliteTx) DeleteWorkerNode(ctx context.Context, id string) error {
	return tx.db.DeleteWorkerNode(ctx, id)
}

func (tx *sqliteTx) ListWorkerNodes(ctx context.Context) ([]*models.WorkerNode, error) {
	return tx.db.ListWorkerNodes(ctx)
}

func (tx *sqliteTx) UpdateWorkerNodeHealth(ctx context.Context, id string, status string, gpuInfo, cpuInfo, memoryInfo, modelsRunning []byte, lastError string) error {
	return tx.db.UpdateWorkerNodeHealth(ctx, id, status, gpuInfo, cpuInfo, memoryInfo, modelsRunning, lastError)
}
