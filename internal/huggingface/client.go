// Package huggingface provides integration with Hugging Face Hub API
// Version: v3.0.0 - Direct model browsing and downloading
package huggingface

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	DefaultAPIURL = "https://huggingface.co"
	APIEndpoint   = "/api/models"
)

// Client represents Hugging Face API client
type Client struct {
	baseURL    string
	apiToken   string // Optional: для private models
	httpClient *http.Client
	logger     *logrus.Logger
}

// NewClient creates a new Hugging Face client
func NewClient(apiToken string, logger *logrus.Logger) *Client {
	return &Client{
		baseURL:  DefaultAPIURL,
		apiToken: apiToken,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// ModelFilters represents search filters for models
type ModelFilters struct {
	Search       string   // Model name search
	Author       string   // Filter by author/organization
	Tags         []string // Filter by tags (e.g., "gguf", "text-generation")
	Library      string   // Filter by library (e.g., "transformers", "llama.cpp")
	Language     []string // Filter by language
	License      []string // Filter by license
	Sort         string   // Sort by: "downloads", "likes", "trending", "createdAt"
	Direction    int      // Sort direction: -1 (desc), 1 (asc)
	Limit        int      // Results limit (default: 30, max: 100)
	CardData     bool     // Include model card data
	Config       bool     // Include config data
	FullResponse bool     // Return full model info
}

// ModelInfo represents model information from Hugging Face
type ModelInfo struct {
	ID            string    `json:"id"`               // e.g., "TheBloke/Llama-2-7B-GGUF"
	Author        string    `json:"author"`           // e.g., "TheBloke"
	ModelID       string    `json:"modelId"`          // Same as ID
	Private       bool      `json:"private"`          // Is private model
	Downloads     int       `json:"downloads"`        // Download count
	Likes         int       `json:"likes"`            // Likes count
	Tags          []string  `json:"tags"`             // Model tags
	PipelineTag   string    `json:"pipeline_tag"`     // e.g., "text-generation"
	Library       string    `json:"library_name"`     // e.g., "transformers"
	CreatedAt     time.Time `json:"createdAt"`        // Creation date
	LastModified  time.Time `json:"lastModified"`     // Last update date
	Siblings      []File    `json:"siblings"`         // Model files
	SHA           string    `json:"sha"`              // Git commit SHA
	Config        *Config   `json:"config,omitempty"` // Model config
	CardData      *CardData `json:"cardData,omitempty"` // Model card metadata
	
	// Computed fields
	TotalSize     int64  `json:"total_size,omitempty"`     // Total size of all files
	GGUFFiles     []File `json:"gguf_files,omitempty"`     // Only GGUF files
	HasGGUF       bool   `json:"has_gguf"`                 // Has GGUF files
	ParameterSize string `json:"parameter_size,omitempty"` // e.g., "7B", "13B"
}

// File represents a model file
type File struct {
	Filename string `json:"rfilename"` // Relative filename
	Size     int64  `json:"size"`      // File size in bytes
	BlobID   string `json:"blobId"`    // Git blob ID
	LFS      *LFS   `json:"lfs,omitempty"` // LFS pointer data
}

// LFS represents Git LFS pointer
type LFS struct {
	OID  string `json:"oid"`  // SHA256
	Size int64  `json:"size"` // Size in bytes
}

// Config represents model configuration
type Config struct {
	Architecture     []string `json:"architectures,omitempty"`
	ModelType        string   `json:"model_type,omitempty"`
	VocabSize        int      `json:"vocab_size,omitempty"`
	HiddenSize       int      `json:"hidden_size,omitempty"`
	NumAttentionHeads int     `json:"num_attention_heads,omitempty"`
	NumHiddenLayers  int      `json:"num_hidden_layers,omitempty"`
	MaxPositionEmbeddings int `json:"max_position_embeddings,omitempty"`
}

// CardData represents model card metadata
type CardData struct {
	Language []string          `json:"language,omitempty"`
	License  string            `json:"license,omitempty"`
	Tags     []string          `json:"tags,omitempty"`
	Datasets []string          `json:"datasets,omitempty"`
	Metrics  []string          `json:"metrics,omitempty"`
	BaseModel string           `json:"base_model,omitempty"`
	ModelIndex []ModelIndexItem `json:"model-index,omitempty"`
}

// ModelIndexItem represents model index entry
type ModelIndexItem struct {
	Name    string                 `json:"name"`
	Results []map[string]interface{} `json:"results,omitempty"`
}

// SearchModels searches for models on Hugging Face
func (c *Client) SearchModels(ctx context.Context, filters ModelFilters) ([]ModelInfo, error) {
	// Build query parameters
	params := url.Values{}
	
	if filters.Search != "" {
		params.Add("search", filters.Search)
	}
	
	if filters.Author != "" {
		params.Add("author", filters.Author)
	}
	
	// Add tags filter
	if len(filters.Tags) > 0 {
		for _, tag := range filters.Tags {
			params.Add("filter", tag)
		}
	}
	
	if filters.Library != "" {
		params.Add("library", filters.Library)
	}
	
	// Add language filter
	if len(filters.Language) > 0 {
		for _, lang := range filters.Language {
			params.Add("language", lang)
		}
	}
	
	// Sort
	if filters.Sort != "" {
		params.Add("sort", filters.Sort)
	}
	if filters.Direction != 0 {
		params.Add("direction", fmt.Sprintf("%d", filters.Direction))
	}
	
	// Limit
	limit := filters.Limit
	if limit <= 0 {
		limit = 30
	}
	if limit > 100 {
		limit = 100
	}
	params.Add("limit", fmt.Sprintf("%d", limit))
	
	// Full response
	if filters.FullResponse {
		params.Add("full", "true")
	}
	if filters.CardData {
		params.Add("cardData", "true")
	}
	if filters.Config {
		params.Add("config", "true")
	}
	
	// Build URL
	reqURL := fmt.Sprintf("%s%s?%s", c.baseURL, APIEndpoint, params.Encode())
	
	c.logger.WithFields(logrus.Fields{
		"url":    reqURL,
		"search": filters.Search,
		"tags":   filters.Tags,
	}).Debug("Searching Hugging Face models")
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add authorization header if token is provided
	if c.apiToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))
	}
	
	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var models []ModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	// Enrich models with computed fields
	for i := range models {
		c.enrichModelInfo(&models[i])
	}
	
	c.logger.WithField("count", len(models)).Debug("Found models")
	
	return models, nil
}

// GetModelInfo retrieves detailed information about a specific model
func (c *Client) GetModelInfo(ctx context.Context, modelID string) (*ModelInfo, error) {
	// Build URL
	reqURL := fmt.Sprintf("%s%s/%s", c.baseURL, APIEndpoint, modelID)
	
	c.logger.WithField("model_id", modelID).Debug("Fetching model info")
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	// Add authorization header if token is provided
	if c.apiToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))
	}
	
	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse response
	var model ModelInfo
	if err := json.NewDecoder(resp.Body).Decode(&model); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	
	// Enrich model with computed fields
	c.enrichModelInfo(&model)
	
	return &model, nil
}

// enrichModelInfo adds computed fields to model info
func (c *Client) enrichModelInfo(model *ModelInfo) {
	// Filter GGUF files
	ggufFiles := []File{}
	var totalSize int64
	
	for _, file := range model.Siblings {
		// Calculate total size
		if file.LFS != nil {
			totalSize += file.LFS.Size
		} else {
			totalSize += file.Size
		}
		
		// Check if GGUF file
		if strings.HasSuffix(strings.ToLower(file.Filename), ".gguf") {
			ggufFiles = append(ggufFiles, file)
		}
	}
	
	model.GGUFFiles = ggufFiles
	model.HasGGUF = len(ggufFiles) > 0
	model.TotalSize = totalSize
	
	// Extract parameter size from tags
	for _, tag := range model.Tags {
		tag = strings.ToLower(tag)
		if strings.Contains(tag, "b") && (strings.Contains(tag, "7") || strings.Contains(tag, "13") || strings.Contains(tag, "70")) {
			model.ParameterSize = strings.ToUpper(tag)
			break
		}
	}
	
	// Try to extract from model ID
	if model.ParameterSize == "" {
		parts := strings.Split(strings.ToLower(model.ID), "-")
		for _, part := range parts {
			if strings.HasSuffix(part, "b") && len(part) <= 4 {
				model.ParameterSize = strings.ToUpper(part)
				break
			}
		}
	}
}

// ListGGUFModels returns models with GGUF files
func (c *Client) ListGGUFModels(ctx context.Context, search string, limit int) ([]ModelInfo, error) {
	filters := ModelFilters{
		Search:       search,
		Tags:         []string{"gguf"},
		Sort:         "downloads",
		Direction:    -1,
		Limit:        limit,
		FullResponse: true,
	}
	
	return c.SearchModels(ctx, filters)
}

// GetFileURL returns direct download URL for a model file
func (c *Client) GetFileURL(modelID, filename string) string {
	return fmt.Sprintf("%s/%s/resolve/main/%s", c.baseURL, modelID, filename)
}

// FormatFileSize formats file size to human-readable string
func FormatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)
	
	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

