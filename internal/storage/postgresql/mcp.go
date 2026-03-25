// Package sqlite provides SQLite implementation for MCP servers storage
package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"aigateway/internal/models"
)

// CreateMCPServer создает новую запись MCP сервера
func (db *PostgreSQLDB) CreateMCPServer(ctx context.Context, server *models.MCPServer) error {
	// Set timestamps
	now := time.Now()
	if server.CreatedAt.IsZero() {
		server.CreatedAt = now
	}
	server.UpdatedAt = now

	// Serialize tags to JSON
	tagsJSON, err := json.Marshal(server.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		INSERT INTO mcp_servers (
			id, name, description, category, installation_guide,
			website_url, github_url, tags, is_active, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err = db.db.ExecContext(ctx, query,
		server.ID,
		server.Name,
		server.Description,
		server.Category,
		server.InstallationGuide,
		server.WebsiteURL,
		server.GitHubURL,
		string(tagsJSON),
		server.IsActive,
		server.CreatedAt,
		server.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create MCP server: %w", err)
	}

	db.logger.WithField("mcp_server_id", server.ID).Info("MCP server created")
	return nil
}

// GetMCPServer получает MCP сервер по ID
func (db *PostgreSQLDB) GetMCPServer(ctx context.Context, id string) (*models.MCPServer, error) {
	query := `
		SELECT id, name, description, category, installation_guide,
		       website_url, github_url, tags, is_active, created_at, updated_at
		FROM mcp_servers
		WHERE id = $1
	`

	server := &models.MCPServer{}
	var tagsJSON string

	err := db.db.QueryRowContext(ctx, query, id).Scan(
		&server.ID,
		&server.Name,
		&server.Description,
		&server.Category,
		&server.InstallationGuide,
		&server.WebsiteURL,
		&server.GitHubURL,
		&tagsJSON,
		&server.IsActive,
		&server.CreatedAt,
		&server.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("MCP server not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get MCP server: %w", err)
	}

	// Deserialize tags
	if tagsJSON != "" && tagsJSON != "null" {
		if err := json.Unmarshal([]byte(tagsJSON), &server.Tags); err != nil {
			db.logger.WithError(err).Warn("Failed to unmarshal tags, using empty array")
			server.Tags = []string{}
		}
	} else {
		server.Tags = []string{}
	}

	return server, nil
}

// UpdateMCPServer обновляет MCP сервер
func (db *PostgreSQLDB) UpdateMCPServer(ctx context.Context, server *models.MCPServer) error {
	server.UpdatedAt = time.Now()

	// Serialize tags to JSON
	tagsJSON, err := json.Marshal(server.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		UPDATE mcp_servers
		SET name = $1, description = $2, category = $3, installation_guide = $4,
		    website_url = $5, github_url = $6, tags = $7, is_active = $8, updated_at = $9
		WHERE id = $10
	`

	result, err := db.db.ExecContext(ctx, query,
		server.Name,
		server.Description,
		server.Category,
		server.InstallationGuide,
		server.WebsiteURL,
		server.GitHubURL,
		string(tagsJSON),
		server.IsActive,
		server.UpdatedAt,
		server.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update MCP server: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("MCP server not found: %s", server.ID)
	}

	db.logger.WithField("mcp_server_id", server.ID).Info("MCP server updated")
	return nil
}

// DeleteMCPServer удаляет MCP сервер
func (db *PostgreSQLDB) DeleteMCPServer(ctx context.Context, id string) error {
	query := `DELETE FROM mcp_servers WHERE id = $1`

	result, err := db.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete MCP server: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("MCP server not found: %s", id)
	}

	db.logger.WithField("mcp_server_id", id).Info("MCP server deleted")
	return nil
}

// ListMCPServers возвращает список MCP серверов с фильтрацией
func (db *PostgreSQLDB) ListMCPServers(ctx context.Context, req models.MCPServerListRequest) (*models.MCPServerListResponse, error) {
	// Build query
	baseQuery := `
		SELECT id, name, description, category, installation_guide,
		       website_url, github_url, tags, is_active, created_at, updated_at
		FROM mcp_servers
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM mcp_servers WHERE 1=1`

	var args []any
	var filters string

	// Apply filters
	if req.ActiveOnly {
		filters += ` AND is_active = $1`
		args = append(args, true)
	}

	if req.Category != nil && *req.Category != "" {
		filters += ` AND category = $1`
		args = append(args, *req.Category)
	}

	if req.Search != nil && *req.Search != "" {
		searchPattern := "%" + *req.Search + "%"
		filters += ` AND (name LIKE $1 OR description LIKE $2 OR tags LIKE $3)`
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	// Get total count
	var total int
	err := db.db.QueryRowContext(ctx, countQuery+filters, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count MCP servers: %w", err)
	}

	// Sorting
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := req.SortOrder
	if sortOrder == "" {
		sortOrder = "DESC"
	}

	orderClause := fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Pagination
	limit := req.Limit
	if limit == 0 {
		limit = 50
	}
	offset := req.Offset

	limitClause := fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	// Final query
	finalQuery := baseQuery + filters + orderClause + limitClause

	rows, err := db.db.QueryContext(ctx, finalQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list MCP servers: %w", err)
	}
	defer rows.Close()

	servers := make([]models.MCPServer, 0)
	for rows.Next() {
		server := models.MCPServer{}
		var tagsJSON string

		err := rows.Scan(
			&server.ID,
			&server.Name,
			&server.Description,
			&server.Category,
			&server.InstallationGuide,
			&server.WebsiteURL,
			&server.GitHubURL,
			&tagsJSON,
			&server.IsActive,
			&server.CreatedAt,
			&server.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan MCP server: %w", err)
		}

		// Deserialize tags
		if tagsJSON != "" && tagsJSON != "null" {
			if err := json.Unmarshal([]byte(tagsJSON), &server.Tags); err != nil {
				db.logger.WithError(err).Warn("Failed to unmarshal tags, using empty array")
				server.Tags = []string{}
			}
		} else {
			server.Tags = []string{}
		}

		servers = append(servers, server)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating MCP servers: %w", err)
	}

	return &models.MCPServerListResponse{
		Servers: servers,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
	}, nil
}
