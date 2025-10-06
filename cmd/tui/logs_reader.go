package main

import (
	"bufio"
	"os"
	"strings"
)

// LogEntry представляет одну запись лога
type LogEntry struct {
	Raw   string
	Level string // DEBUG, INFO, WARN, ERROR
}

// readLastLogs читает последние N строк из файла логов
func readLastLogs(filePath string, maxLines int) ([]LogEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Читаем все строки
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Берем последние maxLines строк
	startIdx := 0
	if len(lines) > maxLines {
		startIdx = len(lines) - maxLines
	}

	// Конвертируем в LogEntry
	entries := make([]LogEntry, 0, maxLines)
	for i := startIdx; i < len(lines); i++ {
		entry := LogEntry{
			Raw:   lines[i],
			Level: detectLogLevel(lines[i]),
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// detectLogLevel определяет уровень логирования из строки
func detectLogLevel(line string) string {
	lineLower := strings.ToLower(line)

	// Ищем ключевые слова
	if strings.Contains(lineLower, "level=error") || strings.Contains(lineLower, "\"level\":\"error\"") {
		return "ERROR"
	}
	if strings.Contains(lineLower, "level=warning") || strings.Contains(lineLower, "level=warn") ||
		strings.Contains(lineLower, "\"level\":\"warning\"") || strings.Contains(lineLower, "\"level\":\"warn\"") {
		return "WARN"
	}
	if strings.Contains(lineLower, "level=debug") || strings.Contains(lineLower, "\"level\":\"debug\"") {
		return "DEBUG"
	}
	if strings.Contains(lineLower, "level=info") || strings.Contains(lineLower, "\"level\":\"info\"") {
		return "INFO"
	}

	// По ключевым словам в тексте
	if strings.Contains(lineLower, "error") || strings.Contains(lineLower, "fatal") || strings.Contains(lineLower, "panic") {
		return "ERROR"
	}
	if strings.Contains(lineLower, "warn") || strings.Contains(lineLower, "warning") {
		return "WARN"
	}
	if strings.Contains(lineLower, "debug") {
		return "DEBUG"
	}

	// По умолчанию INFO
	return "INFO"
}

// colorizeLogLine раскрашивает строку лога по уровню
func colorizeLogLine(entry LogEntry) string {
	switch entry.Level {
	case "ERROR":
		return errorStyle.Render(entry.Raw)
	case "WARN":
		return warningStyle.Render(entry.Raw)
	case "DEBUG":
		return helpStyle.Render(entry.Raw)
	case "INFO":
		return entry.Raw
	default:
		return entry.Raw
	}
}
