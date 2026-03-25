// Package optimizer provides smart prompt optimization for local models
package optimizer

import (
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"aigateway/internal/models"
)

// PromptOptimizer оптимизирует промпты для локальных моделей
type PromptOptimizer struct {
	logger *logrus.Logger
	config OptimizerConfig
}

// OptimizerConfig конфигурация оптимизатора
type OptimizerConfig struct {
	// EnableSystemMessageSimplification упрощать system message для локальных моделей
	EnableSystemMessageSimplification bool

	// EnableSmartToolFiltering фильтровать tools по релевантности
	EnableSmartToolFiltering bool

	// MaxToolsPerRequest максимум tools в одном запросе
	MaxToolsPerRequest int

	// PreserveKeyInstructions ключевые инструкции которые нужно сохранить
	PreserveKeyInstructions []string
}

// NewPromptOptimizer создает новый оптимизатор
func NewPromptOptimizer(logger *logrus.Logger, config OptimizerConfig) *PromptOptimizer {
	return &PromptOptimizer{
		logger: logger,
		config: config,
	}
}

// OptimizeRequest оптимизирует запрос для локальной модели
func (o *PromptOptimizer) OptimizeRequest(req *models.ChatCompletionRequest) *models.ChatCompletionRequest {
	optimized := *req // Копируем

	// 1. Упрощаем system message
	if o.config.EnableSystemMessageSimplification {
		optimized.Messages = o.simplifySystemMessage(optimized.Messages)
	}

	// 2. Сжимаем историю диалога если слишком длинная
	if len(optimized.Messages) > 4 {
		optimized.Messages = o.compressMessageHistory(optimized.Messages)
	}

	// 3. Фильтруем tools по релевантности
	if o.config.EnableSmartToolFiltering && len(optimized.Tools) > 0 {
		optimized.Tools = o.filterRelevantTools(optimized.Tools, optimized.Messages)
	}

	o.logOptimizationStats(req, &optimized)

	return &optimized
}

// compressMessageHistory сжимает длинную историю диалога
// Оставляет только system + последние 2 пары user/assistant сообщений
func (o *PromptOptimizer) compressMessageHistory(messages []models.ChatMessage) []models.ChatMessage {
	if len(messages) <= 4 {
		return messages // Уже короткая история
	}

	compressed := make([]models.ChatMessage, 0, 5)

	// 1. Сохраняем system message
	for _, msg := range messages {
		if msg.Role == "system" {
			compressed = append(compressed, msg)
			break
		}
	}

	// 2. Берём только последние 3 сообщения (обычно user-assistant-user)
	startIdx := max(len(messages)-3, 0)

	for i := startIdx; i < len(messages); i++ {
		msg := messages[i]

		// Удаляем огромные вложенные контексты из user messages
		if msg.Role == "user" {
			if contentStr, ok := msg.Content.(string); ok {
				// Если в сообщении есть <context> или <files> блоки - удаляем их
				if strings.Contains(contentStr, "<context>") || strings.Contains(contentStr, "<files>") {
					// Извлекаем только основной вопрос до <context>
					if idx := strings.Index(contentStr, "<context>"); idx > 0 {
						contentStr = strings.TrimSpace(contentStr[:idx])
					}
					if idx := strings.Index(contentStr, "\\u003ccontext\\u003e"); idx > 0 {
						contentStr = strings.TrimSpace(contentStr[:idx])
					}
					msg.Content = contentStr
					o.logger.WithField("message_index", i).Debug("Removed embedded context from user message")
				}
			}
		}

		compressed = append(compressed, msg)
	}

	o.logger.WithFields(logrus.Fields{
		"original_messages":   len(messages),
		"compressed_messages": len(compressed),
		"removed_messages":    len(messages) - len(compressed),
	}).Info("Message history compressed for local model")

	return compressed
}

// simplifySystemMessage упрощает system message убирая избыточные инструкции
func (o *PromptOptimizer) simplifySystemMessage(messages []models.ChatMessage) []models.ChatMessage {
	result := make([]models.ChatMessage, len(messages))
	copy(result, messages)

	for i, msg := range result {
		if msg.Role != "system" {
			continue
		}

		// Извлекаем Content как строку
		contentStr, ok := msg.Content.(string)
		if !ok {
			continue // Пропускаем если Content не строка
		}

		// Извлекаем только критически важные части
		simplified := o.extractEssentialInstructions(contentStr)

		if simplified != contentStr {
			result[i].Content = simplified
			o.logger.WithFields(logrus.Fields{
				"original_length":   len(contentStr),
				"simplified_length": len(simplified),
				"reduction_percent": (1 - float64(len(simplified))/float64(len(contentStr))) * 100,
			}).Debug("System message simplified")
		}
	}

	return result
}

// extractEssentialInstructions извлекает только важные инструкции
func (o *PromptOptimizer) extractEssentialInstructions(systemMsg string) string {
	// Ключевые секции которые нужно сохранить
	essentialSections := []string{
		"## Tool Use",      // Инструкции по использованию tools
		"## Communication", // Базовые правила коммуникации
	}

	var essential []string

	// Добавляем краткое введение
	essential = append(essential, "You are a helpful AI assistant with access to tools.")

	// Извлекаем только ключевые секции
	lines := strings.Split(systemMsg, "\n")
	var inEssentialSection bool
	var currentSection []string

	for _, line := range lines {
		// Проверяем начало важной секции
		isEssentialHeader := false
		for _, section := range essentialSections {
			if strings.Contains(line, section) {
				inEssentialSection = true
				isEssentialHeader = true

				// Сохраняем предыдущую секцию
				if len(currentSection) > 0 {
					essential = append(essential, strings.Join(currentSection, "\n"))
					currentSection = []string{}
				}
				break
			}
		}

		// Проверяем начало новой секции (##)
		if strings.HasPrefix(strings.TrimSpace(line), "##") && !isEssentialHeader {
			inEssentialSection = false
			if len(currentSection) > 0 {
				essential = append(essential, strings.Join(currentSection, "\n"))
				currentSection = []string{}
			}
		}

		// Собираем строки важной секции
		if inEssentialSection {
			currentSection = append(currentSection, line)
		}
	}

	// Добавляем последнюю секцию
	if len(currentSection) > 0 {
		essential = append(essential, strings.Join(currentSection, "\n"))
	}

	// 🎯 ULTRA-MINIMAL MODE: заменяем всё минимальным промптом
	ultraMinimal := []string{
		"You are a helpful AI assistant.",
		"",
		"When user requests:",
	}

	// Добавляем пользовательские инструкции
	for _, instruction := range o.config.PreserveKeyInstructions {
		ultraMinimal = append(ultraMinimal, "- "+instruction)
	}

	result := strings.Join(ultraMinimal, "\n")

	o.logger.WithFields(logrus.Fields{
		"mode":            "ultra-minimal",
		"original_length": len(systemMsg),
		"result_length":   len(result),
		"reduction":       fmt.Sprintf("%.1f%%", (1-float64(len(result))/float64(len(systemMsg)))*100),
	}).Info("🎯 Ultra-minimal system message for reliable tool calling")

	return result
}

// filterRelevantTools фильтрует tools оставляя только релевантные для запроса
func (o *PromptOptimizer) filterRelevantTools(tools []models.Tool, messages []models.ChatMessage) []models.Tool {
	if len(tools) <= o.config.MaxToolsPerRequest {
		return tools // Уже в пределах лимита
	}

	// Получаем последний user message для анализа
	var userQuery string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			if contentStr, ok := messages[i].Content.(string); ok {
				userQuery = strings.ToLower(contentStr)
				break
			}
		}
	}

	// Скоринг tools по релевантности
	type scoredTool struct {
		tool  models.Tool
		score int
	}

	scored := make([]scoredTool, 0, len(tools))

	for _, tool := range tools {
		score := o.calculateToolRelevance(tool, userQuery)
		scored = append(scored, scoredTool{tool: tool, score: score})
	}

	// Сортируем по score (простой bubble sort для малого количества)
	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	// Берем топ N tools
	result := make([]models.Tool, 0, o.config.MaxToolsPerRequest)
	toolNames := make([]string, 0, o.config.MaxToolsPerRequest)

	for i := 0; i < len(scored) && i < o.config.MaxToolsPerRequest; i++ {
		result = append(result, scored[i].tool)
		toolNames = append(toolNames, scored[i].tool.Function.Name)
	}

	o.logger.WithFields(logrus.Fields{
		"original_count": len(tools),
		"filtered_count": len(result),
		"selected_tools": toolNames,
		"query_keywords": extractKeywords(userQuery),
	}).Info("Tools filtered by relevance")

	return result
}

// calculateToolRelevance вычисляет релевантность tool для запроса
func (o *PromptOptimizer) calculateToolRelevance(tool models.Tool, userQuery string) int {
	score := 0

	toolName := strings.ToLower(tool.Function.Name)
	toolDesc := strings.ToLower(tool.Function.Description)

	// Ключевые слова для разных типов операций
	keywords := map[string][]string{
		"file_ops":   {"file", "read", "write", "create", "edit", "delete", "move", "copy"},
		"search":     {"find", "search", "grep", "list", "show", "locate"},
		"navigation": {"directory", "folder", "path", "tree", "structure"},
		"execution":  {"run", "execute", "command", "terminal", "shell"},
		"analysis":   {"analyze", "check", "diagnose", "inspect", "validate"},
	}

	// Проверяем совпадение ключевых слов
	for category, words := range keywords {
		for _, word := range words {
			if strings.Contains(userQuery, word) {
				// Если tool name или description содержит это слово
				if strings.Contains(toolName, word) {
					score += 10 // Высокий приоритет - совпадение в имени
				}
				if strings.Contains(toolDesc, word) {
					score += 5 // Средний приоритет - совпадение в описании
				}
				if strings.Contains(toolName, category) {
					score += 3 // Низкий приоритет - категория
				}
			}
		}
	}

	// Специальные правила для популярных tools
	if strings.Contains(userQuery, "file") && strings.Contains(toolName, "find_path") {
		score += 15 // find_path критичен для поиска файлов
	}
	if strings.Contains(userQuery, "read") && strings.Contains(toolName, "read_file") {
		score += 15
	}
	if strings.Contains(userQuery, "list") && strings.Contains(toolName, "list_directory") {
		score += 15
	}

	return score
}

// logOptimizationStats логирует статистику оптимизации
func (o *PromptOptimizer) logOptimizationStats(original, optimized *models.ChatCompletionRequest) {
	originalSystemLen := 0
	optimizedSystemLen := 0

	for _, msg := range original.Messages {
		if msg.Role == "system" {
			if contentStr, ok := msg.Content.(string); ok {
				originalSystemLen += len(contentStr)
			}
		}
	}

	for _, msg := range optimized.Messages {
		if msg.Role == "system" {
			if contentStr, ok := msg.Content.(string); ok {
				optimizedSystemLen += len(contentStr)
			}
		}
	}

	o.logger.WithFields(logrus.Fields{
		"original_system_length":   originalSystemLen,
		"optimized_system_length":  optimizedSystemLen,
		"system_reduction_percent": float64(originalSystemLen-optimizedSystemLen) / float64(originalSystemLen) * 100,
		"original_tools_count":     len(original.Tools),
		"optimized_tools_count":    len(optimized.Tools),
	}).Info("Request optimized for local model")
}

// extractKeywords извлекает ключевые слова из запроса
func extractKeywords(query string) []string {
	words := strings.Fields(query)
	keywords := make([]string, 0)

	// Простой фильтр стоп-слов
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"but": true, "in": true, "on": true, "at": true, "to": true,
		"for": true, "of": true, "with": true, "by": true, "from": true,
		"is": true, "are": true, "was": true, "were": true, "be": true,
		"me": true, "i": true, "you": true, "show": true, "help": true,
	}

	for _, word := range words {
		word = strings.ToLower(strings.Trim(word, ".,!?;:"))
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
		}
	}

	return keywords
}
