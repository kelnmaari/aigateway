// Package main provides the HTTP server entry point for Ollama-OpenAI Proxy
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/iyashjayesh/monigo"

	"aigateway/internal/api/router"
	"aigateway/internal/auth/jwt"
	"aigateway/internal/client/ollama"
	"aigateway/internal/config"
	"aigateway/internal/dbfactory"
	"aigateway/internal/logger"
	"aigateway/internal/metrics"
	"aigateway/internal/observability"
	"aigateway/internal/services/model"
	ragservice "aigateway/internal/services/rag"
	"aigateway/internal/storage"
	"aigateway/internal/version"
)

func main() {
	// Парсинг флагов
	showVersion := flag.Bool("version", false, "Show version information and exit")
	configPath := flag.String("config", "", "Path to configuration file (default: auto-detect)")
	flag.Parse()

	// Показываем версию и выходим если запрошено
	if *showVersion {
		versionInfo := version.GetInfo()
		fmt.Printf("%s\n", versionInfo.String())
		os.Exit(0)
	}

	fmt.Printf("🚀 Ollama-OpenAI Proxy Server v%s\n", version.Short())

	// Инициализация конфигурации
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("❌ Не удалось загрузить конфигурацию: %v", err)
	}

	// Логируем источник конфигурации
	if *configPath != "" {
		fmt.Printf("📋 Configuration loaded from: %s\n", *configPath)
	} else {
		fmt.Printf("📋 Configuration auto-detected\n")
	}

	// Настройка логирования
	appLogger := logger.Setup(cfg)
	appLogger.WithField("version", version.Version).
		WithField("git_commit", version.GitCommit).
		WithField("build_date", version.BuildDate).
		Info("Starting Ollama-OpenAI Proxy Server")

	// Prometheus метрики (если включено)
	// Metrics are already registered globally via promauto in prometheus.go
	if cfg.Metrics.Enabled {
		appLogger.Info("Prometheus metrics enabled")
	}

	// Инициализация базы данных (Version 1.3.0+)
	var db storage.Database
	if cfg.Database.Type != "" {
		appLogger.Info("Initializing database...")

		db, err = dbfactory.NewDatabase(cfg, appLogger)
		if err != nil {
			log.Fatalf("❌ Не удалось создать базу данных: %v", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := dbfactory.InitializeDatabase(ctx, db, appLogger); err != nil {
			log.Fatalf("❌ Не удалось инициализировать базу данных: %v", err)
		}

		appLogger.Info("✅ Database initialized successfully")
		fmt.Println("💾 База данных инициализирована")
	} else {
		appLogger.Info("Database not configured, skipping initialization")
	}

	// Инициализация JWT Manager (Version 1.3.0+)
	var jwtManager *jwt.Manager
	if db != nil && cfg.Auth.JWT.Secret != "" {
		appLogger.Info("Initializing JWT Manager...")

		jwtManager, err = jwt.NewManager(jwt.Config{
			SecretKey:            cfg.Auth.JWT.Secret,
			AccessTokenDuration:  cfg.Auth.JWT.AccessTokenExpiry,
			RefreshTokenDuration: cfg.Auth.JWT.RefreshTokenExpiry,
			Logger:               appLogger,
		})
		if err != nil {
			log.Fatalf("❌ Не удалось создать JWT Manager: %v", err)
		}

		appLogger.Info("✅ JWT Manager initialized successfully")
		fmt.Println("🔐 JWT Manager инициализирован")
	}

	// Инициализация OpenTelemetry Tracing (Version 1.6.0+)
	var tracerProvider *observability.TracerProvider
	if cfg.Observability.Tracing.Enabled {
		appLogger.Info("Initializing OpenTelemetry Tracing...")

		// Определяем endpoint в зависимости от provider
		endpoint := ""
		switch cfg.Observability.Tracing.Provider {
		case "jaeger":
			endpoint = cfg.Observability.Tracing.Jaeger.Endpoint
		case "zipkin":
			endpoint = cfg.Observability.Tracing.Zipkin.Endpoint
		}

		tracerProvider, err = observability.NewTracerProvider(observability.TracingConfig{
			Enabled:      true,
			Provider:     cfg.Observability.Tracing.Provider,
			ServiceName:  cfg.Observability.Tracing.ServiceName,
			Endpoint:     endpoint,
			SamplingRate: cfg.Observability.Tracing.SamplingRate,
		})
		if err != nil {
			log.Fatalf("❌ Не удалось инициализировать OpenTelemetry: %v", err)
		}

		// Graceful shutdown для tracer provider
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
				appLogger.WithError(err).Error("Error shutting down tracer provider")
			}
		}()

		appLogger.WithFields(map[string]interface{}{
			"provider":      cfg.Observability.Tracing.Provider,
			"endpoint":      endpoint,
			"sampling_rate": cfg.Observability.Tracing.SamplingRate,
		}).Info("✅ OpenTelemetry Tracing initialized successfully")
		fmt.Println("🔍 OpenTelemetry трассировка включена")
	}

	// Инициализация Performance Monitor (Version 1.6.2+)
	var perfMonitor *observability.PerformanceMonitor
	var leakDetector *observability.LeakDetector
	if cfg.Observability.Performance.Enabled {
		appLogger.Info("Initializing Performance Monitor...")

		// Parse durations
		collectionInterval, err := time.ParseDuration(cfg.Observability.Performance.CollectionInterval)
		if err != nil {
			appLogger.WithError(err).Warn("Invalid collection interval, using default 30s")
			collectionInterval = 30 * time.Second
		}

		slowRequestThreshold, err := time.ParseDuration(cfg.Observability.Performance.SlowRequestThreshold)
		if err != nil {
			appLogger.WithError(err).Warn("Invalid slow request threshold, using default 5s")
			slowRequestThreshold = 5 * time.Second
		}

		perfMonitor = observability.NewPerformanceMonitor(appLogger, observability.PerfMonConfig{
			Enabled:              true,
			CollectionInterval:   collectionInterval,
			MemoryThresholdMB:    cfg.Observability.Performance.MemoryThresholdMB,
			GoroutineThreshold:   cfg.Observability.Performance.GoroutineThreshold,
			SlowRequestThreshold: slowRequestThreshold,
			GCPercentage:         cfg.Observability.Performance.GCPercentage,
		})

		// Start performance monitor in background
		monitorCtx, monitorCancel := context.WithCancel(context.Background())
		defer monitorCancel()
		go perfMonitor.Start(monitorCtx)

		// Graceful shutdown
		defer perfMonitor.Stop()

		appLogger.WithFields(map[string]interface{}{
			"collection_interval":    collectionInterval,
			"memory_threshold_mb":    cfg.Observability.Performance.MemoryThresholdMB,
			"goroutine_threshold":    cfg.Observability.Performance.GoroutineThreshold,
			"slow_request_threshold": slowRequestThreshold,
		}).Info("✅ Performance Monitor initialized successfully")
		fmt.Println("📊 Performance мониторинг включен")

		// Initialize Leak Detector if enabled
		if cfg.Observability.Performance.LeakDetection {
			appLogger.Info("Initializing Leak Detector...")
			leakDetector = observability.NewLeakDetector(appLogger)

			// Start leak detector in background
			leakCtx, leakCancel := context.WithCancel(context.Background())
			defer leakCancel()
			go leakDetector.Start(leakCtx)

			appLogger.Info("✅ Leak Detector initialized successfully")
			fmt.Println("🔍 Leak Detection включен")
		}
	}

	// Инициализация GPU Monitor (Version 1.9.3+)
	var gpuMonitor *metrics.GPUMonitor
	gpuMonitor, err = metrics.NewGPUMonitor(appLogger, 10*time.Second)
	if err != nil {
		appLogger.WithError(err).Error("Failed to initialize GPU monitor")
		// Не критичная ошибка, продолжаем
	} else if gpuMonitor != nil {
		gpuMonitor.Start()
		defer gpuMonitor.Stop()
		appLogger.Info("✅ GPU Monitor initialized successfully")
		fmt.Println("🎮 NVIDIA GPU мониторинг включен")
	}

	// Инициализация MoniGo Performance Dashboard (Version 1.9.3+)
	var monigoInstance *monigo.Monigo
	var customMetrics *metrics.CustomMonigoMetrics
	var monigoPort int = 9091 // Отдельный порт для MoniGo dashboard (9091 чтобы избежать конфликта с hwinfo)

	if db != nil && jwtManager != nil {
		appLogger.Info("Initializing MoniGo Performance Dashboard...")

		// Создаем MoniGo instance
		monigoInstance = &monigo.Monigo{
			ServiceName: "Ollama-OpenAI-Proxy",
		}

		// Инициализируем MoniGo
		appLogger.Debug("Calling MoniGo.Initialize()...")
		monigoInstance.Initialize()
		appLogger.Debug("MoniGo.Initialize() completed")

		// Запускаем отдельный HTTP сервер для MoniGo на порту 9090
		monigoMux := http.NewServeMux()

		// Получаем unified handler от MoniGo
		monigoHandler := monigo.GetUnifiedHandler()
		monigoMux.Handle("/", monigoHandler)

		monigoServer := &http.Server{
			Addr:    fmt.Sprintf(":%d", monigoPort),
			Handler: monigoMux,
		}

		// Запускаем MoniGo сервер в отдельной горутине
		go func() {
			appLogger.WithField("port", monigoPort).Info("Starting MoniGo HTTP server...")
			if err := monigoServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				appLogger.WithError(err).Error("MoniGo server failed")
			}
		}()

		// Graceful shutdown для MoniGo сервера
		defer func() {
			appLogger.Info("Shutting down MoniGo server...")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := monigoServer.Shutdown(ctx); err != nil {
				appLogger.WithError(err).Error("Failed to shutdown MoniGo server")
			}
		}()

		// Запускаем custom metrics collector
		customMetrics = metrics.NewCustomMonigoMetrics(monigoInstance, db, cfg, appLogger)
		customMetrics.Start()

		// Graceful shutdown для custom metrics
		defer customMetrics.Stop()

		appLogger.WithFields(map[string]interface{}{
			"service_name": "Ollama-OpenAI-Proxy",
			"monigo_port":  monigoPort,
			"proxy_path":   "/admin/performance",
		}).Info("✅ MoniGo Performance Dashboard initialized successfully")
		fmt.Printf("📊 MoniGo Dashboard: http://localhost:%d (metrics API proxy: /admin/performance/monigo/api/v1/metrics)\n", monigoPort)
	} else {
		appLogger.WithFields(map[string]interface{}{
			"db_initialized":  db != nil,
			"jwt_initialized": jwtManager != nil,
		}).Warn("MoniGo Performance Dashboard NOT initialized - missing dependencies")
		fmt.Println("⚠️  MoniGo Performance Dashboard НЕ включен (требуется database + JWT)")
	}

	// Инициализация RAG Data Source Service (Version 1.13.1+)
	var ragDataSourceService *ragservice.DataSourceService
	if cfg.RAG.Enabled && db != nil {
		appLogger.Info("Initializing RAG Data Source Service...")
		
		// Encryption key для credentials (32 bytes для AES-256)
		encryptionKey := cfg.RAG.Security.EncryptionKey
		if encryptionKey == "" {
			appLogger.Warn("RAG encryption key not set, using default (NOT SECURE FOR PRODUCTION)")
			encryptionKey = "12345678901234567890123456789012" // 32 bytes placeholder
		}
		
		ragDataSourceService, err = ragservice.NewDataSourceService(
			db, // DB implements RAGDataSourceRepository
			encryptionKey,
			appLogger,
		)
		if err != nil {
			appLogger.WithError(err).Warn("Failed to initialize RAG Data Source Service")
		} else {
			appLogger.Info("✅ RAG Data Source Service initialized successfully")
			fmt.Println("🧠 RAG System включен")
		}
	}

	// Инициализация Model Preloader (Version 1.12.1+)
	var modelPreloader *model.ModelPreloader
	if cfg.Models.Preload.Enabled {
		appLogger.Info("Initializing Model Preloader...")

		// Создаем Ollama client для preloader
		ollamaClient, err := ollama.NewClient(cfg, appLogger)
		if err != nil {
			appLogger.WithError(err).Warn("Failed to create Ollama client for preloader, continuing without preloading")
		} else {
			modelPreloader = model.NewModelPreloader(
				ollamaClient,
				&cfg.Models.Preload,
				appLogger,
			)

			// Start preloader (async, non-blocking)
			preloadCtx := context.Background()
			if err := modelPreloader.Start(preloadCtx); err != nil {
				appLogger.WithError(err).Warn("Failed to start model preloader")
			} else {
				appLogger.WithFields(map[string]interface{}{
					"models":                len(cfg.Models.Preload.Models),
					"on_startup":            cfg.Models.Preload.OnStartup,
					"keep_warm":             cfg.Models.Preload.KeepWarm,
					"health_check_interval": cfg.Models.Preload.HealthCheckInterval,
				}).Info("✅ Model Preloader initialized successfully")
				fmt.Println("🔥 Model Preloading включен")
			}
		}
	}

	// Создание роутера
	var appRouter *router.Router
	// Всегда используем NewWithOptions для передачи database (необходим для MCP и других фич)

	// DEBUG: Проверяем что MoniGo настроен корректно
	appLogger.WithFields(map[string]interface{}{
		"monigo_enabled": monigoInstance != nil,
		"monigo_port":    monigoPort,
	}).Debug("Creating router with MoniGo configuration")

	appRouter, err = router.NewWithOptions(router.NewOptions{
		Config:               cfg,
		Logger:               appLogger,
		Version:              version.Version,
		Database:             db,                    // Может быть nil для legacy mode
		JWTManager:           jwtManager,            // Может быть nil для legacy mode
		TracerProvider:       tracerProvider,        // Может быть nil если tracing отключен (v1.6.0+)
		PerformanceMonitor:   perfMonitor,           // Может быть nil если performance monitoring отключен (v1.6.2+)
		LeakDetector:         leakDetector,          // Может быть nil если leak detection отключен (v1.6.2+)
		MonigoPort:           monigoPort,            // Порт на котором запущен MoniGo (0 если отключен) (v1.9.3+)
		GPUMonitor:           gpuMonitor,            // Может быть nil если NVIDIA GPU не обнаружены (v1.9.3+)
		ModelPreloader:       modelPreloader,        // Может быть nil если preloading отключен (v1.12.1+)
		RAGDataSourceService: ragDataSourceService,  // Может быть nil если RAG отключен (v1.13.1+)
	})

	if err != nil {
		log.Fatalf("❌ Не удалось создать роутер: %v", err)
	}

	// Start Prometheus Metrics Collector (v1.11.6+)
	if cfg.Metrics.Enabled && appRouter.GetMetricsCollector() != nil {
		go appRouter.GetMetricsCollector().Start(context.Background())
		appLogger.Info("Prometheus metrics collector started")
	}

	// Инициализация роутера (включая API Key Management если включен)
	if err := appRouter.Initialize(); err != nil {
		log.Fatalf("❌ Не удалось инициализировать роутер: %v", err)
	}

	// Создание HTTP сервера
	server := &http.Server{
		Addr:           cfg.GetServerAddr(),
		Handler:        appRouter.Engine(),
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Запуск сервера в горутине
	go func() {
		appLogger.WithField("addr", server.Addr).Info("Starting HTTP server")
		fmt.Printf("🌐 Сервер запущен на %s\n", server.Addr)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()

	// Ожидание сигнала остановки
	<-ctx.Done()
	stop()

	appLogger.Info("Shutdown signal received, stopping server...")
	fmt.Println("🛑 Получен сигнал остановки, завершаем сервер...")

	// Graceful shutdown с таймаутом
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Закрываем роутер и освобождаем ресурсы
	if err := appRouter.Close(); err != nil {
		appLogger.WithError(err).Error("Error closing router")
	}

	// Закрываем базу данных
	if db != nil {
		if err := dbfactory.CloseDatabase(db, appLogger); err != nil {
			appLogger.WithError(err).Error("Error closing database")
		}
	}

	if err := server.Shutdown(shutdownCtx); err != nil {
		appLogger.WithError(err).Error("Error during server shutdown")
		log.Printf("❌ Ошибка при остановке сервера: %v", err)
	}

	appLogger.Info("Server stopped successfully")
	fmt.Println("✅ Сервер успешно остановлен")
}

