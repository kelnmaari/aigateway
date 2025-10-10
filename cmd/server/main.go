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

	"ollama-openai-proxy/internal/api/router"
	"ollama-openai-proxy/internal/auth/jwt"
	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/dbfactory"
	"ollama-openai-proxy/internal/logger"
	"ollama-openai-proxy/internal/metrics"
	"ollama-openai-proxy/internal/storage"
	"ollama-openai-proxy/internal/version"
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

	// Инициализация Prometheus метрик (если включено)
	if cfg.Metrics.Enabled {
		metrics.Init("ollama_proxy")
		appLogger.Info("Prometheus metrics initialized")
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

	// Создание роутера
	var appRouter *router.Router
	// Всегда используем NewWithOptions для передачи database (необходим для MCP и других фич)
	appRouter, err = router.NewWithOptions(router.NewOptions{
		Config:     cfg,
		Logger:     appLogger,
		Version:    version.Version,
		Database:   db,         // Может быть nil для legacy mode
		JWTManager: jwtManager, // Может быть nil для legacy mode
	})

	if err != nil {
		log.Fatalf("❌ Не удалось создать роутер: %v", err)
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
