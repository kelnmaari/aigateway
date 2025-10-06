// Package main provides simplified Phase 8 demo server
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"ollama-openai-proxy/internal/auth/apikey"
	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/logger"
	"ollama-openai-proxy/internal/models"
	"ollama-openai-proxy/internal/storage"
)

func main() {
	fmt.Println("🔑 Phase 8 Demo - API Key Management System")

	// Простая конфигурация
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "0.0.0.0",
			Port: 8082, // Отдельный порт для demo
		},
		Auth: config.AuthConfig{
			Enabled:     true,
			StorageType: "json",
			StoragePath: "data/demo_api_keys.json",
			AdminKey:    "sk-demo-admin-12345",
		},
		Logging: config.LoggingConfig{
			Level:  "info",
			Format: "text",
			Output: "stdout",
		},
	}

	// Настройка логирования
	appLogger := logger.Setup(cfg)

	// Создание JSON storage
	stor := storage.NewJSONStorage(cfg.Auth.StoragePath, appLogger)

	// Создание API Key Manager
	keyManager := apikey.NewManager(cfg, appLogger, stor)

	// Инициализация
	ctx := context.Background()
	if err := keyManager.Initialize(ctx); err != nil {
		log.Fatalf("Failed to initialize API Key Manager: %v", err)
	}

	// Создание простого HTTP сервера
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// Health endpoint (без аутентификации)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "phase8-demo",
			"port":    8082,
		})
	})

	// Demo endpoints для тестирования API Key Management
	demo := router.Group("/demo")
	{
		// Создание тестового API ключа
		demo.POST("/create-key", func(c *gin.Context) {
			req := models.CreateAPIKeyRequest{
				Name:        "Demo Key " + time.Now().Format("15:04:05"),
				Description: "Demo API key for testing",
				Models:      []string{"*"},
				Permissions: []string{"chat", "models"},
			}

			response, err := keyManager.CreateAPIKey(ctx, req)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(201, response)
		})

		// Список API ключей
		demo.GET("/list-keys", func(c *gin.Context) {
			req := models.ListAPIKeysRequest{Limit: 10}
			response, err := keyManager.ListAPIKeys(ctx, req)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, response)
		})

		// Валидация API ключа
		demo.POST("/validate-key", func(c *gin.Context) {
			var req struct {
				APIKey string `json:"api_key" binding:"required"`
			}

			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}

			result, err := keyManager.ValidateAPIKey(ctx, req.APIKey)
			if err != nil {
				c.JSON(500, gin.H{"error": err.Error()})
				return
			}

			c.JSON(200, result)
		})
	}

	// Запуск сервера
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("🌐 Phase 8 Demo server starting on %s\n", addr)
	fmt.Println("📋 Available endpoints:")
	fmt.Println("  GET  /health - health check")
	fmt.Println("  POST /demo/create-key - create test API key")
	fmt.Println("  GET  /demo/list-keys - list all API keys")
	fmt.Println("  POST /demo/validate-key - validate API key")

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
