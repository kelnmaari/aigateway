// Package main provides Phase 8 HTTP server with API Key Management
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"ollama-openai-proxy/internal/api/router"
	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/logger"
)

var (
	// Version информация о версии
	Version   = "phase8"
	BuildTime = "unknown"
)

func main() {
	fmt.Printf("🚀 Ollama-OpenAI Proxy Server Phase 8 v%s (built %s)\n", Version, BuildTime)
	fmt.Println("🔑 API Key Management System Enabled")

	// Инициализация конфигурации для Phase 8
	cfg, err := config.Load("configs/phase8.yaml")
	if err != nil {
		log.Fatalf("❌ Не удалось загрузить конфигурацию Phase 8: %v", err)
	}

	// Настройка логирования
	appLogger := logger.Setup(cfg)
	appLogger.WithFields(map[string]interface{}{
		"version": Version,
		"phase":   "8",
		"port":    cfg.Server.Port,
	}).Info("Starting Ollama-OpenAI Proxy Server Phase 8")

	// Создание роутера Phase 8
	appRouter, err := router.NewPhase8Router(cfg, appLogger)
	if err != nil {
		log.Fatalf("❌ Не удалось создать Phase 8 роутер: %v", err)
	}

	// Инициализация API Key Management
	if err := appRouter.Initialize(); err != nil {
		log.Fatalf("❌ Не удалось инициализировать API Key Management: %v", err)
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
		appLogger.WithFields(map[string]interface{}{
			"addr":     server.Addr,
			"phase":    "8",
			"features": []string{"api_keys", "rate_limiting", "auth"},
		}).Info("Starting Phase 8 HTTP server")

		fmt.Printf("🌐 Phase 8 Server запущен на %s\n", server.Addr)
		fmt.Println("🔑 API Key Management: ENABLED")
		fmt.Println("⚡ Rate Limiting: ENABLED")
		fmt.Println("🛡️ Authentication: ENABLED")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.WithError(err).Fatal("Failed to start Phase 8 HTTP server")
		}
	}()

	// Ожидание сигнала остановки
	<-ctx.Done()
	stop()

	appLogger.Info("Phase 8 shutdown signal received, stopping server...")
	fmt.Println("🛑 Получен сигнал остановки Phase 8, завершаем сервер...")

	// Graceful shutdown с таймаутом
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Закрываем Phase 8 компоненты
	if err := appRouter.Close(); err != nil {
		appLogger.WithError(err).Error("Error closing Phase 8 router")
	}

	if err := server.Shutdown(shutdownCtx); err != nil {
		appLogger.WithError(err).Error("Error during Phase 8 server shutdown")
		log.Printf("❌ Ошибка при остановке Phase 8 сервера: %v", err)
	}

	appLogger.Info("Phase 8 server stopped successfully")
	fmt.Println("✅ Phase 8 сервер успешно остановлен")
}
