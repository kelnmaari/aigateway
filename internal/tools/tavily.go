// Package tools provides external tool integrations for LLM assistants.
package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// TavilyClient provides web search via Tavily API.
// https://docs.tavily.com/docs/rest-api/api-reference
type TavilyClient struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// TavilySearchRequest represents a search request to Tavily.
type TavilySearchRequest struct {
	APIKey            string   `json:"api_key"`
	Query             string   `json:"query"`
	SearchDepth       string   `json:"search_depth,omitempty"`        // "basic" or "advanced"
	IncludeAnswer     bool     `json:"include_answer,omitempty"`      // Include AI-generated answer
	IncludeRawContent bool     `json:"include_raw_content,omitempty"` // Include raw HTML content
	MaxResults        int      `json:"max_results,omitempty"`         // Max number of results (default: 5)
	IncludeDomains    []string `json:"include_domains,omitempty"`     // Limit to these domains
	ExcludeDomains    []string `json:"exclude_domains,omitempty"`     // Exclude these domains
}

// TavilySearchResponse represents the search response from Tavily.
type TavilySearchResponse struct {
	Query   string         `json:"query"`
	Answer  string         `json:"answer,omitempty"`
	Results []TavilyResult `json:"results"`
}

// TavilyResult represents a single search result.
type TavilyResult struct {
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
	RawContent string  `json:"raw_content,omitempty"`
}

// NewTavilyClient creates a new Tavily client.
func NewTavilyClient(apiKey string) *TavilyClient {
	return &TavilyClient{
		apiKey:  apiKey,
		baseURL: "https://api.tavily.com",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Search performs a web search using Tavily API.
func (c *TavilyClient) Search(ctx context.Context, query string, maxResults int) (*TavilySearchResponse, error) {
	if maxResults <= 0 {
		maxResults = 5
	}

	reqBody := TavilySearchRequest{
		APIKey:        c.apiKey,
		Query:         query,
		SearchDepth:   "basic",
		IncludeAnswer: true,
		MaxResults:    maxResults,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/search", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("tavily API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result TavilySearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

// FormatResultsAsText formats search results as readable text for LLM context.
func (r *TavilySearchResponse) FormatResultsAsText() string {
	var buf bytes.Buffer

	if r.Answer != "" {
		buf.WriteString("## AI Summary\n")
		buf.WriteString(r.Answer)
		buf.WriteString("\n\n")
	}

	buf.WriteString("## Search Results\n\n")
	for i, result := range r.Results {
		buf.WriteString(fmt.Sprintf("### %d. %s\n", i+1, result.Title))
		buf.WriteString(fmt.Sprintf("URL: %s\n", result.URL))
		buf.WriteString(result.Content)
		buf.WriteString("\n\n")
	}

	return buf.String()
}

