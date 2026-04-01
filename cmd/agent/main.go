package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aigateway/internal/agent"
	"aigateway/internal/inference"
	"aigateway/internal/version"

	"github.com/sirupsen/logrus"
)

func main() {
	configPath := flag.String("config", "configs/agent.yaml", "Path to agent config file")
	showVersion := flag.Bool("version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("aigateway-agent %s (built %s, commit %s)\n",
			version.Version, version.BuildDate, version.GitCommit)
		os.Exit(0)
	}

	// Logger
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
	})
	logger.SetLevel(logrus.InfoLevel)

	logger.WithFields(logrus.Fields{
		"version": version.Version,
		"config":  *configPath,
	}).Info("AIGateway Agent starting")

	// Load config
	cfg, err := agent.LoadAgentConfig(*configPath)
	if err != nil {
		logger.WithError(err).Fatal("Failed to load agent config")
	}

	logger.WithFields(logrus.Fields{
		"node_name":  cfg.NodeName,
		"node_type":  cfg.NodeType,
		"listen":     cfg.ListenAddr,
		"hf_cache":   cfg.HFCacheDir,
		"gguf_cache": cfg.GGUFCacheDir,
	}).Info("Agent config loaded")

	// Create directories
	_ = os.MkdirAll(cfg.HFCacheDir, 0755)
	_ = os.MkdirAll(cfg.GGUFCacheDir, 0755)

	// Build inference service (same as main server, but minimal)
	svcCfg := inference.ServiceConfig{
		HFToken:            cfg.HFToken,
		HFCacheDir:         cfg.HFCacheDir,
		GGUFCacheDir:       cfg.GGUFCacheDir,
		ContainerLogsDir:   "logs/containers",
		DockerBin:          cfg.DockerBin,
		MaxRunningModels:   cfg.MaxRunningModels,
		HealthCheckTimeout: cfg.HealthCheckTimeout,
		StartupTimeout:     cfg.StartupTimeout,
		Logger:             logger,
	}

	infService, err := inference.NewService(svcCfg)
	if err != nil {
		logger.WithError(err).Fatal("Failed to create inference service")
	}

	// Login to configured Docker registries (for private image pulls)
	if len(cfg.DockerRegistries) > 0 {
		runtime := infService.GetDockerRuntime()
		if runtime != nil {
			for _, reg := range cfg.DockerRegistries {
				logger.WithField("registry", reg.Registry).Info("Logging in to Docker registry")
				if err := runtime.DockerLogin(reg.Registry, reg.Username, reg.Password); err != nil {
					logger.WithError(err).WithField("registry", reg.Registry).Warn("Docker registry login failed")
				} else {
					logger.WithField("registry", reg.Registry).Info("Docker registry login successful")
				}
			}
		}
	}

	// Create agent server
	srv := agent.NewServer(cfg, infService, logger)

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := srv.Start(); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("Shutdown signal received")
	case err := <-errCh:
		logger.WithError(err).Fatal("Agent server failed")
	}

	// Shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.WithError(err).Error("Agent shutdown error")
	}

	logger.Info("Agent stopped")
}
