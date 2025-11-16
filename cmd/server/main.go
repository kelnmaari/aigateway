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
	"github.com/sirupsen/logrus"

	"aigateway/internal/api/router"
	"aigateway/internal/auth/jwt"
	"aigateway/internal/config"
	"aigateway/internal/dbfactory"
	"aigateway/internal/debug"
	"aigateway/internal/logger"
	"aigateway/internal/metrics"
	"aigateway/internal/observability"
	ragorchestrator "aigateway/internal/rag/orchestrator"
	"aigateway/internal/rag/embeddings"
	"aigateway/internal/rag/vector"
	ragworker "aigateway/internal/rag/worker"
	ragservice "aigateway/internal/services/rag"
	"aigateway/internal/settings"
	"aigateway/internal/storage"
	"aigateway/internal/tls"
	"aigateway/internal/version"
)

func main() {
	// Парсинг флагов
	showVersion := flag.Bool("version", false, "Show version information and exit")
	configPath := flag.String("config", "", "Path to configuration file (default: auto-detect)")

	// Migration management flags (REFACTOR-01)
	rollbackCount := flag.Int("rollback", 0, "Rollback last N migrations")
	rollbackTo := flag.Int("rollback-to", -1, "Rollback to specific migration version")
	showMigrationVersion := flag.Bool("migration-version", false, "Show current migration version")
	listMigrations := flag.Bool("migrations-list", false, "List all migrations with their status")
	destroyDatabase := flag.Bool("destroy", false, "DESTROY database - drops ALL tables (requires confirmation)")

	// Settings management flags (Phase 5: v3.0.9)
	migrateConfig := flag.Bool("migrate-config", false, "Migrate settings from YAML config to database")
	exportConfig := flag.String("export-config", "", "Export database settings to YAML file")
	validateConfig := flag.Bool("validate-config", false, "Validate database settings against YAML schema")
	categoryFilter := flag.String("category", "", "Filter by category (for migrate-config)")

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

	// FlightRecorder для production debugging (v2.4.9+) - Go 1.25 feature
	// Автоматически сохраняет trace последних 30 секунд при panic
	flightRecorder := debug.NewFlightRecorderManager(debug.FlightRecorderConfig{
		Enabled:         true,
		MinAge:          30 * time.Second,
		MaxBytes:        10 * 1024 * 1024, // 10MB
		OutputDir:       "traces",
		AutoSaveOnPanic: true,
	}, appLogger)

	if err := flightRecorder.Start(); err != nil {
		appLogger.WithError(err).Warn("Failed to start FlightRecorder, continuing without it")
	} else {
		defer flightRecorder.Stop()
		appLogger.Info("FlightRecorder started (Go 1.25) - production debugging enabled")
	}

	// Panic recovery с FlightRecorder trace saving
	defer func() {
		if r := recover(); r != nil {
			flightRecorder.SaveTraceOnPanic(context.Background(), r)
			appLogger.WithField("panic", r).Fatal("Application panicked - trace saved to traces/")
			panic(r) // Re-panic после сохранения trace
		}
	}()

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

		// Handle migration management commands (REFACTOR-01)
		if *showMigrationVersion || *listMigrations || *rollbackCount > 0 || *rollbackTo >= 0 || *destroyDatabase {
			handleMigrationCommands(db, appLogger, *showMigrationVersion, *listMigrations, *rollbackCount, *rollbackTo, *destroyDatabase)
			os.Exit(0)
		}

		// Handle settings management commands (Phase 5: v3.0.9)
		if *migrateConfig || *exportConfig != "" || *validateConfig {
			handleSettingsCommands(ctx, db, cfg, appLogger, *migrateConfig, *exportConfig, *validateConfig, *categoryFilter)
			os.Exit(0)
		}
	} else {
		appLogger.Info("Database not configured, skipping initialization")

		// Cannot use migration commands without database
		if *showMigrationVersion || *listMigrations || *rollbackCount > 0 || *rollbackTo >= 0 || *destroyDatabase {
			fmt.Println("❌ Migration commands require database to be configured")
			os.Exit(1)
		}
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
	var ragOrchestrator *ragorchestrator.RAGOrchestrator
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
			
			// Создаем отдельный logger для RAG worker
			ragLogger := logger.NewFileLogger("logs/rag-worker.log", cfg.Logging.Level)
			ragLogger.Info("RAG Worker logger initialized")
			
			// Инициализация Embeddings (Version 1.14.0+)
			var embedder embeddings.Embedder
			var vectorStore vector.VectorStore
			
			if cfg.RAG.Embeddings.Provider != "" {
				appLogger.Info("Initializing Embeddings...")
				
				// Конфигурация embedder
				embedConfig := embeddings.EmbedderConfig{
					Provider:     cfg.RAG.Embeddings.Provider,
					BaseURL:      cfg.RAG.Embeddings.OllamaURL,
					Model:        cfg.RAG.Embeddings.Model,
					Dimensions:   cfg.RAG.Embeddings.Dimensions,
					Timeout:      int(cfg.RAG.Embeddings.Timeout.Seconds()),
					MaxBatchSize: cfg.RAG.Embeddings.BatchSize,
					RateLimit:    10.0, // 10 req/sec
				}
				
				embedder = embeddings.NewOllamaEmbedder(embedConfig, appLogger)
				appLogger.WithFields(map[string]interface{}{
					"provider": embedConfig.Provider,
					"model":    embedConfig.Model,
					"dims":     embedConfig.Dimensions,
				}).Info("✅ Embeddings initialized")
				fmt.Printf("🧮 Embeddings: %s (%s, %d dims)\n", embedConfig.Provider, embedConfig.Model, embedConfig.Dimensions)
				
				// Инициализация Vector Store (pgvector)
				if cfg.RAG.VectorStore.Type == "pgvector" {
					appLogger.Info("Initializing pgvector store...")
					
					// Извлекаем параметры подключения из connection string
					pgConfig := vector.PostgreSQLConfig{
						ConnectionString: cfg.RAG.VectorStore.ConnectionString,
						TableName:        "rag_vectors",
						Dimensions:       cfg.RAG.VectorStore.Dimensions,
						DistanceMetric:   cfg.RAG.VectorStore.DistanceMetric,
						CreateIndex:      true,
						IndexType:        "hnsw", // или ivfflat
					}
					
					vs, err := vector.NewPgVectorStore(pgConfig, appLogger)
					if err != nil {
						appLogger.WithError(err).Warn("Failed to initialize pgvector store, continuing without vector search")
					} else {
						vectorStore = vs
						appLogger.WithFields(map[string]interface{}{
							"table":  pgConfig.TableName,
							"dims":   pgConfig.Dimensions,
							"metric": pgConfig.DistanceMetric,
						}).Info("✅ pgvector store initialized")
						fmt.Printf("🔍 Vector Store: pgvector (%d dims, %s metric)\n", pgConfig.Dimensions, pgConfig.DistanceMetric)
					}
				}
			} else {
				appLogger.Warn("Embeddings not configured, RAG will work in simple mode without similarity search")
			}
			
			// Инициализация RAG Orchestrator для чата (Version 1.14.0+)
			appLogger.Info("Initializing RAG Orchestrator...")
			ragOrchestrator = ragorchestrator.NewRAGOrchestrator(
				db,
				embedder,    // Embedder для генерации query embeddings
				vectorStore, // Vector store для similarity search
				appLogger,
			)
			
			if embedder != nil && vectorStore != nil {
				appLogger.Info("✅ RAG Orchestrator initialized (full mode with embeddings + vector search)")
				fmt.Println("🔍 RAG Orchestrator готов (полный режим с embeddings)")
			} else {
				appLogger.Info("✅ RAG Orchestrator initialized (simple mode)")
				fmt.Println("🔍 RAG Orchestrator готов (упрощенный режим)")
			}
			
			// Запуск RAG Worker для обработки jobs (Version 1.14.0+)
			appLogger.Info("Starting RAG Worker...")
			
			// Создаем и запускаем worker с embedder и vectorStore
			ragWorker := ragworker.NewRAGWorker(db, embedder, vectorStore, ragLogger)
			
			// Запускаем в отдельной goroutine
			go func() {
				workerCtx := context.Background()
				ragWorker.Start(workerCtx)
			}()
			
			// Graceful shutdown для worker
			defer ragWorker.Stop()
			
			if embedder != nil {
				appLogger.Info("✅ RAG Worker started with embeddings support")
				fmt.Println("⚙️  RAG Worker запущен (с embeddings)")
			} else {
				appLogger.Info("✅ RAG Worker started (simple mode)")
				fmt.Println("⚙️  RAG Worker запущен (без embeddings)")
			}
		}
	}

	// Инициализация Model Preloader (Version 1.12.1+)
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
		Database:             db,                   // Может быть nil для legacy mode
		JWTManager:           jwtManager,           // Может быть nil для legacy mode
		TracerProvider:       tracerProvider,       // Может быть nil если tracing отключен (v1.6.0+)
		PerformanceMonitor:   perfMonitor,          // Может быть nil если performance monitoring отключен (v1.6.2+)
		LeakDetector:         leakDetector,         // Может быть nil если leak detection отключен (v1.6.2+)
		MonigoPort:           monigoPort,           // Порт на котором запущен MoniGo (0 если отключен) (v1.9.3+)
		GPUMonitor:           gpuMonitor,           // Может быть nil если NVIDIA GPU не обнаружены (v1.9.3+)
		RAGDataSourceService: ragDataSourceService, // Может быть nil если RAG отключен (v1.13.1+)
		RAGOrchestrator:      ragOrchestrator,      // Может быть nil если RAG отключен (v1.13.1+)
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

	// Auto-seed settings from YAML (v3.0.9 Phase 2)
	if db != nil {
		appLogger.Info("Checking settings database...")
		
		// Create settings storage adapter
		settingsStorage := settings.NewSQLStorageAdapter(db, appLogger)
		seeder := settings.NewConfigSeeder(settingsStorage, appLogger)
		
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		
		seeded, err := seeder.SeedFromYAML(ctx, cfg)
		if err != nil {
			appLogger.WithError(err).Warn("Failed to seed settings from YAML, continuing...")
		} else if seeded > 0 {
			appLogger.WithField("count", seeded).Info("✅ Settings seeded from YAML config")
			fmt.Printf("🌱 Настройки загружены в БД: %d параметров\n", seeded)
		}
	}

	// Создание HTTP сервера
	// HTTP server
	server := &http.Server{
		Addr:           cfg.GetServerAddr(),
		Handler:        appRouter.Engine(),
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    cfg.Server.IdleTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
	}
	
	// HTTPS server (v3.0.8+: Auto-generated self-signed certificate)
	var tlsServer *http.Server
	if cfg.Server.TLS.Enabled {
		// Initialize certificate manager
		certManager := tls.NewCertificateManager("certs", appLogger)
		
		// Prepare certificate configuration from config or defaults
		certConfig := tls.CertificateConfig{
			CommonName:   cfg.Server.TLS.CommonName,
			Organization: "AIGateway",
			ValidFor:     365 * 24 * time.Hour, // Default: 1 year
			Hosts:        cfg.Server.TLS.Hosts,
		}
		
		// Apply defaults if not configured
		if certConfig.CommonName == "" {
			certConfig.CommonName = "localhost"
		}
		if len(certConfig.Hosts) == 0 {
			certConfig.Hosts = []string{"localhost", "127.0.0.1", "::1"}
		}
		if cfg.Server.TLS.ValidDays > 0 {
			certConfig.ValidFor = time.Duration(cfg.Server.TLS.ValidDays) * 24 * time.Hour
		}
		
		// Ensure certificate exists (auto-generate if missing)
		if err := certManager.EnsureCertificate("server.crt", "server.key", certConfig); err != nil {
			appLogger.WithError(err).Fatal("Failed to ensure TLS certificate")
		}
		
		certPath, keyPath := certManager.GetCertificatePaths("server.crt", "server.key")
		
		// Validate certificate
		if err := certManager.ValidateCertificate(certPath); err != nil {
			appLogger.WithError(err).Warn("Certificate validation failed, regenerating...")
			// Delete old certificate and regenerate
			os.Remove(certPath)
			os.Remove(keyPath)
			if err := certManager.EnsureCertificate("server.crt", "server.key", certConfig); err != nil {
				appLogger.WithError(err).Fatal("Failed to regenerate TLS certificate")
			}
		}
		
		// Log certificate info
		if certInfo, err := certManager.GetCertificateInfo(certPath); err == nil {
			appLogger.WithFields(map[string]interface{}{
				"subject":     certInfo["subject"],
				"issuer":      certInfo["issuer"],
				"dns_names":   certInfo["dns_names"],
				"expires_in":  certInfo["expires_in_days"],
				"self_signed": certInfo["is_self_signed"],
			}).Info("TLS certificate loaded")
			
			fmt.Printf("🔐 TLS Certificate: CN=%s, Hosts=%v, Expires in %d days\n", 
				certInfo["subject"], certInfo["dns_names"], certInfo["expires_in_days"])
		}
		
		// Calculate TLS port (HTTP port + 400)
		// Example: :8080 → :8480, :8085 → :8485
		tlsPort := cfg.Server.Port + 400
		tlsAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, tlsPort)
		
		tlsServer = &http.Server{
			Addr:           tlsAddr,
			Handler:        appRouter.Engine(),
			ReadTimeout:    cfg.Server.ReadTimeout,
			WriteTimeout:   cfg.Server.WriteTimeout,
			IdleTimeout:    cfg.Server.IdleTimeout,
			MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
		}
		
		appLogger.WithFields(map[string]interface{}{
			"http_addr":  server.Addr,
			"https_addr": tlsServer.Addr,
			"cert":       certPath,
			"hosts":      certConfig.Hosts,
		}).Info("Dual-listener mode: HTTP + HTTPS")
	}

	// Graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Запуск HTTP сервера в горутине
	go func() {
		appLogger.WithField("addr", server.Addr).Info("Starting HTTP server")
		fmt.Printf("🌐 HTTP сервер запущен на %s\n", server.Addr)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.WithError(err).Fatal("Failed to start HTTP server")
		}
	}()
	
	// Запуск HTTPS сервера в горутине (если включен TLS)
	if tlsServer != nil {
		go func() {
			certPath, keyPath := tls.NewCertificateManager("certs", appLogger).GetCertificatePaths("server.crt", "server.key")
			
			appLogger.WithField("addr", tlsServer.Addr).Info("Starting HTTPS server")
			fmt.Printf("🔐 HTTPS сервер запущен на %s\n", tlsServer.Addr)
			
			if err := tlsServer.ListenAndServeTLS(certPath, keyPath); err != nil && err != http.ErrServerClosed {
				appLogger.WithError(err).Fatal("Failed to start HTTPS server")
			}
		}()
	}

	// Ожидание сигнала остановки
	<-ctx.Done()
	stop()

	appLogger.Info("Shutdown signal received, stopping servers...")
	fmt.Println("🛑 Получен сигнал остановки, завершаем серверы...")

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

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		appLogger.WithError(err).Error("Failed to gracefully shutdown HTTP server")
		log.Printf("❌ Ошибка при остановке HTTP сервера: %v", err)
	} else {
		appLogger.Info("HTTP server stopped gracefully")
	}
	
	// Shutdown HTTPS server (if running)
	if tlsServer != nil {
		if err := tlsServer.Shutdown(shutdownCtx); err != nil {
			appLogger.WithError(err).Error("Failed to gracefully shutdown HTTPS server")
			log.Printf("❌ Ошибка при остановке HTTPS сервера: %v", err)
		} else {
			appLogger.Info("HTTPS server stopped gracefully")
		}
	}

	appLogger.Info("All servers stopped successfully")
	fmt.Println("✅ Все серверы успешно остановлены")
}

// handleMigrationCommands handles migration management CLI commands (REFACTOR-01)
func handleMigrationCommands(db storage.Database, appLogger interface{},
	showVersion, listMigs bool, rollbackCount, rollbackTo int, destroy bool) {

	ctx := context.Background()

	// Show current migration version
	if showVersion {
		version, err := db.GetMigrationVersion(ctx)
		if err != nil {
			fmt.Printf("❌ Error getting migration version: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("📊 Current migration version: %d\n", version)
		return
	}

	// List all migrations with their status
	if listMigs {
		migrations, err := db.ListMigrations(ctx)
		if err != nil {
			fmt.Printf("❌ Error listing migrations: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("📋 Migrations list:")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("%-7s %-50s %-10s %-10s\n", "Version", "Name", "Status", "Rollback")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		for _, m := range migrations {
			status := "❌ Pending"
			if m.Applied {
				status = "✅ Applied"
			}

			rollback := "✅ Yes"
			if !m.HasRollback {
				rollback = "❌ No"
			} else if m.Irreversible {
				rollback = "⚠️  IRREVERSIBLE"
			}

			fmt.Printf("%-7d %-50s %-10s %-10s\n", m.Version, m.Name, status, rollback)
		}

		currentVersion, _ := db.GetMigrationVersion(ctx)
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Printf("Current version: %d | Total migrations: %d\n", currentVersion, len(migrations))
		return
	}

	// Rollback last N migrations
	if rollbackCount > 0 {
		currentVersion, err := db.GetMigrationVersion(ctx)
		if err != nil {
			fmt.Printf("❌ Error getting migration version: %v\n", err)
			os.Exit(1)
		}

		targetVersion := currentVersion - rollbackCount
		if targetVersion < 0 {
			targetVersion = 0
		}

		fmt.Printf("🔄 Rolling back last %d migrations (from v%d to v%d)...\n",
			rollbackCount, currentVersion, targetVersion)

		if err := db.RollbackMigrations(ctx, targetVersion); err != nil {
			fmt.Printf("❌ Rollback failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Successfully rolled back to version %d\n", targetVersion)
		return
	}

	// Rollback to specific version
	if rollbackTo >= 0 {
		currentVersion, err := db.GetMigrationVersion(ctx)
		if err != nil {
			fmt.Printf("❌ Error getting migration version: %v\n", err)
			os.Exit(1)
		}

		if rollbackTo >= currentVersion {
			fmt.Printf("❌ Target version (%d) must be less than current version (%d)\n",
				rollbackTo, currentVersion)
			os.Exit(1)
		}

		fmt.Printf("🔄 Rolling back to version %d (from v%d)...\n", rollbackTo, currentVersion)

		if err := db.RollbackMigrations(ctx, rollbackTo); err != nil {
			fmt.Printf("❌ Rollback failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Successfully rolled back to version %d\n", rollbackTo)
		return
	}

	// Destroy database
	if destroy {
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("⚠️  WARNING: DESTRUCTIVE OPERATION")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Println("This will PERMANENTLY DELETE:")
		fmt.Println("  - All tables")
		fmt.Println("  - All data (users, conversations, API keys, etc)")
		fmt.Println("  - All migrations")
		fmt.Println("")
		fmt.Println("A backup will be created in data/backups/ before destruction")
		fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Print("\nType 'YES' in CAPITALS to confirm destruction: ")

		var confirmation string
		fmt.Scanln(&confirmation)

		if confirmation != "YES" {
			fmt.Println("❌ Destruction cancelled")
			os.Exit(0)
		}

		fmt.Println("\n🔥 Destroying database...")

		if err := db.DestroyDatabase(ctx); err != nil {
			fmt.Printf("❌ Destruction failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("✅ Database destroyed successfully")
		fmt.Println("💡 To start fresh, run the server - migrations will be applied automatically")
		return
	}
}

// handleSettingsCommands handles settings management CLI commands (Phase 5: v3.0.9)
func handleSettingsCommands(
	ctx context.Context,
	db storage.Database,
	cfg *config.Config,
	logger *logrus.Logger,
	migrateConfig bool,
	exportConfig string,
	validateConfig bool,
	categoryFilter string,
) {
	// Create settings storage adapter
	settingsStorage := settings.NewSQLStorageAdapter(db, logger)

	// Migrate config
	if migrateConfig {
		if err := settings.MigrateCommand(ctx, cfg, settingsStorage, logger, categoryFilter); err != nil {
			fmt.Printf("❌ Migration failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Export config
	if exportConfig != "" {
		if err := settings.ExportCommand(ctx, settingsStorage, logger, exportConfig); err != nil {
			fmt.Printf("❌ Export failed: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Validate config
	if validateConfig {
		if err := settings.ValidateCommand(ctx, cfg, settingsStorage, logger); err != nil {
			fmt.Printf("❌ Validation failed: %v\n", err)
			os.Exit(1)
		}
		return
	}
}
