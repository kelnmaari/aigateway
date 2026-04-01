package agent

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"aigateway/internal/inference"
)

// Client communicates with a remote agent node from the main server.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// ClientConfig holds agent client configuration.
type ClientConfig struct {
	BaseURL       string
	APIKey        string
	TLSSkipVerify bool   // for self-signed certs in dev
	CACertPath    string // CA cert for verifying agent TLS
	Timeout       time.Duration
}

// NewClient creates an agent client for communicating with a remote agent.
func NewClient(cfg ClientConfig) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 30 * time.Second
	}

	tlsCfg := &tls.Config{
		InsecureSkipVerify: cfg.TLSSkipVerify, //nolint:gosec // configurable for dev
	}

	// Load CA cert if provided
	if cfg.CACertPath != "" {
		caCert, err := os.ReadFile(cfg.CACertPath)
		if err == nil {
			pool := x509.NewCertPool()
			pool.AppendCertsFromPEM(caCert)
			tlsCfg.RootCAs = pool
		}
	}

	return &Client{
		baseURL: cfg.BaseURL,
		apiKey:  cfg.APIKey,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
			Transport: &http.Transport{
				TLSClientConfig:     tlsCfg,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// ──────────────────────────────────────────────────────────────
// Health & Info
// ──────────────────────────────────────────────────────────────

// Health checks if the agent is alive and responsive.
func (c *Client) Health(ctx context.Context) (*AgentHealthResponse, error) {
	var resp AgentHealthResponse
	if err := c.get(ctx, "/api/agent/health", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// SystemInfo returns GPU, CPU, and memory info from the agent.
func (c *Client) SystemInfo(ctx context.Context) (*AgentSystemInfo, error) {
	var resp AgentSystemInfo
	if err := c.get(ctx, "/api/agent/system", &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ──────────────────────────────────────────────────────────────
// Model Lifecycle
// ──────────────────────────────────────────────────────────────

// LoadModel sends a model spec to the agent for loading.
// Uses a longer timeout since model downloads can take a while.
func (c *Client) LoadModel(ctx context.Context, spec inference.ModelSpec) (*LoadModelResponse, error) {
	// Use a longer timeout for model loading (artifact download + container start)
	loadCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	var resp LoadModelResponse
	if err := c.post(loadCtx, "/api/agent/models/load", LoadModelRequest{ModelSpec: spec}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// StopModel stops a running model on the agent.
func (c *Client) StopModel(ctx context.Context, alias string) error {
	return c.post(ctx, "/api/agent/models/stop", StopModelRequest{Alias: alias}, nil)
}

// EvictModel evicts a model from the agent (stop + remove from registry).
func (c *Client) EvictModel(ctx context.Context, alias string) error {
	return c.delete(ctx, "/api/agent/models/"+alias)
}

// ListModels returns all models tracked by the agent.
func (c *Client) ListModels(ctx context.Context) ([]AgentModelStatus, error) {
	var resp []AgentModelStatus
	if err := c.get(ctx, "/api/agent/models", &resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// PushImage streams a Docker image tar to the agent for loading.
// The reader should be the output of `docker save`.
func (c *Client) PushImage(ctx context.Context, imageName string, reader io.Reader) error {
	url := fmt.Sprintf("%s/api/agent/images/push?image=%s", c.baseURL, imageName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, reader)
	if err != nil {
		return fmt.Errorf("create push request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/octet-stream")

	// Use a longer timeout for image transfers (images can be multi-GB)
	transferClient := &http.Client{
		Timeout: 30 * time.Minute,
		Transport: c.httpClient.Transport,
	}
	resp, err := transferClient.Do(req)
	if err != nil {
		return fmt.Errorf("push image to agent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("agent image push returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ListImages returns Docker images available on the agent.
func (c *Client) ListImages(ctx context.Context) ([]map[string]interface{}, error) {
	var resp struct {
		Images []map[string]interface{} `json:"images"`
	}
	if err := c.get(ctx, "/api/agent/images", &resp); err != nil {
		return nil, err
	}
	return resp.Images, nil
}

// ModelLogs returns container logs for a model on the agent.
func (c *Client) ModelLogs(ctx context.Context, alias string, tailLines int) (string, error) {
	var resp struct {
		Logs string `json:"logs"`
	}
	url := fmt.Sprintf("/api/agent/models/%s/logs?tail=%d", alias, tailLines)
	if err := c.get(ctx, url, &resp); err != nil {
		return "", err
	}
	return resp.Logs, nil
}

// ──────────────────────────────────────────────────────────────
// HTTP helpers
// ──────────────────────────────────────────────────────────────

func (c *Client) get(ctx context.Context, path string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	return c.doRequest(req, result)
}

func (c *Client) post(ctx context.Context, path string, body interface{}, result interface{}) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return c.doRequest(req, result)
}

func (c *Client) delete(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	return c.doRequest(req, nil)
}

func (c *Client) doRequest(req *http.Request, result interface{}) error {
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("agent request %s %s: %w", req.Method, req.URL.Path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.Unmarshal(body, &errResp)
		msg := errResp.Error
		if msg == "" {
			msg = string(body)
		}
		return fmt.Errorf("agent %s %s returned %d: %s", req.Method, req.URL.Path, resp.StatusCode, msg)
	}

	if result != nil && len(body) > 0 {
		if err := json.Unmarshal(body, result); err != nil {
			return fmt.Errorf("unmarshal response: %w", err)
		}
	}
	return nil
}
