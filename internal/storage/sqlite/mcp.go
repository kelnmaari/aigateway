// Package sqlite provides SQLite implementation for MCP servers storage
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"ollama-openai-proxy/internal/models"
)

// CreateMCPServer создает новую запись MCP сервера
func (s *SQLiteDB) CreateMCPServer(ctx context.Context, server *models.MCPServer) error {
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
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err = s.db.ExecContext(ctx, query,
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

	s.logger.WithField("mcp_server_id", server.ID).Info("MCP server created")
	return nil
}

// GetMCPServer получает MCP сервер по ID
func (s *SQLiteDB) GetMCPServer(ctx context.Context, id string) (*models.MCPServer, error) {
	query := `
		SELECT id, name, description, category, installation_guide,
		       website_url, github_url, tags, is_active, created_at, updated_at
		FROM mcp_servers
		WHERE id = ?
	`

	server := &models.MCPServer{}
	var tagsJSON string

	err := s.db.QueryRowContext(ctx, query, id).Scan(
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
			s.logger.WithError(err).Warn("Failed to unmarshal tags, using empty array")
			server.Tags = []string{}
		}
	} else {
		server.Tags = []string{}
	}

	return server, nil
}

// UpdateMCPServer обновляет MCP сервер
func (s *SQLiteDB) UpdateMCPServer(ctx context.Context, server *models.MCPServer) error {
	server.UpdatedAt = time.Now()

	// Serialize tags to JSON
	tagsJSON, err := json.Marshal(server.Tags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	query := `
		UPDATE mcp_servers
		SET name = ?, description = ?, category = ?, installation_guide = ?,
		    website_url = ?, github_url = ?, tags = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`

	result, err := s.db.ExecContext(ctx, query,
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

	s.logger.WithField("mcp_server_id", server.ID).Info("MCP server updated")
	return nil
}

// DeleteMCPServer удаляет MCP сервер
func (s *SQLiteDB) DeleteMCPServer(ctx context.Context, id string) error {
	query := `DELETE FROM mcp_servers WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete MCP server: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("MCP server not found: %s", id)
	}

	s.logger.WithField("mcp_server_id", id).Info("MCP server deleted")
	return nil
}

// ListMCPServers возвращает список MCP серверов с фильтрацией
func (s *SQLiteDB) ListMCPServers(ctx context.Context, req models.MCPServerListRequest) (*models.MCPServerListResponse, error) {
	// Build query
	baseQuery := `
		SELECT id, name, description, category, installation_guide,
		       website_url, github_url, tags, is_active, created_at, updated_at
		FROM mcp_servers
		WHERE 1=1
	`
	countQuery := `SELECT COUNT(*) FROM mcp_servers WHERE 1=1`

	var args []interface{}
	var filters string

	// Apply filters
	if req.ActiveOnly {
		filters += ` AND is_active = ?`
		args = append(args, true)
	}

	if req.Category != nil && *req.Category != "" {
		filters += ` AND category = ?`
		args = append(args, *req.Category)
	}

	if req.Search != nil && *req.Search != "" {
		searchPattern := "%" + *req.Search + "%"
		filters += ` AND (name LIKE ? OR description LIKE ? OR tags LIKE ?)`
		args = append(args, searchPattern, searchPattern, searchPattern)
	}

	// Get total count
	var total int
	err := s.db.QueryRowContext(ctx, countQuery+filters, args...).Scan(&total)
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

	rows, err := s.db.QueryContext(ctx, finalQuery, args...)
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
				s.logger.WithError(err).Warn("Failed to unmarshal tags, using empty array")
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
