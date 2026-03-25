// Package main provides metrics parsing functionality for TUI
package main

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// MetricsSnapshot хранит snapshot Prometheus метрик
type MetricsSnapshot struct {
	// HTTP Metrics
	HTTPRequestsTotal    map[string]float64 // по endpoint
	HTTPRequestDuration  map[string]float64 // средняя latency по endpoint
	HTTPRequestsInFlight float64

	// Ollama Metrics
	OllamaRequestsTotal     map[string]float64 // по model
	OllamaRequestDuration   map[string]float64 // средняя latency по model
	OllamaErrorsTotal       map[string]float64 // по error_type
	OllamaConnectionsActive float64

	// API Key Metrics
	APIKeyRequestsTotal     map[string]float64 // по key_id
	APIKeyRateLimitExceeded map[string]float64 // по key_id
	APIKeysActiveTotal      float64
	APIKeyTokensUsedTotal   map[string]float64 // по key_id

	// Circuit Breaker Metrics
	CircuitBreakerState      map[string]float64 // по breaker_name
	CircuitBreakerTripsTotal map[string]float64 // по breaker_name
}

// NewMetricsSnapshot создает пустой snapshot
func NewMetricsSnapshot() *MetricsSnapshot {
	return &MetricsSnapshot{
		HTTPRequestsTotal:        make(map[string]float64),
		HTTPRequestDuration:      make(map[string]float64),
		OllamaRequestsTotal:      make(map[string]float64),
		OllamaRequestDuration:    make(map[string]float64),
		OllamaErrorsTotal:        make(map[string]float64),
		APIKeyRequestsTotal:      make(map[string]float64),
		APIKeyRateLimitExceeded:  make(map[string]float64),
		APIKeyTokensUsedTotal:    make(map[string]float64),
		CircuitBreakerState:      make(map[string]float64),
		CircuitBreakerTripsTotal: make(map[string]float64),
	}
}

// ParsePrometheusMetrics парсит Prometheus текстовый формат
func ParsePrometheusMetrics(reader io.Reader) (*MetricsSnapshot, error) {
	snapshot := NewMetricsSnapshot()
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Пропускаем комментарии и пустые строки
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Парсим метрику
		if err := parseMetricLine(line, snapshot); err != nil {
			// Логируем ошибку, но продолжаем
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading metrics: %w", err)
	}

	return snapshot, nil
}

// parseMetricLine парсит одну строку метрики
func parseMetricLine(line string, snapshot *MetricsSnapshot) error {
	// Формат: metric_name{label1="value1",label2="value2"} value timestamp
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return fmt.Errorf("invalid metric line: %s", line)
	}

	metricPart := parts[0]
	valueStr := parts[1]

	// Парсим значение
	value, err := strconv.ParseFloat(valueStr, 64)
	if err != nil {
		return fmt.Errorf("invalid metric value: %s", valueStr)
	}

	// Разбираем имя метрики и labels
	metricName, labels := parseMetricNameAndLabels(metricPart)

	// Распределяем по snapshot
	switch {
	case strings.HasPrefix(metricName, "ollama_proxy_http_requests_total"):
		endpoint := labels["endpoint"]
		if endpoint != "" {
			snapshot.HTTPRequestsTotal[endpoint] = value
		}

	case strings.HasPrefix(metricName, "ollama_proxy_http_request_duration_seconds_sum"):
		endpoint := labels["endpoint"]
		if endpoint != "" {
			snapshot.HTTPRequestDuration[endpoint] = value
		}

	case strings.HasPrefix(metricName, "ollama_proxy_http_requests_in_flight"):
		snapshot.HTTPRequestsInFlight += value

	case strings.HasPrefix(metricName, "ollama_proxy_ollama_requests_total"):
		model := labels["model"]
		operation := labels["operation"]
		key := fmt.Sprintf("%s:%s", model, operation)
		snapshot.OllamaRequestsTotal[key] = value

	case strings.HasPrefix(metricName, "ollama_proxy_ollama_request_duration_seconds_sum"):
		model := labels["model"]
		operation := labels["operation"]
		key := fmt.Sprintf("%s:%s", model, operation)
		snapshot.OllamaRequestDuration[key] = value

	case strings.HasPrefix(metricName, "ollama_proxy_ollama_errors_total"):
		errorType := labels["error_type"]
		if errorType != "" {
			snapshot.OllamaErrorsTotal[errorType] = value
		}

	case strings.HasPrefix(metricName, "ollama_proxy_ollama_connections_active"):
		snapshot.OllamaConnectionsActive = value

	case strings.HasPrefix(metricName, "ollama_proxy_api_key_requests_total"):
		keyID := labels["key_id"]
		if keyID != "" {
			snapshot.APIKeyRequestsTotal[keyID] = value
		}

	case strings.HasPrefix(metricName, "ollama_proxy_api_key_rate_limit_exceeded_total"):
		keyID := labels["key_id"]
		if keyID != "" {
			snapshot.APIKeyRateLimitExceeded[keyID] = value
		}

	case strings.HasPrefix(metricName, "ollama_proxy_api_keys_active_total"):
		snapshot.APIKeysActiveTotal = value

	case strings.HasPrefix(metricName, "ollama_proxy_api_key_tokens_used_total"):
		keyID := labels["key_id"]
		if keyID != "" {
			snapshot.APIKeyTokensUsedTotal[keyID] = value
		}

	case strings.HasPrefix(metricName, "ollama_proxy_circuit_breaker_state"):
		breakerName := labels["breaker_name"]
		if breakerName != "" {
			snapshot.CircuitBreakerState[breakerName] = value
		}

	case strings.HasPrefix(metricName, "ollama_proxy_circuit_breaker_trips_total"):
		breakerName := labels["breaker_name"]
		if breakerName != "" {
			snapshot.CircuitBreakerTripsTotal[breakerName] = value
		}
	}

	return nil
}

// parseMetricNameAndLabels разбирает имя метрики и labels
func parseMetricNameAndLabels(metricPart string) (string, map[string]string) {
	labels := make(map[string]string)

	// Ищем открывающую скобку
	bracketIdx := strings.Index(metricPart, "{")
	if bracketIdx == -1 {
		// Нет labels
		return metricPart, labels
	}

	metricName := metricPart[:bracketIdx]

	// Ищем закрывающую скобку
	closeBracketIdx := strings.Index(metricPart, "}")
	if closeBracketIdx == -1 {
		return metricName, labels
	}

	labelsStr := metricPart[bracketIdx+1 : closeBracketIdx]

	// Парсим labels: label1="value1",label2="value2"
	labelPairs := strings.SplitSeq(labelsStr, ",")
	for pair := range labelPairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
			labels[key] = value
		}
	}

	return metricName, labels
}

// GetTotalHTTPRequests возвращает суммарное количество HTTP запросов
func (m *MetricsSnapshot) GetTotalHTTPRequests() float64 {
	var total float64
	for _, count := range m.HTTPRequestsTotal {
		total += count
	}
	return total
}

// GetTotalOllamaRequests возвращает суммарное количество Ollama запросов
func (m *MetricsSnapshot) GetTotalOllamaRequests() float64 {
	var total float64
	for _, count := range m.OllamaRequestsTotal {
		total += count
	}
	return total
}

// GetTotalOllamaErrors возвращает суммарное количество ошибок Ollama
func (m *MetricsSnapshot) GetTotalOllamaErrors() float64 {
	var total float64
	for _, count := range m.OllamaErrorsTotal {
		total += count
	}
	return total
}

// GetTotalAPIKeyRequests возвращает суммарное количество запросов по API ключам
func (m *MetricsSnapshot) GetTotalAPIKeyRequests() float64 {
	var total float64
	for _, count := range m.APIKeyRequestsTotal {
		total += count
	}
	return total
}

// GetTotalTokensUsed возвращает суммарное количество использованных токенов
func (m *MetricsSnapshot) GetTotalTokensUsed() float64 {
	var total float64
	for _, count := range m.APIKeyTokensUsedTotal {
		total += count
	}
	return total
}

// GetAverageHTTPDuration возвращает среднюю длительность HTTP запросов (в миллисекундах)
func (m *MetricsSnapshot) GetAverageHTTPDuration() float64 {
	if len(m.HTTPRequestDuration) == 0 {
		return 0
	}

	var totalDuration float64
	var totalRequests float64

	for endpoint, duration := range m.HTTPRequestDuration {
		totalDuration += duration
		if requests, ok := m.HTTPRequestsTotal[endpoint]; ok {
			totalRequests += requests
		}
	}

	if totalRequests == 0 {
		return 0
	}

	// Конвертируем в миллисекунды
	return (totalDuration / totalRequests) * 1000
}

// GetAverageOllamaDuration возвращает среднюю длительность Ollama запросов (в миллисекундах)
func (m *MetricsSnapshot) GetAverageOllamaDuration() float64 {
	if len(m.OllamaRequestDuration) == 0 {
		return 0
	}

	var totalDuration float64
	var totalRequests float64

	for key, duration := range m.OllamaRequestDuration {
		totalDuration += duration
		if requests, ok := m.OllamaRequestsTotal[key]; ok {
			totalRequests += requests
		}
	}

	if totalRequests == 0 {
		return 0
	}

	// Конвертируем в миллисекунды
	return (totalDuration / totalRequests) * 1000
}
