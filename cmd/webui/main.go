package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"aigateway/internal/web"
)

const (
	Version = "1.0.0"
)

var (
	serverURL = flag.String("server-url", "http://localhost:8080", "Ollama-OpenAI Proxy server URL")
	port      = flag.Int("port", 8081, "WebUI server port")
	host      = flag.String("host", "0.0.0.0", "WebUI server host")
)

func main() {
	flag.Parse()

	// Настройка Gin
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(LoggerMiddleware())

	// CORS для разработки
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Встраиваем статические файлы
	staticFS, err := fs.Sub(web.StaticFiles, "static")
	if err != nil {
		log.Fatalf("Failed to load static files: %v", err)
	}

	// Статические файлы
	r.StaticFS("/static", http.FS(staticFS))

	// Главная страница
	r.GET("/", func(c *gin.Context) {
		file, err := web.StaticFiles.ReadFile("static/index.html")
		if err != nil {
			c.String(500, "Error loading page")
			return
		}
		c.Data(200, "text/html; charset=utf-8", file)
	})

	// API proxy endpoints (для обхода CORS)
	api := r.Group("/api")
	{
		// Проксируем запросы к основному серверу
		api.GET("/stats", proxyHandler(*serverURL+"/api/stats"))
		api.GET("/config", proxyHandler(*serverURL+"/api/config"))
		api.GET("/models", proxyHandler(*serverURL+"/api/models"))
		api.GET("/logs", proxyHandler(*serverURL+"/api/logs"))

		// Metrics endpoints для Analytics page
		api.GET("/metrics", proxyHandler(*serverURL+"/metrics"))
		api.GET("/metrics/history", func(c *gin.Context) {
			// Forward query parameters
			query := c.Request.URL.RawQuery
			url := *serverURL + "/api/metrics/history"
			if query != "" {
				url += "?" + query
			}
			proxyHandler(url)(c)
		})
		api.GET("/metrics/recent", func(c *gin.Context) {
			// Forward query parameters
			query := c.Request.URL.RawQuery
			url := *serverURL + "/api/metrics/recent"
			if query != "" {
				url += "?" + query
			}
			proxyHandler(url)(c)
		})
		api.GET("/metrics/stats", func(c *gin.Context) {
			// Forward query parameters
			query := c.Request.URL.RawQuery
			url := *serverURL + "/api/metrics/stats"
			if query != "" {
				url += "?" + query
			}
			proxyHandler(url)(c)
		})
		api.GET("/metrics/stats/all", proxyHandler(*serverURL+"/api/metrics/stats/all"))

		// API Keys management (AUTH-04: Enhanced Key Management)
		api.POST("/admin/keys", proxyHandler(*serverURL+"/api/admin/keys"))
		api.GET("/admin/keys", proxyHandler(*serverURL+"/api/admin/keys"))

		// Single key operations
		api.GET("/admin/keys/:id", func(c *gin.Context) {
			keyID := c.Param("id")
			url := *serverURL + "/api/admin/keys/" + keyID
			proxyHandler(url)(c)
		})
		api.PUT("/admin/keys/:id", func(c *gin.Context) {
			keyID := c.Param("id")
			url := *serverURL + "/api/admin/keys/" + keyID
			proxyHandler(url)(c)
		})
		api.DELETE("/admin/keys/:id", func(c *gin.Context) {
			keyID := c.Param("id")
			url := *serverURL + "/api/admin/keys/" + keyID
			proxyHandler(url)(c)
		})

		// Enhanced key operations (AUTH-04)
		api.PATCH("/admin/keys/:id/revoke", func(c *gin.Context) {
			keyID := c.Param("id")
			url := *serverURL + "/api/admin/keys/" + keyID + "/revoke"
			proxyHandler(url)(c)
		})
		api.PATCH("/admin/keys/:id/enable", func(c *gin.Context) {
			keyID := c.Param("id")
			url := *serverURL + "/api/admin/keys/" + keyID + "/enable"
			proxyHandler(url)(c)
		})
		api.POST("/admin/keys/:id/extend", func(c *gin.Context) {
			keyID := c.Param("id")
			url := *serverURL + "/api/admin/keys/" + keyID + "/extend"
			proxyHandler(url)(c)
		})
		api.PATCH("/admin/keys/:id/permissions", func(c *gin.Context) {
			keyID := c.Param("id")
			url := *serverURL + "/api/admin/keys/" + keyID + "/permissions"
			proxyHandler(url)(c)
		})
	}

	// WebSocket proxy
	r.GET("/ws", proxyWebSocketHandler(*serverURL+"/ws"))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "version": Version})
	})

	// Информация о WebUI
	r.GET("/info", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version":    Version,
			"server_url": *serverURL,
			"port":       *port,
			"host":       *host,
		})
	})

	addr := fmt.Sprintf("%s:%d", *host, *port)

	fmt.Printf("🌐 Ollama-OpenAI Proxy WebUI v%s\n", Version)
	fmt.Printf("📡 Connecting to: %s\n", *serverURL)
	fmt.Printf("🚀 WebUI running on: http://%s:%d\n", *host, *port)
	fmt.Printf("Press Ctrl+C to stop\n\n")

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start WebUI server: %v", err)
	}
}

// proxyHandler создает handler для проксирования запросов к основному серверу
func proxyHandler(targetURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		// Создаем новый запрос
		req, err := http.NewRequest(c.Request.Method, targetURL, c.Request.Body)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to create request"})
			return
		}

		// Копируем заголовки
		for key, values := range c.Request.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		// Добавляем заголовки для обхода антивируса
		req.Header.Set("X-Application", "OllamaProxy-WebUI")
		req.Header.Set("X-Client-Type", "Dashboard")
		if req.Header.Get("User-Agent") == "" {
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		}

		// Выполняем запрос
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(502, gin.H{"error": "Failed to connect to server", "details": err.Error()})
			return
		}
		defer resp.Body.Close()

		// Копируем заголовки ответа
		for key, values := range resp.Header {
			for _, value := range values {
				c.Writer.Header().Add(key, value)
			}
		}

		// Возвращаем ответ
		c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
	}
}

// proxyWebSocketHandler создает handler для проксирования WebSocket соединений
func proxyWebSocketHandler(targetURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse target URL
		target, err := url.Parse(targetURL)
		if err != nil {
			log.Printf("[WebUI] Failed to parse target URL %s: %v", targetURL, err)
			c.JSON(500, gin.H{"error": "Invalid target URL"})
			return
		}

		// Replace http with ws
		target.Scheme = strings.Replace(target.Scheme, "http", "ws", 1)

		// Upgrade client connection
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for now
			},
		}

		clientConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Printf("[WebUI] Failed to upgrade client connection: %v", err)
			return
		}
		defer clientConn.Close()

		// Connect to backend WebSocket
		backendConn, _, err := websocket.DefaultDialer.Dial(target.String(), nil)
		if err != nil {
			log.Printf("[WebUI] Failed to connect to backend %s: %v", target.String(), err)
			clientConn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseInternalServerErr, "Failed to connect to backend"))
			return
		}
		defer backendConn.Close()

		log.Printf("[WebUI] WebSocket proxy established: client <-> %s", target.String())

		// Proxy messages bidirectionally
		errChan := make(chan error, 2)

		// Client -> Backend
		go func() {
			for {
				messageType, message, err := clientConn.ReadMessage()
				if err != nil {
					errChan <- err
					return
				}
				if err := backendConn.WriteMessage(messageType, message); err != nil {
					errChan <- err
					return
				}
			}
		}()

		// Backend -> Client
		go func() {
			for {
				messageType, message, err := backendConn.ReadMessage()
				if err != nil {
					errChan <- err
					return
				}
				if err := clientConn.WriteMessage(messageType, message); err != nil {
					errChan <- err
					return
				}
			}
		}()

		// Wait for error or connection close
		err = <-errChan
		if err != nil && !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
			log.Printf("[WebUI] WebSocket proxy error: %v", err)
		}
	}
}

// LoggerMiddleware - простой логгер для WebUI
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		duration := time.Since(start)
		statusCode := c.Writer.Status()

		// Не логируем статические файлы и health checks
		if path == "/health" || path == "/favicon.ico" ||
			len(path) > 7 && path[:7] == "/static" {
			return
		}

		// Простой цветной вывод
		statusColor := "\033[0m"
		switch {
		case statusCode >= 500:
			statusColor = "\033[31m" // Red
		case statusCode >= 400:
			statusColor = "\033[33m" // Yellow
		case statusCode >= 300:
			statusColor = "\033[36m" // Cyan
		case statusCode >= 200:
			statusColor = "\033[32m" // Green
		}

		fmt.Fprintf(os.Stdout, "%s[WebUI]%s %s %s%d%s %s %v\n",
			"\033[36m", "\033[0m",
			method,
			statusColor, statusCode, "\033[0m",
			path,
			duration,
		)
	}
}

