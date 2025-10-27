package main

import (
	"strings"
	"time"
)

// parseDuration парсит строку длительности вида "1h23m45s" в time.Duration
func parseDuration(s string) time.Duration {
	// Попробуем стандартный парсинг
	d, err := time.ParseDuration(s)
	if err == nil {
		return d
	}

	// Если не получилось, пробуем парсить вручную
	// Поддерживаем форматы: "1h", "30m", "45s", "1h30m", "2h15m30s"
	s = strings.TrimSpace(s)
	if s == "" || s == "0" || s == "0s" {
		return 0
	}

	// Если уже в правильном формате
	if strings.Contains(s, "h") || strings.Contains(s, "m") || strings.Contains(s, "s") {
		d, err := time.ParseDuration(s)
		if err == nil {
			return d
		}
	}

	// Возвращаем 0 если не смогли распарсить
	return 0
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

