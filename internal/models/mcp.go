// Package models provides data structures for Ollama-OpenAI Proxy
package models

import "time"

// MCPServer represents an MCP (Model Context Protocol) server entry in the catalog
type MCPServer struct {
	ID                string    `json:"id" db:"id"`
	Name              string    `json:"name" db:"name"`
	Description       string    `json:"description" db:"description"`
	Category          string    `json:"category" db:"category"`
	InstallationGuide string    `json:"installation_guide" db:"installation_guide"`
	WebsiteURL        string    `json:"website_url,omitempty" db:"website_url"`
	GitHubURL         string    `json:"github_url,omitempty" db:"github_url"`
	Tags              []string  `json:"tags" db:"tags"`
	IsActive          bool      `json:"is_active" db:"is_active"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// MCPServerListRequest represents request parameters for listing MCP servers
type MCPServerListRequest struct {
	Limit      int     `json:"limit" form:"limit"`
	Offset     int     `json:"offset" form:"offset"`
	Category   *string `json:"category,omitempty" form:"category"`
	Search     *string `json:"search,omitempty" form:"search"`
	ActiveOnly bool    `json:"active_only" form:"active_only"`
	SortBy     string  `json:"sort_by" form:"sort_by"`
	SortOrder  string  `json:"sort_order" form:"sort_order"`
}

// MCPServerListResponse represents response for listing MCP servers
type MCPServerListResponse struct {
	Servers []MCPServer `json:"servers"`
	Total   int         `json:"total"`
	Limit   int         `json:"limit"`
	Offset  int         `json:"offset"`
}

// CreateMCPServerRequest represents request to create an MCP server entry
type CreateMCPServerRequest struct {
	Name              string   `json:"name" binding:"required"`
	Description       string   `json:"description" binding:"required"`
	Category          string   `json:"category" binding:"required"`
	InstallationGuide string   `json:"installation_guide" binding:"required"`
	WebsiteURL        string   `json:"website_url,omitempty"`
	GitHubURL         string   `json:"github_url,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	IsActive          *bool    `json:"is_active,omitempty"`
}

// UpdateMCPServerRequest represents request to update an MCP server entry
type UpdateMCPServerRequest struct {
	Name              *string  `json:"name,omitempty"`
	Description       *string  `json:"description,omitempty"`
	Category          *string  `json:"category,omitempty"`
	InstallationGuide *string  `json:"installation_guide,omitempty"`
	WebsiteURL        *string  `json:"website_url,omitempty"`
	GitHubURL         *string  `json:"github_url,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	IsActive          *bool    `json:"is_active,omitempty"`
}

// MCPCategory represents available MCP server categories
type MCPCategory string

const (
	MCPCategoryDevelopment  MCPCategory = "development"
	MCPCategoryProductivity MCPCategory = "productivity"
	MCPCategoryDatabase     MCPCategory = "database"
	MCPCategoryCloud        MCPCategory = "cloud"
	MCPCategoryAI           MCPCategory = "ai"
	MCPCategoryOther        MCPCategory = "other"
)

// GetMCPCategories returns list of available MCP categories
func GetMCPCategories() []string {
	return []string{
		string(MCPCategoryDevelopment),
		string(MCPCategoryProductivity),
		string(MCPCategoryDatabase),
		string(MCPCategoryCloud),
		string(MCPCategoryAI),
		string(MCPCategoryOther),
	}
}

