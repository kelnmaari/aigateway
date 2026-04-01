package agent

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"aigateway/internal/inference"
	"aigateway/internal/version"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Server is the agent HTTP server that manages local inference containers
// and exposes an API for the main AIGateway server to control them.
type Server struct {
	config    *AgentConfig
	engine    *gin.Engine
	http      *http.Server
	logger    *logrus.Logger
	startedAt time.Time

	// Inference subsystem (reused from main server)
	infService *inference.Service
	infRouter  *inference.Router
}

// NewServer creates an agent server with the given config and inference service.
func NewServer(cfg *AgentConfig, infService *inference.Service, logger *logrus.Logger) *Server {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(gin.Recovery())

	mgr := inference.NewManager(infService)
	router := inference.NewRouter(mgr)

	s := &Server{
		config:     cfg,
		engine:     engine,
		logger:     logger,
		startedAt:  time.Now(),
		infService: infService,
		infRouter:  router,
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Health endpoint — no auth (for load balancers / external health checks)
	s.engine.GET("/api/agent/health", s.handleHealth)

	// Authenticated API routes
	api := s.engine.Group("/api/agent")
	api.Use(APIKeyAuth(s.config.APIKey))
	{
		api.GET("/system", s.handleSystemInfo)
		api.GET("/models", s.handleListModels)
		api.POST("/models/load", s.handleLoadModel)
		api.POST("/models/stop", s.handleStopModel)
		api.DELETE("/models/:alias", s.handleEvictModel)
		api.GET("/models/:alias/logs", s.handleModelLogs)

		// Docker image management
		api.POST("/images/push", s.handleImagePush)   // receive image tar from main server
		api.GET("/images", s.handleListImages)          // list local images
	}

	// OpenAI-compatible inference proxy — authenticated
	// The main server forwards client requests here; the agent proxies
	// to the local container endpoint.
	v1 := s.engine.Group("/v1")
	v1.Use(APIKeyAuth(s.config.APIKey))
	{
		v1.POST("/chat/completions", s.handleInferenceProxy)
		v1.POST("/completions", s.handleInferenceProxy)
		v1.POST("/embeddings", s.handleInferenceProxy)
	}
}

// Start begins listening on the configured address.
func (s *Server) Start() error {
	s.http = &http.Server{
		Addr:              s.config.ListenAddr,
		Handler:           s.engine,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	s.logger.WithFields(logrus.Fields{
		"addr":      s.config.ListenAddr,
		"node_name": s.config.NodeName,
		"node_type": s.config.NodeType,
		"tls":       s.config.TLS.Enabled,
		"version":   version.Version,
	}).Info("Agent server starting")

	if s.config.TLS.Enabled {
		return s.http.ListenAndServeTLS(s.config.TLS.CertFile, s.config.TLS.KeyFile)
	}
	return s.http.ListenAndServe()
}

// Shutdown gracefully stops the server and all inference containers.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Agent server shutting down...")

	// Stop all running containers
	s.infService.StopAll()

	// Shutdown HTTP server
	if s.http != nil {
		return s.http.Shutdown(ctx)
	}
	return nil
}

// InferenceRouter returns the inference router for external access.
func (s *Server) InferenceRouter() *inference.Router {
	return s.infRouter
}

// uptimeSeconds returns seconds since server start.
func (s *Server) uptimeSeconds() int64 {
	return int64(time.Since(s.startedAt).Seconds())
}

// resolveProviderModelName resolves what name the inference provider expects.
// Same logic as InferenceProxyHandler: use HFRepo if available, else LocalPath, else alias.
func resolveProviderModelName(spec inference.ModelSpec) string {
	if spec.HFRepo != "" {
		return spec.HFRepo
	}
	if spec.LocalPath != "" {
		return spec.LocalPath
	}
	return spec.Alias
}

// buildContainerStartRequest constructs the provider-specific container request.
// Reuses the same builder functions as the main server.
func buildContainerStartRequest(spec inference.ModelSpec, hfCacheDir, hfToken string) (inference.ContainerStartRequest, error) {
	switch spec.Provider {
	case inference.ProviderVLLM:
		return inference.BuildVLLMRequest(spec, hfCacheDir, hfToken), nil
	case inference.ProviderSGLang:
		return inference.BuildSGLangRequest(spec, hfCacheDir, hfToken), nil
	case inference.ProviderTGI:
		return inference.BuildTGIRequest(spec, hfCacheDir, hfToken), nil
	case inference.ProviderTEI:
		return inference.BuildTEIRequest(spec, hfCacheDir, hfToken), nil
	case inference.ProviderLlamaCPP:
		return inference.BuildLlamaCPPRequest(spec)
	case inference.ProviderTRTLLM:
		// TRT engines dir defaults to data/engines/trt alongside HF cache
		trtEnginesDir := filepath.Join(filepath.Dir(hfCacheDir), "engines", "trt")
		return inference.BuildTRTLLMRequest(spec, trtEnginesDir, hfToken)
	default:
		return inference.ContainerStartRequest{}, fmt.Errorf("unsupported provider: %s", spec.Provider)
	}
}
