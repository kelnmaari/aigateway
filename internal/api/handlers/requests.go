// Package handlers provides HTTP request handlers
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	"ollama-openai-proxy/internal/config"
	"ollama-openai-proxy/internal/request"
)

// RequestsHandler обрабатывает запросы для request monitoring (TUI-04)
type RequestsHandler struct {
	config  *config.Config
	logger  *logrus.Logger
	storage *request.Storage
}

// NewRequestsHandler создает новый handler для requests
func NewRequestsHandler(cfg *config.Config, logger *logrus.Logger, storage *request.Storage) *RequestsHandler {
	return &RequestsHandler{
		config:  cfg,
		logger:  logger,
		storage: storage,
	}
}

// ListRequests возвращает список запросов с фильтрацией и пагинацией
// GET /api/requests?status=success&endpoint=/v1/chat/completions&model=gpt-3.5-turbo&sort=time&order=desc&page=1&per_page=50
func (h *RequestsHandler) ListRequests(c *gin.Context) {
	// Получаем query parameters
	statusFilter := c.Query("status")
	endpointFilter := c.Query("endpoint")
	modelFilter := c.Query("model")
	sortField := c.DefaultQuery("sort", "time")
	order := c.DefaultQuery("order", "desc")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "50"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 500 {
		perPage = 50
	}

	// Применяем фильтры
	var filters []request.FilterFunc

	if statusFilter != "" {
		filters = append(filters, request.FilterByStatus(request.Status(statusFilter)))
	}

	if endpointFilter != "" {
		filters = append(filters, request.FilterByEndpoint(endpointFilter))
	}

	if modelFilter != "" {
		filters = append(filters, request.FilterByModel(modelFilter))
	}

	// Получаем отфильтрованные запросы
	requests := h.storage.Filter(filters...)

	// Сортируем
	sortFieldEnum := request.SortByTime
	switch sortField {
	case "duration":
		sortFieldEnum = request.SortByDuration
	case "status":
		sortFieldEnum = request.SortByStatus
	case "endpoint":
		sortFieldEnum = request.SortByEndpoint
	}

	descending := order == "desc"
	requests = h.storage.Sort(requests, sortFieldEnum, descending)

	// Пагинация
	total := len(requests)
	startIdx := (page - 1) * perPage
	endIdx := startIdx + perPage

	if startIdx >= total {
		requests = []*request.RequestInfo{}
	} else {
		if endIdx > total {
			endIdx = total
		}
		requests = requests[startIdx:endIdx]
	}

	// Формируем ответ
	c.JSON(http.StatusOK, gin.H{
		"requests": requests,
		"pagination": gin.H{
			"page":     page,
			"per_page": perPage,
			"total":    total,
			"pages":    (total + perPage - 1) / perPage,
		},
	})
}

// GetRequest возвращает детальную информацию о запросе по ID
// GET /api/requests/:id
func (h *RequestsHandler) GetRequest(c *gin.Context) {
	requestID := c.Param("id")

	req := h.storage.Get(requestID)
	if req == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Request not found",
		})
		return
	}

	c.JSON(http.StatusOK, req)
}

// GetRequestStats возвращает статистику по запросам
// GET /api/requests/stats
func (h *RequestsHandler) GetRequestStats(c *gin.Context) {
	stats := h.storage.GetStats()
	c.JSON(http.StatusOK, stats)
}

// ClearRequests очищает историю запросов
// DELETE /api/requests
func (h *RequestsHandler) ClearRequests(c *gin.Context) {
	h.storage.Clear()

	h.logger.Info("Request history cleared")

	c.JSON(http.StatusOK, gin.H{
		"message": "Request history cleared successfully",
	})
}
