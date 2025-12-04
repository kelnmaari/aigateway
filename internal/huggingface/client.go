// Package huggingface provides integration with Hugging Face Hub API
// Version: v3.0.8 - Added circuit breaker for resilience (sony/gobreaker)
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
	"github.com/sony/gobreaker/v2"
)

const (
	DefaultAPIURL = "https://huggingface.co"
	APIEndpoint   = "/api/models"
)

// Client represents Hugging Face API client
type Client struct {
	baseURL          string
	apiToken         string // Optional: для private models
	httpClient       *http.Client // For API requests (with timeout)
	downloadClient   *http.Client // For file downloads (without timeout)
	circuitBreaker   *gobreaker.CircuitBreaker[any] // v3.0.8: Circuit breaker for API resilience
	logger           *logrus.Logger
}

// NewClient creates a new Hugging Face client
func NewClient(apiToken string, logger *logrus.Logger) *Client {
	logger.WithFields(logrus.Fields{
		"api_timeout":      "30s",
		"download_timeout": "none (context-controlled)",
		"circuit_breaker":  "5 failures, 2min timeout (sony/gobreaker)",
	}).Debug("Hugging Face client initialized with separate HTTP clients and circuit breaker")
	
	// Circuit breaker settings (v3.0.8)
	cbSettings := gobreaker.Settings{
		Name:        "HuggingFaceAPI",
		MaxRequests: 3,  // Half-open: allow 3 requests to test
		Interval:    0,  // No automatic state reset (manual timeout only)
		Timeout:     2 * time.Minute, // Open -> Half-open after 2 minutes
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Open circuit after 5 consecutive failures
			return counts.ConsecutiveFailures >= 5
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			logger.WithFields(logrus.Fields{
				"circuit": name,
				"from":    from.String(),
				"to":      to.String(),
			}).Warn("Circuit breaker state changed")
		},
	}
	
	return &Client{
		baseURL:  DefaultAPIURL,
		apiToken: apiToken,
		// HTTP client for API requests - short timeout
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		// HTTP client for downloads - no timeout (context controls it)
		downloadClient: &http.Client{
			Timeout: 0, // No timeout - use context instead
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				// Disable compression to accurately track download progress
				DisableCompression: true,
			},
		},
		// Circuit breaker for API resilience (v3.0.8)
		circuitBreaker: gobreaker.NewCircuitBreaker[any](cbSettings),
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
	
	// README content (fetched separately)
	Description   string `json:"description,omitempty"`    // Model README/description (truncated)
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
	Language   FlexibleStringArray `json:"language,omitempty"`   // Can be string or []string
	License    string              `json:"license,omitempty"`
	Tags       FlexibleStringArray `json:"tags,omitempty"`       // Can be string or []string
	Datasets   FlexibleStringArray `json:"datasets,omitempty"`   // Can be string or []string
	Metrics    FlexibleStringArray `json:"metrics,omitempty"`    // Can be string or []string
	BaseModel  FlexibleStringArray `json:"base_model,omitempty"` // Can be string or []string
	ModelIndex []ModelIndexItem    `json:"model-index,omitempty"`
}

// FlexibleStringArray can unmarshal from either a string or []string
type FlexibleStringArray []string

// UnmarshalJSON implements custom unmarshaling for FlexibleStringArray
func (f *FlexibleStringArray) UnmarshalJSON(data []byte) error {
	// Try unmarshaling as array first
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*f = FlexibleStringArray(arr)
		return nil
	}
	
	// Try unmarshaling as single string
	var str string
	if err := json.Unmarshal(data, &str); err == nil {
		*f = FlexibleStringArray([]string{str})
		return nil
	}
	
	// If both fail, return empty array
	*f = FlexibleStringArray([]string{})
	return nil
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
	
	var models []ModelInfo
	
	// Execute request with circuit breaker (v3.0.8)
	_, err := c.circuitBreaker.Execute(func() (any, error) {
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
			return nil, fmt.Errorf("HuggingFace API error: %w", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}
		
		// Parse response
		if err := json.NewDecoder(resp.Body).Decode(&models); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		
		return nil, nil
	})
	
	if err != nil {
		return nil, err
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
	// Build URL with query parameters to get full model information
	params := url.Values{}
	params.Add("expand[]", "siblings")    // Include full file information with LFS data
	params.Add("expand[]", "cardData")    // Include model card metadata (license, languages, base_model)
	params.Add("expand[]", "config")      // Include model config (architecture, hidden_size, etc.)
	params.Add("expand[]", "lastModified")// Include last modified date
	params.Add("expand[]", "downloads")   // Include download count
	params.Add("expand[]", "tags")        // Include tags
	params.Add("blobs", "true")           // Include blob information
	
	reqURL := fmt.Sprintf("%s%s/%s?%s", c.baseURL, APIEndpoint, modelID, params.Encode())
	
	c.logger.WithFields(logrus.Fields{
		"model_id": modelID,
		"url":      reqURL,
	}).Debug("Fetching model info with full file details")
	
	var model ModelInfo
	
	// Execute request with circuit breaker (v3.0.8)
	_, err := c.circuitBreaker.Execute(func() (any, error) {
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
			return nil, fmt.Errorf("HuggingFace API error: %w", err)
		}
		defer resp.Body.Close()
		
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
		}
		
		// Parse response
		if err := json.NewDecoder(resp.Body).Decode(&model); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		
		return nil, nil
	})
	
	if err != nil {
		return nil, err
	}
	
	// Enrich model with computed fields
	c.enrichModelInfo(&model)
	
	// Fetch README for description (non-blocking, ignore errors)
	if readme, err := c.GetModelReadme(ctx, modelID); err == nil {
		model.Description = extractDescriptionFromReadme(readme)
		c.logger.WithFields(logrus.Fields{
			"model_id":    modelID,
			"desc_length": len(model.Description),
		}).Debug("Extracted model description from README")
	} else {
		c.logger.WithFields(logrus.Fields{
			"model_id": modelID,
			"error":    err.Error(),
		}).Debug("Could not fetch README (optional)")
	}
	
	return &model, nil
}

// enrichModelInfo adds computed fields to model info
func (c *Client) enrichModelInfo(model *ModelInfo) {
	// Filter GGUF files
	ggufFiles := []File{}
	var totalSize int64
	
	c.logger.WithFields(logrus.Fields{
		"model_id":      model.ID,
		"siblings_count": len(model.Siblings),
	}).Debug("Enriching model info")
	
	for i, file := range model.Siblings {
		// Log file details for debugging
		c.logger.WithFields(logrus.Fields{
			"file_index": i,
			"filename":   file.Filename,
			"size":       file.Size,
			"blob_id":    file.BlobID,
			"has_lfs":    file.LFS != nil,
		}).Debug("Processing file")
		
		if file.LFS != nil {
			c.logger.WithFields(logrus.Fields{
				"lfs_size": file.LFS.Size,
				"lfs_oid":  file.LFS.OID,
			}).Debug("LFS data present")
		}
		
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

// GetModelReadme fetches the README content for a model
func (c *Client) GetModelReadme(ctx context.Context, modelID string) (string, error) {
	// README is available at /raw/main/README.md
	readmeURL := fmt.Sprintf("%s/%s/raw/main/README.md", c.baseURL, modelID)
	
	c.logger.WithField("url", readmeURL).Debug("Fetching model README")
	
	req, err := http.NewRequestWithContext(ctx, "GET", readmeURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	
	if c.apiToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiToken))
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch README: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("README not found (status %d)", resp.StatusCode)
	}
	
	// Read README content (limit to 50KB to avoid huge files)
	limitedReader := io.LimitReader(resp.Body, 50*1024)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read README: %w", err)
	}
	
	return string(content), nil
}

// extractDescriptionFromReadme extracts a clean description from README markdown
func extractDescriptionFromReadme(readme string) string {
	if readme == "" {
		return ""
	}
	
	lines := strings.Split(readme, "\n")
	var description strings.Builder
	inFrontMatter := false
	foundContent := false
	lineCount := 0
	maxLines := 20 // Limit description to ~20 lines
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip YAML front matter
		if trimmed == "---" {
			inFrontMatter = !inFrontMatter
			continue
		}
		if inFrontMatter {
			continue
		}
		
		// Skip empty lines at the beginning
		if !foundContent && trimmed == "" {
			continue
		}
		
		// Skip headers (we want prose text)
		if strings.HasPrefix(trimmed, "#") {
			if foundContent {
				// Stop at next header after finding content
				break
			}
			continue
		}
		
		// Skip badges, links to images, etc.
		if strings.HasPrefix(trimmed, "[![") || strings.HasPrefix(trimmed, "![") {
			continue
		}
		
		// Skip HTML comments
		if strings.HasPrefix(trimmed, "<!--") {
			continue
		}
		
		// Skip license agreement blocks
		if strings.Contains(strings.ToLower(trimmed), "license agreement") ||
		   strings.Contains(strings.ToLower(trimmed), "you need to agree") {
			continue
		}
		
		// Found content
		if trimmed != "" {
			foundContent = true
			if description.Len() > 0 {
				description.WriteString("\n")
			}
			description.WriteString(trimmed)
			lineCount++
			
			if lineCount >= maxLines {
				description.WriteString("...")
				break
			}
		}
	}
	
	result := description.String()
	
	// Truncate if too long (max 2000 chars)
	if len(result) > 2000 {
		result = result[:1997] + "..."
	}
	
	return result
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

