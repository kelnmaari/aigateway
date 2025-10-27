// Package main provides the TUI (Terminal User Interface) entry point for Ollama-OpenAI Proxy
package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"aigateway/internal/config"
)

var (
	// Version информация о версии (устанавливается при сборке)
	Version   = "dev"
	BuildTime = "unknown"
)

// APIKeyInfo информация об API ключе для TUI
type APIKeyInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"` // AUTH-04
	Status      string   `json:"status"`
	Permissions []string `json:"permissions"`
	Models      []string `json:"models"`
	CreatedAt   string   `json:"created_at"`
	LastUsed    string   `json:"last_used"`
	Usage       struct {
		TotalRequests int64 `json:"total_requests"`
		Success       int64 `json:"success"`
		Failed        int64 `json:"failed"`
	} `json:"usage"`
}

// ServerStats структура для хранения статистики с сервера
type ServerStats struct {
	Server struct {
		Status string `json:"status"`
		Host   string `json:"host"`
		Port   int    `json:"port"`
		URL    string `json:"url"`
	} `json:"server"`
	Ollama struct {
		Connected   bool     `json:"connected"`
		URL         string   `json:"url"`
		ModelsCount int      `json:"models_count"`
		Models      []string `json:"models"`
	} `json:"ollama"`
	Stats struct {
		UptimeSeconds   float64 `json:"uptime_seconds"`
		Uptime          string  `json:"uptime"`
		TotalRequests   int64   `json:"total_requests"`
		ActiveRequests  int64   `json:"active_requests"`
		SuccessRequests int64   `json:"success_requests"`
		ErrorRequests   int64   `json:"error_requests"`
		AverageDuration string  `json:"average_duration"`
	} `json:"stats"`
	APIKeys struct {
		Enabled bool         `json:"enabled"`
		Count   int          `json:"count"`
		Keys    []APIKeyInfo `json:"keys"`
	} `json:"api_keys"`
}

// CreateKeyForm данные формы создания API ключа
type CreateKeyForm struct {
	Name         string
	Description  string
	Models       string // Comma-separated
	CurrentField int    // 0=name, 1=description, 2=models
}

// EditKeyForm данные формы редактирования API ключа (AUTH-04)
type EditKeyForm struct {
	KeyID        string
	Name         string
	Description  string
	Models       string // Comma-separated
	Permissions  string // Comma-separated
	CurrentField int    // 0=name, 1=description, 2=models, 3=permissions
}

// RequestInfo информация о запросе для мониторинга (TUI-04)
type RequestInfo struct {
	ID           string   `json:"id"`
	Timestamp    int64    `json:"timestamp"`
	Method       string   `json:"method"`
	Endpoint     string   `json:"endpoint"`
	Status       string   `json:"status"`
	DurationMS   int64    `json:"duration_ms"`
	Model        string   `json:"model,omitempty"`
	APIKeyID     string   `json:"api_key_id,omitempty"`
	APIKeyName   string   `json:"api_key_name,omitempty"`
	RemoteAddr   string   `json:"remote_addr,omitempty"`
	UserAgent    string   `json:"user_agent,omitempty"`
	StatusCode   int      `json:"status_code,omitempty"`
	ErrorMessage string   `json:"error,omitempty"`
	Messages     int      `json:"messages,omitempty"`
	Tools        int      `json:"tools,omitempty"`
	Stream       bool     `json:"stream,omitempty"`
	Temperature  *float64 `json:"temperature,omitempty"`
	TotalTokens  int      `json:"total_tokens,omitempty"`
}

// RequestsResponse ответ от API /api/requests
type RequestsResponse struct {
	Requests   []RequestInfo `json:"requests"`
	Pagination struct {
		Page    int `json:"page"`
		PerPage int `json:"per_page"`
		Total   int `json:"total"`
		Pages   int `json:"pages"`
	} `json:"pagination"`
}

// RequestsState состояние для Request Monitor screen
type RequestsState struct {
	requests []RequestInfo
	total    int
	pages    int

	// Навигация и фильтрация
	page         int
	perPage      int
	sortField    string // time, duration, status, endpoint
	sortOrder    string // asc, desc
	statusFilter string // all, success, error, pending

	// Детальный просмотр
	selectedIndex int
	showDetail    bool
	detailRequest *RequestInfo
}

// ServerConfig структура для хранения конфигурации с сервера
type ServerConfig struct {
	Server struct {
		Host           string `json:"host"`
		Port           int    `json:"port"`
		ReadTimeout    string `json:"read_timeout"`
		WriteTimeout   string `json:"write_timeout"`
		IdleTimeout    string `json:"idle_timeout"`
		MaxHeaderBytes int    `json:"max_header_bytes"`
	} `json:"server"`
	Ollama struct {
		URL                string `json:"url"`
		Timeout            string `json:"timeout"`
		RetryAttempts      int    `json:"retry_attempts"`
		RetryDelay         string `json:"retry_delay"`
		ConnectionPoolSize int    `json:"connection_pool_size"`
		KeepAlive          bool   `json:"keep_alive"`
	} `json:"ollama"`
	Auth struct {
		Enabled      bool   `json:"enabled"`
		StorageType  string `json:"storage_type"`
		StoragePath  string `json:"storage_path"`
		RateLimiting struct {
			Enabled                  bool `json:"enabled"`
			DefaultRequestsPerMinute int  `json:"default_requests_per_minute"`
			DefaultRequestsPerHour   int  `json:"default_requests_per_hour"`
		} `json:"rate_limiting"`
	} `json:"auth"`
	Logging struct {
		Level      string `json:"level"`
		Format     string `json:"format"`
		Output     string `json:"output"`
		FilePath   string `json:"file_path,omitempty"`
		MaxSize    int    `json:"max_size,omitempty"`
		MaxBackups int    `json:"max_backups,omitempty"`
		MaxAge     int    `json:"max_age,omitempty"`
		Compress   bool   `json:"compress,omitempty"`
	} `json:"logging"`
	Models struct {
		Mapping map[string]string `json:"mapping"`
		Aliases map[string]string `json:"aliases"`
		Hidden  []string          `json:"hidden"`
		Cache   struct {
			Enabled         bool   `json:"enabled"`
			TTL             string `json:"ttl"`
			RefreshInterval string `json:"refresh_interval"`
		} `json:"cache"`
	} `json:"models"`
	Tools struct {
		ForceUsage    bool   `json:"force_usage"`
		DefaultChoice string `json:"default_choice"`
		FallbackModel string `json:"fallback_model,omitempty"`
		Optimizer     struct {
			Enabled               bool     `json:"enabled"`
			SimplifySystemMessage bool     `json:"simplify_system_message"`
			SmartToolFiltering    bool     `json:"smart_tool_filtering"`
			MaxToolsPerRequest    int      `json:"max_tools_per_request"`
			PreserveInstructions  []string `json:"preserve_instructions"`
		} `json:"optimizer"`
	} `json:"tools"`
	Metrics struct {
		Enabled        bool   `json:"enabled"`
		PrometheusPath string `json:"prometheus_path"`
	} `json:"metrics"`
	TUI struct {
		Enabled     bool   `json:"enabled"`
		RefreshRate string `json:"refresh_rate"`
		Theme       string `json:"theme"`
	} `json:"tui"`
	Development struct {
		HotReload      bool `json:"hot_reload"`
		DebugMode      bool `json:"debug_mode"`
		ProfileEnabled bool `json:"profile_enabled"`
		PProfEnabled   bool `json:"pprof_enabled"`
		RaceDetection  bool `json:"race_detection"`
	} `json:"development"`
}

// Model представляет основную модель TUI приложения
type Model struct {
	currentView  string
	width        int
	height       int
	ready        bool
	stats        ServerStats
	metrics      *MetricsSnapshot // Prometheus метрики
	config       ServerConfig     // Конфигурация сервера
	serverURL    string
	adminToken   string // Admin API token from config
	lastUpdate   time.Time
	errorMessage string

	// Навигация
	activeTab int  // Активная вкладка (0-5)
	showHelp  bool // Показать экран помощи

	// Прокрутка контента
	scrollOffset int // Смещение для прокрутки длинного контента

	// Создание API ключа
	creatingKey    bool
	createForm     CreateKeyForm
	createdKey     string // Показываем созданный ключ
	successMessage string

	// Enhanced Key Management (AUTH-04)
	selectedKeyIndex int    // Индекс выбранного ключа
	selectedKeyID    string // ID выбранного ключа
	editingKey       bool   // Редактируем ключ
	editForm         EditKeyForm
	confirmAction    string // "revoke" или "enable"
	showConfirm      bool   // Показать модальное окно подтверждения

	// Request Monitor (TUI-04)
	requestsState RequestsState
}

// Доступные экраны TUI
const (
	dashboardView = "dashboard"
	requestsView  = "requests"
	modelsView    = "models"
	apiKeysView   = "apikeys"
	configView    = "config"
	logsView      = "logs"
	controlView   = "control"
)

// Стили для TUI
var (
	titleStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#7D56F4")).
			Foreground(lipgloss.Color("#FAFAFA")).
			Padding(0, 1).
			Bold(true)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666")).
			Italic(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true)

	warningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500")).
			Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7D56F4")).
			Padding(1, 2)
)

func main() {
	fmt.Printf("🖥️  Ollama-OpenAI Proxy TUI v%s (built %s)\n", Version, BuildTime)

	// Parse command line flags
	configPath := flag.String("config", "configs/dev.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	// Validate admin key
	if cfg.Auth.AdminKey == "" {
		log.Fatalf("❌ Admin key not configured in auth.admin_key")
	}

	// Build server URL
	serverURL := fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port)

	// Создание TUI приложения
	m := Model{
		currentView: dashboardView,
		serverURL:   serverURL,
		adminToken:  cfg.Auth.AdminKey,    // Load from config
		metrics:     NewMetricsSnapshot(), // Инициализируем метрики
		requestsState: RequestsState{
			page:          1,
			perPage:       20, // 20 запросов на странице для TUI
			sortField:     "time",
			sortOrder:     "desc",
			statusFilter:  "all",
			selectedIndex: 0,
		},
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		log.Fatalf("❌ Ошибка запуска TUI: %v", err)
	}
}

// loadConfig loads configuration from file
func loadConfig(configPath string) (*config.Config, error) {
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Try absolute path
		absPath, _ := filepath.Abs(configPath)
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("config file not found: %s (also tried: %s)", configPath, absPath)
		}
		configPath = absPath
	}

	// Load config
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from %s: %w", configPath, err)
	}

	return cfg, nil
}

// Init инициализирует модель
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		fetchStatsCmd(m.serverURL),   // Сразу загружаем данные
		fetchMetricsCmd(m.serverURL), // Загружаем метрики
		fetchConfigCmd(m.serverURL),  // Загружаем конфигурацию
		tickCmd(),                    // Запускаем периодическое обновление
	)
}

// Update обрабатывает сообщения
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case statsMsg:
		m.stats = ServerStats(msg)
		m.lastUpdate = time.Now()
		// НЕ очищаем ошибку автоматически - пусть пользователь видит
		// m.errorMessage = ""
		return m, nil

	case metricsMsg:
		if msg != nil {
			m.metrics = msg
		}
		return m, nil

	case configMsg:
		m.config = ServerConfig(msg)
		return m, nil

	case requestsMsg:
		m.requestsState.requests = msg.Requests
		m.requestsState.total = msg.Pagination.Total
		m.requestsState.pages = msg.Pagination.Pages
		return m, nil

	case keyCreatedMsg:
		m.createdKey = msg.plainKey
		m.successMessage = fmt.Sprintf("✅ API ключ '%s' создан успешно!", msg.keyName)
		m.errorMessage = ""
		// НЕ обновляем список сразу - пусть пользователь сначала скопирует ключ
		return m, nil

	case errMsg:
		m.errorMessage = msg.Error()
		return m, nil

	case tickMsg:
		// Если показан созданный ключ, НЕ обновляем автоматически
		// чтобы не затереть его показ
		if m.createdKey != "" {
			return m, tickCmd() // Только перезапускаем таймер
		}

		// Обычное автообновление
		return m, tea.Batch(
			fetchStatsCmd(m.serverURL),
			fetchMetricsCmd(m.serverURL),
			tickCmd(),
		)

	case tea.MouseMsg:
		// Mouse support для навигации
		switch msg.Type {
		case tea.MouseWheelUp:
			if m.scrollOffset > 0 {
				m.scrollOffset -= 3 // Быстрее чем клавиатура
			}
			return m, nil
		case tea.MouseWheelDown:
			m.scrollOffset += 3
			return m, nil
		case tea.MouseLeft:
			// Click events для переключения вкладок
			// Вычисляем позицию клика
			clickY := msg.Y
			clickX := msg.X

			// Tab bar находится на линии 2 (после заголовка)
			if clickY == 2 {
				// Примерное расположение табов
				tabs := []string{"Dashboard", "Requests", "Models", "Keys", "Config", "Logs", "Control"}
				tabWidth := 15 // Примерная ширина таба

				for i, tab := range tabs {
					startX := i * tabWidth
					endX := startX + len(tab) + 4 // +4 для отступов

					if clickX >= startX && clickX <= endX {
						m.activeTab = i
						m.scrollOffset = 0 // Сбрасываем прокрутку при смене таба

						// Переключаем view в зависимости от таба
						switch i {
						case 0:
							m.currentView = dashboardView
						case 1:
							m.currentView = requestsView
							return m, fetchRequestsCmd(m.serverURL, m.requestsState)
						case 2:
							m.currentView = modelsView
						case 3:
							m.currentView = apiKeysView
							m.createdKey = ""
							m.successMessage = ""
						case 4:
							m.currentView = configView
						case 5:
							m.currentView = logsView
						case 6:
							m.currentView = controlView
						}

						return m, nil
					}
				}
			}
		}

	case tea.KeyMsg:
		// Если показан созданный ключ, обрабатываем специальные клавиши
		if m.createdKey != "" {
			switch msg.String() {
			case "c", "C":
				// Копируем ключ в буфер обмена
				if err := clipboard.WriteAll(m.createdKey); err != nil {
					m.errorMessage = fmt.Sprintf("Ошибка копирования: %v", err)
				} else {
					m.successMessage = "✅ Ключ скопирован в буфер обмена!"
				}
				return m, nil
			default:
				// Любая другая клавиша закрывает экран
				m.createdKey = ""
				// Теперь обновляем список ключей
				return m, fetchStatsCmd(m.serverURL)
			}
		}

		// Если создаем ключ, обрабатываем ввод формы
		if m.creatingKey {
			return m.handleCreateKeyInput(msg)
		}

		// Обработка клавиш для Request Monitor (TUI-04)
		if m.currentView == requestsView {
			switch msg.String() {
			case "up", "k":
				// Навигация вверх по списку запросов
				if m.requestsState.selectedIndex > 0 {
					m.requestsState.selectedIndex--
				}
				return m, nil
			case "down", "j":
				// Навигация вниз по списку запросов
				if m.requestsState.selectedIndex < len(m.requestsState.requests)-1 {
					m.requestsState.selectedIndex++
				}
				return m, nil
			case "enter":
				// Показать детали выбранного запроса
				if m.requestsState.selectedIndex < len(m.requestsState.requests) {
					m.requestsState.showDetail = true
					m.requestsState.detailRequest = &m.requestsState.requests[m.requestsState.selectedIndex]
				}
				return m, nil
			case "esc":
				// Закрыть детальный просмотр
				if m.requestsState.showDetail {
					m.requestsState.showDetail = false
					m.requestsState.detailRequest = nil
					return m, nil
				}
			case "n":
				// Следующая страница
				if m.requestsState.page < m.requestsState.pages {
					m.requestsState.page++
					return m, fetchRequestsCmd(m.serverURL, m.requestsState)
				}
				return m, nil
			case "p":
				// Предыдущая страница
				if m.requestsState.page > 1 {
					m.requestsState.page--
					return m, fetchRequestsCmd(m.serverURL, m.requestsState)
				}
				return m, nil
			case "s":
				// Переключение сортировки
				return m, m.cycleSortField()
			case "f":
				// Переключение фильтра
				return m, m.cycleStatusFilter()
			case "r":
				// Обновить список запросов
				return m, fetchRequestsCmd(m.serverURL, m.requestsState)
			case "e":
				// Экспорт запросов
				if len(m.requestsState.requests) > 0 {
					err := m.exportRequests()
					if err != nil {
						m.errorMessage = fmt.Sprintf("Ошибка экспорта: %v", err)
					} else {
						m.successMessage = "✅ Запросы успешно экспортированы!"
					}
				}
				return m, nil
			}
		}

		// Обработка клавиш для API Keys Management (AUTH-04)
		if m.currentView == apiKeysView {
			// Если показываем модальное окно подтверждения
			if m.showConfirm {
				switch msg.String() {
				case "y", "Y":
					// Подтверждаем действие
					return m, m.executeConfirmedAction()
				case "n", "N", "esc":
					// Отменяем действие
					m.showConfirm = false
					m.confirmAction = ""
					return m, nil
				}
				return m, nil
			}

			// Если редактируем ключ
			if m.editingKey {
				return m.handleEditKeyInput(msg)
			}

			// Навигация по списку ключей
			switch msg.String() {
			case "up", "k":
				if m.selectedKeyIndex > 0 {
					m.selectedKeyIndex--
					m.updateSelectedKeyID()
				}
				return m, nil
			case "down", "j":
				if m.selectedKeyIndex < len(m.stats.APIKeys.Keys)-1 {
					m.selectedKeyIndex++
					m.updateSelectedKeyID()
				}
				return m, nil
			case "e":
				// Редактировать выбранный API ключ
				if m.selectedKeyIndex < len(m.stats.APIKeys.Keys) {
					return m, m.startEditingKey()
				}
				return m, nil
			case "x":
				// Toggle Revoke/Enable выбранного ключа
				if m.selectedKeyIndex < len(m.stats.APIKeys.Keys) {
					key := m.stats.APIKeys.Keys[m.selectedKeyIndex]
					if key.Status == "active" {
						m.confirmAction = "revoke"
						m.showConfirm = true
					} else if key.Status == "revoked" {
						m.confirmAction = "enable"
						m.showConfirm = true
					} else {
						m.errorMessage = fmt.Sprintf("⚠️  Невозможно изменить статус ключа '%s'", key.Status)
					}
				}
				return m, nil
			case "d":
				// Удалить API ключ (пока не реализовано)
				m.errorMessage = "⏳ Удаление API ключей будет добавлено позже..."
				return m, nil
			}
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit

		case "esc":
			// Очищаем ошибку вручную
			if m.errorMessage != "" {
				m.errorMessage = ""
				return m, nil
			}

		// Прокрутка контента
		case "up", "k":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
			return m, nil
		case "down", "j":
			m.scrollOffset++
			return m, nil
		case "pgup":
			m.scrollOffset -= 10
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
			return m, nil
		case "pgdown":
			m.scrollOffset += 10
			return m, nil
		case "home":
			m.scrollOffset = 0
			return m, nil

		// Навигация между экранами
		case "1":
			m.currentView = dashboardView
			m.scrollOffset = 0 // Сброс прокрутки
		case "2":
			m.currentView = requestsView
			m.scrollOffset = 0
			// Загружаем список запросов при переключении на экран
			return m, fetchRequestsCmd(m.serverURL, m.requestsState)
		case "3":
			m.currentView = modelsView
			m.scrollOffset = 0
		case "4":
			m.currentView = apiKeysView
			m.createdKey = "" // Очищаем показанный ключ
			m.successMessage = ""
			m.scrollOffset = 0
		case "5":
			m.currentView = configView
			m.scrollOffset = 0
		case "6":
			m.currentView = logsView
			m.scrollOffset = 0
		case "7":
			m.currentView = controlView
			m.scrollOffset = 0
		case "r":
			// Ручное обновление статистики
			return m, fetchStatsCmd(m.serverURL)

		case "h", "?":
			// Toggle help screen
			m.showHelp = !m.showHelp
			return m, nil
		case "n":
			// Создание нового ключа (только на экране API Keys)
			if m.currentView == apiKeysView && m.stats.APIKeys.Enabled {
				m.creatingKey = true
				m.createForm = CreateKeyForm{CurrentField: 0}
			}
		}

		return m, nil
	}

	return m, nil
}

// View отрисовывает TUI
func (m Model) View() string {
	if !m.ready {
		return "Инициализация TUI..."
	}

	// Show help screen if active
	if m.showHelp {
		return m.renderHelpScreen()
	}

	// Header
	updateInfo := ""
	if !m.lastUpdate.IsZero() {
		updateInfo = fmt.Sprintf(" | Обновлено: %s", m.lastUpdate.Format("15:04:05"))
	}
	header := titleStyle.Render("🦙 Ollama-OpenAI Proxy TUI") + "\n" +
		subtitleStyle.Render(fmt.Sprintf("v%s%s", Version, updateInfo)) + "\n\n"

	// Error banner
	errorBanner := ""
	if m.errorMessage != "" {
		errorBanner = errorStyle.Render(fmt.Sprintf("❌ ОШИБКА: %s", m.errorMessage)) + "\n"
		errorBanner += helpStyle.Render("(Нажмите Esc чтобы закрыть)") + "\n\n"
	}

	// Navigation menu
	nav := m.renderNavigation()

	// Main content
	content := m.renderCurrentView()

	// Применяем прокрутку к контенту (если он длинный)
	// Вычисляем доступную высоту для контента
	headerHeight := strings.Count(header, "\n") + strings.Count(errorBanner, "\n") + strings.Count(nav, "\n") + 3
	footerHeight := 2
	availableHeight := m.height - headerHeight - footerHeight

	if availableHeight < 5 {
		availableHeight = 5 // Минимум 5 строк
	}

	// Применяем прокрутку
	content = applyScroll(content, m.scrollOffset, availableHeight)

	// Footer
	footer := "\n" + helpStyle.Render("Прокрутка: ↑↓/j/k/PgUp/PgDn | Навигация: 1-7 | Помощь: h/? | Обновить: r | Выход: q")

	// Собираем всё вместе
	view := header + errorBanner + nav + "\n\n" + content + footer

	return view
}

// renderNavigation отрисовывает меню навигации
func (m Model) renderNavigation() string {
	items := []struct {
		key   string
		label string
		view  string
	}{
		{"1", "📊 Dashboard", dashboardView},
		{"2", "🔍 Requests", requestsView},
		{"3", "🤖 Models", modelsView},
		{"4", "🔑 API Keys", apiKeysView},
		{"5", "⚙️  Config", configView},
		{"6", "📋 Logs", logsView},
		{"7", "🎛️  Control", controlView},
	}

	var navItems []string
	for _, item := range items {
		style := helpStyle
		if item.view == m.currentView {
			style = statusStyle
		}
		navItems = append(navItems, style.Render(fmt.Sprintf("[%s] %s", item.key, item.label)))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, navItems...)
}

// renderCurrentView отрисовывает текущий активный экран
func (m Model) renderCurrentView() string {
	switch m.currentView {
	case dashboardView:
		return m.renderDashboard()
	case requestsView:
		return m.renderRequests()
	case modelsView:
		return m.renderModels()
	case apiKeysView:
		return m.renderAPIKeys()
	case configView:
		return m.renderConfig()
	case logsView:
		return m.renderLogs()
	case controlView:
		return m.renderControl()
	default:
		return errorStyle.Render("Неизвестный экран: " + m.currentView)
	}
}

// renderDashboard отрисовывает dashboard экран
func (m Model) renderDashboard() string {
	content := "📊 DASHBOARD\n\n"

	// Статус сервера (из реальных данных)
	if m.stats.Server.Status == "running" {
		content += statusStyle.Render("✅ Сервер: Работает") + "\n"
		content += helpStyle.Render(fmt.Sprintf("🔗 URL: %s", m.stats.Server.URL)) + "\n"
		content += helpStyle.Render(fmt.Sprintf("⚡ Uptime: %s", m.stats.Stats.Uptime)) + "\n\n"
	} else {
		content += errorStyle.Render("❌ Сервер: Недоступен") + "\n"
		content += helpStyle.Render(fmt.Sprintf("🔗 URL: %s", m.serverURL)) + "\n\n"
	}

	// Статистика (из реальных данных)
	content += "📈 СТАТИСТИКА\n"
	content += fmt.Sprintf("• Всего запросов: %d\n", m.stats.Stats.TotalRequests)
	content += fmt.Sprintf("• Активных запросов: %d\n", m.stats.Stats.ActiveRequests)
	content += fmt.Sprintf("• Успешных: %d\n", m.stats.Stats.SuccessRequests)
	content += fmt.Sprintf("• Ошибок: %d\n", m.stats.Stats.ErrorRequests)
	content += fmt.Sprintf("• Среднее время ответа: %s\n", m.stats.Stats.AverageDuration)
	content += "\n"

	// Ollama статус (из реальных данных)
	content += "🦙 OLLAMA STATUS\n"
	if m.stats.Ollama.Connected {
		content += statusStyle.Render("✅ Подключен") + "\n"
		content += helpStyle.Render(fmt.Sprintf("URL: %s", m.stats.Ollama.URL)) + "\n"
		content += helpStyle.Render(fmt.Sprintf("Доступно моделей: %d", m.stats.Ollama.ModelsCount)) + "\n"
	} else {
		content += errorStyle.Render("❌ Не подключен") + "\n"
		content += helpStyle.Render(fmt.Sprintf("URL: %s", m.stats.Ollama.URL)) + "\n"
	}
	content += "\n"

	// Prometheus метрики (если доступны)
	if m.metrics != nil {
		content += "📊 PROMETHEUS METRICS\n"
		content += fmt.Sprintf("• HTTP Requests: %.0f (%.0f in-flight)\n",
			m.metrics.GetTotalHTTPRequests(), m.metrics.HTTPRequestsInFlight)
		content += fmt.Sprintf("• Ollama Requests: %.0f (%.0f errors)\n",
			m.metrics.GetTotalOllamaRequests(), m.metrics.GetTotalOllamaErrors())
		content += fmt.Sprintf("• Avg HTTP Latency: %.2f ms\n", m.metrics.GetAverageHTTPDuration())
		content += fmt.Sprintf("• Avg Ollama Latency: %.2f ms\n", m.metrics.GetAverageOllamaDuration())
		if m.metrics.APIKeysActiveTotal > 0 {
			content += fmt.Sprintf("• Total Tokens Used: %.0f\n", m.metrics.GetTotalTokensUsed())
		}
		content += "\n"
	}

	return borderStyle.Render(content)
}

// renderRequests отрисовывает экран мониторинга запросов
func (m Model) renderRequests() string {
	// Если показываем детали запроса
	if m.requestsState.showDetail && m.requestsState.detailRequest != nil {
		return m.renderRequestDetail()
	}

	var content strings.Builder

	// Заголовок
	content.WriteString(titleStyle.Render("🔍 МОНИТОРИНГ ЗАПРОСОВ") + "\n\n")

	// Если нет запросов
	if len(m.requestsState.requests) == 0 {
		content.WriteString(helpStyle.Render("📭 Нет запросов для отображения\n\n"))
		content.WriteString(subtitleStyle.Render("Выполните несколько запросов к API для просмотра статистики"))
		return borderStyle.Render(content.String())
	}

	// Статистика
	content.WriteString(fmt.Sprintf("📊 Всего: %d | 📄 Страница: %d/%d | 🔍 Фильтр: %s | 📶 Сортировка: %s (%s)\n\n",
		m.requestsState.total,
		m.requestsState.page,
		m.requestsState.pages,
		m.requestsState.statusFilter,
		m.requestsState.sortField,
		m.requestsState.sortOrder,
	))

	// Таблица запросов
	content.WriteString(m.renderRequestsTable())

	// Подсказки
	content.WriteString("\n\n")
	content.WriteString(helpStyle.Render("↑↓: навигация | Enter: детали | n/p: страница | s: сортировка | f: фильтр | e: экспорт | r: обновить"))

	return borderStyle.Render(content.String())
}

// renderRequestsTable отрисовывает таблицу запросов
func (m Model) renderRequestsTable() string {
	var table strings.Builder

	// Заголовок таблицы
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1)

	// Определяем ширины колонок (адаптивно под ширину терминала)
	widths := m.calculateColumnWidths()

	// Рисуем заголовок
	table.WriteString(headerStyle.Render(
		padRight("ID", widths[0]) + " " +
			padRight("Время", widths[1]) + " " +
			padRight("Метод", widths[2]) + " " +
			padRight("Endpoint", widths[3]) + " " +
			padRight("Модель", widths[4]) + " " +
			padRight("Статус", widths[5]) + " " +
			padRight("Длит.", widths[6]),
	))
	table.WriteString("\n")

	// Рисуем строки
	for i, req := range m.requestsState.requests {
		rowStyle := lipgloss.NewStyle().Padding(0, 1)

		// Подсветка выбранной строки
		if i == m.requestsState.selectedIndex {
			rowStyle = rowStyle.Background(lipgloss.Color("#3A3A3A"))
		}

		// Цвет статуса
		statusStr := m.formatRequestStatus(req.Status)

		// Форматируем время
		timestamp := time.Unix(req.Timestamp, 0).Format("15:04:05")

		// Форматируем endpoint (сокращаем если длинный)
		endpoint := req.Endpoint
		if len(endpoint) > widths[3] {
			endpoint = endpoint[:widths[3]-3] + "..."
		}

		// Форматируем модель
		model := req.Model
		if model == "" {
			model = "-"
		}
		if len(model) > widths[4] {
			model = model[:widths[4]-3] + "..."
		}

		// Форматируем длительность
		duration := formatDuration(req.DurationMS)

		row := padRight(req.ID, widths[0]) + " " +
			padRight(timestamp, widths[1]) + " " +
			padRight(req.Method, widths[2]) + " " +
			padRight(endpoint, widths[3]) + " " +
			padRight(model, widths[4]) + " " +
			padRight(statusStr, widths[5]) + " " +
			padRight(duration, widths[6])

		table.WriteString(rowStyle.Render(row) + "\n")
	}

	return table.String()
}

// calculateColumnWidths рассчитывает ширины колонок на основе ширины терминала
func (m Model) calculateColumnWidths() []int {
	// Минимальные ширины
	if m.width < 100 {
		// Узкий терминал - минимальные колонки
		return []int{10, 8, 6, 20, 10, 9, 7} // ID, Time, Method, Endpoint, Model, Status, Duration
	} else if m.width < 140 {
		// Средний терминал
		return []int{10, 8, 6, 30, 15, 9, 7}
	} else {
		// Широкий терминал - полные колонки
		return []int{10, 8, 6, 40, 20, 9, 7}
	}
}

// formatRequestStatus форматирует статус с цветом
func (m Model) formatRequestStatus(status string) string {
	switch status {
	case "success":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Render("✓ Success")
	case "error":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B")).Render("✗ Error")
	case "pending":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("⏳ Pending")
	default:
		return status
	}
}

// formatDuration форматирует длительность запроса
func formatDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	return fmt.Sprintf("%.1fs", float64(ms)/1000.0)
}

// padRight дополняет строку пробелами справа до заданной длины
func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}

// renderRequestDetail отрисовывает детальную информацию о запросе (TUI-04)
func (m Model) renderRequestDetail() string {
	req := m.requestsState.detailRequest
	if req == nil {
		return ""
	}

	var content strings.Builder

	// Заголовок
	content.WriteString(titleStyle.Render("🔍 ДЕТАЛИ ЗАПРОСА") + "\n\n")

	// ID и время
	content.WriteString(fmt.Sprintf("🆔 ID: %s\n", req.ID))
	timestamp := time.Unix(req.Timestamp, 0).Format("2006-01-02 15:04:05")
	content.WriteString(fmt.Sprintf("⏰ Время: %s\n", timestamp))
	content.WriteString(fmt.Sprintf("⏱️  Длительность: %s\n", formatDuration(req.DurationMS)))
	content.WriteString("\n")

	// Request информация
	content.WriteString(subtitleStyle.Render("📤 REQUEST:") + "\n")
	content.WriteString(fmt.Sprintf("  Метод:    %s\n", req.Method))
	content.WriteString(fmt.Sprintf("  Endpoint: %s\n", req.Endpoint))
	if req.Model != "" {
		content.WriteString(fmt.Sprintf("  Модель:   %s\n", req.Model))
	}
	if req.APIKeyName != "" {
		content.WriteString(fmt.Sprintf("  API Key:  %s (%s)\n", req.APIKeyName, req.APIKeyID))
	}
	content.WriteString(fmt.Sprintf("  IP:       %s\n", req.RemoteAddr))
	if req.UserAgent != "" {
		content.WriteString(fmt.Sprintf("  Agent:    %s\n", req.UserAgent))
	}
	content.WriteString("\n")

	// Request параметры (для chat/completions)
	if req.Messages > 0 || req.Tools > 0 {
		content.WriteString(subtitleStyle.Render("⚙️  ПАРАМЕТРЫ:") + "\n")
		if req.Messages > 0 {
			content.WriteString(fmt.Sprintf("  Сообщений: %d\n", req.Messages))
		}
		if req.Tools > 0 {
			content.WriteString(fmt.Sprintf("  Tools:     %d\n", req.Tools))
		}
		if req.Stream {
			content.WriteString("  Streaming: ✅ Включен\n")
		}
		if req.Temperature != nil {
			content.WriteString(fmt.Sprintf("  Temperature: %.2f\n", *req.Temperature))
		}
		content.WriteString("\n")
	}

	// Response информация
	content.WriteString(subtitleStyle.Render("📥 RESPONSE:") + "\n")
	statusStr := m.formatRequestStatus(req.Status)
	content.WriteString(fmt.Sprintf("  Статус:   %s\n", statusStr))
	if req.StatusCode > 0 {
		content.WriteString(fmt.Sprintf("  HTTP:     %d\n", req.StatusCode))
	}
	if req.TotalTokens > 0 {
		content.WriteString(fmt.Sprintf("  Токены:   %d\n", req.TotalTokens))
	}
	content.WriteString("\n")

	// Ошибка (если есть)
	if req.ErrorMessage != "" {
		content.WriteString(subtitleStyle.Render("❌ ОШИБКА:") + "\n")
		content.WriteString(errorStyle.Render("  "+req.ErrorMessage) + "\n\n")
	}

	// Подсказка
	content.WriteString(helpStyle.Render("Нажмите Esc чтобы вернуться к списку"))

	return borderStyle.Render(content.String())
}

// renderModels отрисовывает экран управления моделями
func (m Model) renderModels() string {
	content := "🤖 УПРАВЛЕНИЕ МОДЕЛЯМИ\n\n"

	if m.stats.Ollama.Connected {
		content += fmt.Sprintf("📋 Доступно моделей: %d\n\n", m.stats.Ollama.ModelsCount)

		if len(m.stats.Ollama.Models) > 0 {
			for i, model := range m.stats.Ollama.Models {
				content += fmt.Sprintf("%d. %s\n", i+1, model)
			}
		} else {
			content += helpStyle.Render("Нет загруженных моделей")
		}
	} else {
		content += errorStyle.Render("⚠️  Ollama не подключен\n\n")
		content += helpStyle.Render("Список моделей будет доступен после подключения к Ollama серверу")
	}

	return borderStyle.Render(content)
}

// renderAPIKeys отрисовывает экран управления API ключами
func (m Model) renderAPIKeys() string {
	content := "🔑 УПРАВЛЕНИЕ API КЛЮЧАМИ\n\n"

	if !m.stats.APIKeys.Enabled {
		content += warningStyle.Render("⚠️  API Key аутентификация отключена в конфигурации") + "\n\n"
		content += helpStyle.Render("Включите auth.enabled в configs/dev.yaml для использования API ключей")
		return borderStyle.Render(content)
	}

	// Показываем модальное окно подтверждения (AUTH-04)
	if m.showConfirm {
		return m.renderConfirmDialog()
	}

	// Показываем форму редактирования (AUTH-04)
	if m.editingKey {
		return m.renderEditKeyForm()
	}

	// Показываем форму создания ключа
	if m.creatingKey {
		return m.renderCreateKeyForm()
	}

	// Показываем созданный ключ (только один раз!)
	if m.createdKey != "" {
		content := "🎉 " + statusStyle.Render("API КЛЮЧ УСПЕШНО СОЗДАН!") + "\n\n"
		content += errorStyle.Render("⚠️  ВНИМАНИЕ: Ключ показывается только один раз!") + "\n"
		content += errorStyle.Render("⚠️  Скопируйте его прямо СЕЙЧАС!") + "\n\n"

		// Рамка вокруг ключа для заметности
		keyBox := lipgloss.NewStyle().
			Border(lipgloss.DoubleBorder()).
			BorderForeground(lipgloss.Color("#FFD700")).
			Padding(1, 2).
			Render(m.createdKey)

		content += "🔑 ВАШ НОВЫЙ API КЛЮЧ:\n\n"
		content += keyBox + "\n\n"

		// Показываем сообщение об успешном копировании если есть
		if m.successMessage != "" {
			content += statusStyle.Render(m.successMessage) + "\n\n"
		}

		content += statusStyle.Render("📋 Нажмите 'c' чтобы СКОПИРОВАТЬ в буфер обмена") + "\n"
		content += helpStyle.Render("🔧 Используйте в заголовке: Authorization: Bearer <key>") + "\n\n"
		content += helpStyle.Render("Любая другая клавиша закроет экран и обновит список...")

		return content
	}

	// Показываем сообщение об успехе
	if m.successMessage != "" {
		content += statusStyle.Render(m.successMessage) + "\n\n"
	}

	content += fmt.Sprintf("📊 Всего ключей: %d\n\n", m.stats.APIKeys.Count)

	if len(m.stats.APIKeys.Keys) == 0 {
		content += helpStyle.Render("Нет созданных API ключей\n\n")
		content += "Создайте первый ключ через Admin API:\n"
		content += "POST /admin/api-keys"
		return borderStyle.Render(content)
	}

	// Таблица с ключами (адаптивная к ширине экрана)
	if m.width < 100 {
		// Узкая версия таблицы
		content += "┌──────────────────┬──────────┬──────────┐\n"
		content += "│ Имя (ID)         │ Статус   │ Запросов │\n"
		content += "├──────────────────┼──────────┼──────────┤\n"

		for i, key := range m.stats.APIKeys.Keys {
			name := key.Name
			if len(name) > 14 {
				name = name[:11] + "..."
			}
			// Добавляем последние 4 символа ID
			shortID := key.ID
			if len(shortID) > 4 {
				shortID = shortID[len(shortID)-4:]
			}
			nameWithID := fmt.Sprintf("%s (%s)", name, shortID)

			status := key.Status
			statusColor := helpStyle
			if status == "active" {
				statusColor = statusStyle
			} else if status == "revoked" {
				statusColor = errorStyle
			} else {
				statusColor = warningStyle
			}

			// Подсвечиваем выбранный ключ (AUTH-04)
			rowPrefix := "│"
			if i == m.selectedKeyIndex {
				rowPrefix = "▶"
			}

			content += fmt.Sprintf("%s %-16s │ %-8s │ %8d │\n",
				rowPrefix,
				nameWithID,
				statusColor.Render(status),
				key.Usage.TotalRequests,
			)
		}

		content += "└──────────────────┴──────────┴──────────┘\n\n"
	} else {
		// Полная версия таблицы
		content += "┌──────────────────┬─────────┬──────────┬────────────┬──────────┬─────────────┐\n"
		content += "│ Имя (ID)         │ Статус  │ Перм.    │ Модели     │ Запросов │ Последнее   │\n"
		content += "├──────────────────┼─────────┼──────────┼────────────┼──────────┼─────────────┤\n"

		for i, key := range m.stats.APIKeys.Keys {
			// Форматируем имя с последними символами ID
			name := key.Name
			if len(name) > 10 {
				name = name[:7] + "..."
			}
			shortID := key.ID
			if len(shortID) > 4 {
				shortID = shortID[len(shortID)-4:]
			}
			nameWithID := fmt.Sprintf("%s (%s)", name, shortID)

			// Статус с цветом
			status := key.Status
			if len(status) > 7 {
				status = status[:7]
			}
			statusColor := helpStyle
			if key.Status == "active" {
				statusColor = statusStyle
			} else if key.Status == "revoked" {
				statusColor = errorStyle
			} else {
				statusColor = warningStyle
			}

			// Разрешения (первое или *)
			perms := "*"
			if len(key.Permissions) > 0 {
				perms = key.Permissions[0]
				if len(perms) > 8 {
					perms = perms[:5] + "..."
				}
				if len(key.Permissions) > 1 {
					perms += fmt.Sprintf("+%d", len(key.Permissions)-1)
				}
			}

			// Модели (первая или *)
			models := "*"
			if len(key.Models) > 0 && key.Models[0] != "*" {
				models = key.Models[0]
				if len(models) > 10 {
					models = models[:7] + "..."
				}
				if len(key.Models) > 1 {
					models += fmt.Sprintf("+%d", len(key.Models)-1)
				}
			}

			// Подсвечиваем выбранный ключ (AUTH-04)
			rowPrefix := "│"
			if i == m.selectedKeyIndex {
				rowPrefix = "▶"
			}

			content += fmt.Sprintf("%s %-16s │ %-7s │ %-8s │ %-10s │ %8d │ %-11s │\n",
				rowPrefix,
				nameWithID,
				statusColor.Render(status),
				perms,
				models,
				key.Usage.TotalRequests,
				key.LastUsed,
			)
		}

		content += "└──────────────────┴─────────┴──────────┴────────────┴──────────┴─────────────┘\n\n"
	}

	content += helpStyle.Render("💡 Клавиши: ↑↓/j/k - выбор | 'n' - создать | 'e' - редактировать | 'x' - revoke/enable")

	return borderStyle.Render(content)
}

// renderCreateKeyForm отрисовывает форму создания API ключа
func (m Model) renderCreateKeyForm() string {
	content := "🔑 СОЗДАНИЕ НОВОГО API КЛЮЧА\n\n"

	// Поле имени
	nameLabel := "Имя ключа:"
	if m.createForm.CurrentField == 0 {
		nameLabel = statusStyle.Render("▶ Имя ключа:")
	}
	content += nameLabel + "\n"
	content += borderStyle.Render(m.createForm.Name+"_") + "\n\n"

	// Поле описания
	descLabel := "Описание (опционально):"
	if m.createForm.CurrentField == 1 {
		descLabel = statusStyle.Render("▶ Описание (опционально):")
	}
	content += descLabel + "\n"
	desc := m.createForm.Description
	if desc == "" {
		desc = "-"
	}
	if m.createForm.CurrentField == 1 {
		desc += "_"
	}
	content += borderStyle.Render(desc) + "\n\n"

	// Поле моделей
	modelsLabel := "Модели (через запятую, * = все):"
	if m.createForm.CurrentField == 2 {
		modelsLabel = statusStyle.Render("▶ Модели (через запятую, * = все):")
	}
	content += modelsLabel + "\n"
	models := m.createForm.Models
	if models == "" {
		models = "*"
	}
	if m.createForm.CurrentField == 2 {
		models += "_"
	}
	content += borderStyle.Render(models) + "\n\n"

	// Подсказки
	content += helpStyle.Render("Tab - следующее поле | Enter - создать | Esc - отмена")

	return borderStyle.Render(content)
}

// renderConfig отрисовывает экран конфигурации
func (m Model) renderConfig() string {
	content := "⚙️ КОНФИГУРАЦИЯ СЕРВЕРА\n\n"

	// Проверяем что конфигурация загружена
	if m.config.Server.Port == 0 {
		content += helpStyle.Render("⏳ Загрузка конфигурации...")
		return borderStyle.Render(content)
	}

	// 🌐 СЕРВЕР
	content += "🌐 SERVER\n"
	content += fmt.Sprintf("• Host: %s\n", m.config.Server.Host)
	content += fmt.Sprintf("• Port: %d\n", m.config.Server.Port)
	content += fmt.Sprintf("• Read Timeout: %s\n", m.config.Server.ReadTimeout)
	content += fmt.Sprintf("• Write Timeout: %s\n", m.config.Server.WriteTimeout)
	content += fmt.Sprintf("• Idle Timeout: %s\n", m.config.Server.IdleTimeout)
	content += fmt.Sprintf("• Max Header Bytes: %d\n", m.config.Server.MaxHeaderBytes)
	content += "\n"

	// 🦙 OLLAMA
	content += "🦙 OLLAMA\n"
	content += fmt.Sprintf("• URL: %s\n", m.config.Ollama.URL)
	content += fmt.Sprintf("• Timeout: %s\n", m.config.Ollama.Timeout)
	content += fmt.Sprintf("• Retry Attempts: %d\n", m.config.Ollama.RetryAttempts)
	content += fmt.Sprintf("• Retry Delay: %s\n", m.config.Ollama.RetryDelay)
	content += fmt.Sprintf("• Connection Pool: %d\n", m.config.Ollama.ConnectionPoolSize)
	content += fmt.Sprintf("• Keep Alive: %v\n", m.config.Ollama.KeepAlive)
	content += "\n"

	// 🔐 AUTH
	content += "🔐 AUTHENTICATION\n"
	content += fmt.Sprintf("• Enabled: %v\n", m.config.Auth.Enabled)
	content += fmt.Sprintf("• Storage Type: %s\n", m.config.Auth.StorageType)
	content += fmt.Sprintf("• Storage Path: %s\n", m.config.Auth.StoragePath)
	content += fmt.Sprintf("• Rate Limiting: %v\n", m.config.Auth.RateLimiting.Enabled)
	if m.config.Auth.RateLimiting.Enabled {
		content += fmt.Sprintf("  - Requests/Minute: %d\n", m.config.Auth.RateLimiting.DefaultRequestsPerMinute)
		content += fmt.Sprintf("  - Requests/Hour: %d\n", m.config.Auth.RateLimiting.DefaultRequestsPerHour)
	}
	content += "\n"

	// 📝 LOGGING
	content += "📝 LOGGING\n"
	content += fmt.Sprintf("• Level: %s\n", m.config.Logging.Level)
	content += fmt.Sprintf("• Format: %s\n", m.config.Logging.Format)
	content += fmt.Sprintf("• Output: %s\n", m.config.Logging.Output)
	if m.config.Logging.FilePath != "" {
		content += fmt.Sprintf("• File Path: %s\n", m.config.Logging.FilePath)
		content += fmt.Sprintf("• Max Size: %d MB\n", m.config.Logging.MaxSize)
		content += fmt.Sprintf("• Max Backups: %d\n", m.config.Logging.MaxBackups)
		content += fmt.Sprintf("• Max Age: %d days\n", m.config.Logging.MaxAge)
		content += fmt.Sprintf("• Compress: %v\n", m.config.Logging.Compress)
	}
	content += "\n"

	// 🤖 MODELS
	content += "🤖 MODELS\n"
	if len(m.config.Models.Mapping) > 0 {
		content += fmt.Sprintf("• Mappings: %d configured\n", len(m.config.Models.Mapping))
	}
	if len(m.config.Models.Aliases) > 0 {
		content += fmt.Sprintf("• Aliases: %d configured\n", len(m.config.Models.Aliases))
	}
	if len(m.config.Models.Hidden) > 0 {
		content += fmt.Sprintf("• Hidden Models: %d\n", len(m.config.Models.Hidden))
	}
	content += fmt.Sprintf("• Cache Enabled: %v\n", m.config.Models.Cache.Enabled)
	if m.config.Models.Cache.Enabled {
		content += fmt.Sprintf("  - TTL: %s\n", m.config.Models.Cache.TTL)
		content += fmt.Sprintf("  - Refresh: %s\n", m.config.Models.Cache.RefreshInterval)
	}
	content += "\n"

	// 🔧 TOOLS
	content += "🔧 TOOLS (Function Calling)\n"
	content += fmt.Sprintf("• Force Usage: %v\n", m.config.Tools.ForceUsage)
	content += fmt.Sprintf("• Default Choice: %s\n", m.config.Tools.DefaultChoice)
	if m.config.Tools.FallbackModel != "" {
		content += fmt.Sprintf("• Fallback Model: %s\n", m.config.Tools.FallbackModel)
	}
	content += "\n"

	// ⚡ OPTIMIZER
	content += "⚡ PROMPT OPTIMIZER\n"
	content += fmt.Sprintf("• Enabled: %v\n", m.config.Tools.Optimizer.Enabled)
	if m.config.Tools.Optimizer.Enabled {
		content += fmt.Sprintf("• Simplify System Message: %v\n", m.config.Tools.Optimizer.SimplifySystemMessage)
		content += fmt.Sprintf("• Smart Tool Filtering: %v\n", m.config.Tools.Optimizer.SmartToolFiltering)
		content += fmt.Sprintf("• Max Tools Per Request: %d\n", m.config.Tools.Optimizer.MaxToolsPerRequest)
		if len(m.config.Tools.Optimizer.PreserveInstructions) > 0 {
			content += fmt.Sprintf("• Preserve Instructions: %d\n", len(m.config.Tools.Optimizer.PreserveInstructions))
		}
	}
	content += "\n"

	// 📊 METRICS
	content += "📊 METRICS\n"
	content += fmt.Sprintf("• Enabled: %v\n", m.config.Metrics.Enabled)
	if m.config.Metrics.Enabled {
		content += fmt.Sprintf("• Prometheus Path: %s\n", m.config.Metrics.PrometheusPath)
	}
	content += "\n"

	// 🖥️ TUI
	content += "🖥️ TUI\n"
	content += fmt.Sprintf("• Enabled: %v\n", m.config.TUI.Enabled)
	content += fmt.Sprintf("• Refresh Rate: %s\n", m.config.TUI.RefreshRate)
	content += fmt.Sprintf("• Theme: %s\n", m.config.TUI.Theme)
	content += "\n"

	// 🔬 DEVELOPMENT
	content += "🔬 DEVELOPMENT\n"
	content += fmt.Sprintf("• Hot Reload: %v\n", m.config.Development.HotReload)
	content += fmt.Sprintf("• Debug Mode: %v\n", m.config.Development.DebugMode)
	content += fmt.Sprintf("• Profile Enabled: %v\n", m.config.Development.ProfileEnabled)
	content += fmt.Sprintf("• PProf Enabled: %v\n", m.config.Development.PProfEnabled)
	content += fmt.Sprintf("• Race Detection: %v\n", m.config.Development.RaceDetection)

	return borderStyle.Render(content)
}

// renderLogs отрисовывает экран просмотра логов
func (m Model) renderLogs() string {
	content := "📋 ПРОСМОТР ЛОГОВ СЕРВЕРА\n\n"

	// Путь к файлу логов
	logPath := "logs/proxy-dev.log"

	// Читаем последние 100 строк
	entries, err := readLastLogs(logPath, 100)
	if err != nil {
		content += errorStyle.Render(fmt.Sprintf("❌ Ошибка чтения логов: %v", err)) + "\n\n"
		content += helpStyle.Render("Проверьте что сервер запущен и файл логов существует:\n")
		content += helpStyle.Render(fmt.Sprintf("  %s", logPath))
		return borderStyle.Render(content)
	}

	// Если логов нет
	if len(entries) == 0 {
		content += helpStyle.Render("📝 Логи пока отсутствуют\n")
		content += helpStyle.Render("Запустите сервер чтобы увидеть логи здесь")
		return borderStyle.Render(content)
	}

	// Статистика по уровням
	stats := make(map[string]int)
	for _, entry := range entries {
		stats[entry.Level]++
	}

	content += fmt.Sprintf("📊 Статистика: ")
	if stats["ERROR"] > 0 {
		content += errorStyle.Render(fmt.Sprintf("ERROR: %d ", stats["ERROR"]))
	}
	if stats["WARN"] > 0 {
		content += warningStyle.Render(fmt.Sprintf("WARN: %d ", stats["WARN"]))
	}
	if stats["INFO"] > 0 {
		content += fmt.Sprintf("INFO: %d ", stats["INFO"])
	}
	if stats["DEBUG"] > 0 {
		content += helpStyle.Render(fmt.Sprintf("DEBUG: %d", stats["DEBUG"]))
	}
	content += "\n\n"

	// Выводим логи с цветовым кодированием
	content += fmt.Sprintf("📄 Последние %d строк (из %s):\n\n", len(entries), logPath)

	for _, entry := range entries {
		content += colorizeLogLine(entry) + "\n"
	}

	content += "\n"
	content += helpStyle.Render("💡 Используйте прокрутку (↑↓/PgUp/PgDn) для просмотра всех логов")

	return content // Не оборачиваем в borderStyle - пусть прокрутка работает лучше
}

// renderControl отрисовывает экран управления сервером
func (m Model) renderControl() string {
	content := "🎛️ УПРАВЛЕНИЕ СЕРВЕРОМ\n\n"

	// 🟢 СТАТУС HTTP СЕРВЕРА
	content += "🌐 HTTP SERVER\n"
	if m.stats.Server.Status == "running" {
		content += statusStyle.Render("• Статус: ✅ Активен") + "\n"
		content += fmt.Sprintf("• Адрес: http://%s:%d\n", m.config.Server.Host, m.config.Server.Port)
		content += fmt.Sprintf("• Uptime: %s\n", m.stats.Stats.Uptime)
		if m.stats.Stats.Uptime != "" {
			content += fmt.Sprintf("• Запущен: %s\n", time.Now().Add(-parseDuration(m.stats.Stats.Uptime)).Format("2006-01-02 15:04:05"))
		}
	} else {
		content += errorStyle.Render("• Статус: ❌ Недоступен") + "\n"
		content += helpStyle.Render("  Сервер не отвечает на запросы\n")
		content += helpStyle.Render("  Проверьте что сервер запущен\n")
	}
	content += "\n"

	// 🦙 СТАТУС OLLAMA
	content += "🦙 OLLAMA CONNECTION\n"
	if m.stats.Ollama.Connected {
		content += statusStyle.Render("• Статус: ✅ Подключен") + "\n"
		content += fmt.Sprintf("• URL: %s\n", m.config.Ollama.URL)
		content += fmt.Sprintf("• Доступно моделей: %d\n", m.stats.Ollama.ModelsCount)
		if len(m.stats.Ollama.Models) > 0 {
			content += fmt.Sprintf("• Модели: %s\n", strings.Join(m.stats.Ollama.Models[:min(3, len(m.stats.Ollama.Models))], ", "))
			if len(m.stats.Ollama.Models) > 3 {
				content += fmt.Sprintf("  ... и еще %d\n", len(m.stats.Ollama.Models)-3)
			}
		}
	} else {
		content += errorStyle.Render("• Статус: ❌ Недоступен") + "\n"
		content += helpStyle.Render(fmt.Sprintf("  URL: %s\n", m.config.Ollama.URL))
		content += warningStyle.Render("  ⚠️  Проверьте что Ollama запущен:\n")
		content += helpStyle.Render("     ollama serve\n")
	}
	content += "\n"

	// 🖥️ СТАТУС TUI
	content += "🖥️ TUI STATUS\n"
	content += statusStyle.Render("• Статус: ✅ Активен") + "\n"
	content += fmt.Sprintf("• Refresh Rate: %s\n", m.config.TUI.RefreshRate)
	if !m.lastUpdate.IsZero() {
		content += fmt.Sprintf("• Последнее обновление: %s\n", m.lastUpdate.Format("15:04:05"))
	}
	content += "\n"

	// 📊 СТАТИСТИКА
	content += "📊 СТАТИСТИКА\n"
	content += fmt.Sprintf("• Всего запросов: %d\n", m.stats.Stats.TotalRequests)
	content += fmt.Sprintf("• Успешных: %d\n", m.stats.Stats.SuccessRequests)
	if m.stats.Stats.ErrorRequests > 0 {
		content += errorStyle.Render(fmt.Sprintf("• Ошибок: %d\n", m.stats.Stats.ErrorRequests))
	} else {
		content += fmt.Sprintf("• Ошибок: %d\n", m.stats.Stats.ErrorRequests)
	}
	content += fmt.Sprintf("• Активных: %d\n", m.stats.Stats.ActiveRequests)
	content += fmt.Sprintf("• Средняя длительность: %s\n", m.stats.Stats.AverageDuration)
	content += fmt.Sprintf("• Активных ключей: %d\n", len(m.stats.APIKeys.Keys))
	content += "\n"

	// 💡 ПОЛЕЗНЫЕ КОМАНДЫ
	content += "💡 ПОЛЕЗНЫЕ КОМАНДЫ\n"
	content += helpStyle.Render("Перезапуск сервера (Linux):") + "\n"
	content += "  systemctl restart ollama-proxy\n"
	content += helpStyle.Render("Просмотр логов в реальном времени:") + "\n"
	content += "  tail -f logs/proxy-dev.log\n"
	content += helpStyle.Render("Проверка Ollama:") + "\n"
	content += "  curl http://localhost:11434/api/tags\n"
	content += "\n"

	// 🎮 БЫСТРЫЕ ДЕЙСТВИЯ
	content += "🎮 БЫСТРЫЕ ДЕЙСТВИЯ\n"
	content += "• [R] Обновить статистику\n"
	content += "• [1-7] Перейти к другим экранам\n"
	content += "• [Q] Выход из TUI\n"

	return content // Без borderStyle для лучшей прокрутки
}

// cycleSortField переключает поле сортировки запросов (TUI-04)
func (m Model) cycleSortField() tea.Cmd {
	// Цикл: time -> duration -> status -> endpoint -> time
	switch m.requestsState.sortField {
	case "time":
		m.requestsState.sortField = "duration"
	case "duration":
		m.requestsState.sortField = "status"
	case "status":
		m.requestsState.sortField = "endpoint"
	case "endpoint":
		m.requestsState.sortField = "time"
	}

	// Сбрасываем на первую страницу
	m.requestsState.page = 1
	m.requestsState.selectedIndex = 0

	return fetchRequestsCmd(m.serverURL, m.requestsState)
}

// cycleStatusFilter переключает фильтр по статусу (TUI-04)
func (m Model) cycleStatusFilter() tea.Cmd {
	// Цикл: all -> success -> error -> pending -> all
	switch m.requestsState.statusFilter {
	case "all":
		m.requestsState.statusFilter = "success"
	case "success":
		m.requestsState.statusFilter = "error"
	case "error":
		m.requestsState.statusFilter = "pending"
	case "pending":
		m.requestsState.statusFilter = "all"
	}

	// Сбрасываем на первую страницу
	m.requestsState.page = 1
	m.requestsState.selectedIndex = 0

	return fetchRequestsCmd(m.serverURL, m.requestsState)
}

// exportRequests экспортирует текущие запросы в CSV и JSON (TUI-04)
func (m Model) exportRequests() error {
	if len(m.requestsState.requests) == 0 {
		return fmt.Errorf("нет запросов для экспорта")
	}

	// Создаем директорию logs если её нет
	if err := ensureLogDir(); err != nil {
		return fmt.Errorf("не удалось создать директорию logs: %w", err)
	}

	timestamp := time.Now().Format("20060102-150405")

	// Экспорт в CSV
	csvFile := fmt.Sprintf("logs/requests-%s.csv", timestamp)
	if err := m.exportToCSV(csvFile); err != nil {
		return fmt.Errorf("ошибка экспорта в CSV: %w", err)
	}

	// Экспорт в JSON
	jsonFile := fmt.Sprintf("logs/requests-%s.json", timestamp)
	if err := m.exportToJSON(jsonFile); err != nil {
		return fmt.Errorf("ошибка экспорта в JSON: %w", err)
	}

	return nil
}

// exportToCSV экспортирует запросы в CSV формат
func (m Model) exportToCSV(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Заголовок CSV
	header := []string{
		"ID", "Timestamp", "Method", "Endpoint", "Status", "Duration_ms",
		"Model", "API_Key_Name", "Remote_Addr", "User_Agent",
		"Status_Code", "Messages", "Tools", "Stream", "Temperature",
		"Total_Tokens", "Error",
	}
	if err := writer.Write(header); err != nil {
		return err
	}

	// Записываем данные
	for _, req := range m.requestsState.requests {
		timestamp := time.Unix(req.Timestamp, 0).Format("2006-01-02 15:04:05")

		temperature := ""
		if req.Temperature != nil {
			temperature = fmt.Sprintf("%.2f", *req.Temperature)
		}

		row := []string{
			req.ID,
			timestamp,
			req.Method,
			req.Endpoint,
			req.Status,
			fmt.Sprintf("%d", req.DurationMS),
			req.Model,
			req.APIKeyName,
			req.RemoteAddr,
			req.UserAgent,
			fmt.Sprintf("%d", req.StatusCode),
			fmt.Sprintf("%d", req.Messages),
			fmt.Sprintf("%d", req.Tools),
			fmt.Sprintf("%t", req.Stream),
			temperature,
			fmt.Sprintf("%d", req.TotalTokens),
			req.ErrorMessage,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// exportToJSON экспортирует запросы в JSON формат
func (m Model) exportToJSON(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	exportData := map[string]interface{}{
		"exported_at": time.Now().Format(time.RFC3339),
		"total":       len(m.requestsState.requests),
		"filter":      m.requestsState.statusFilter,
		"sort_field":  m.requestsState.sortField,
		"sort_order":  m.requestsState.sortOrder,
		"requests":    m.requestsState.requests,
	}

	return encoder.Encode(exportData)
}

// ensureLogDir создает директорию logs если её нет
func ensureLogDir() error {
	return os.MkdirAll("logs", 0755)
}

// Сообщения для Bubble Tea

type statsMsg ServerStats
type metricsMsg *MetricsSnapshot
type configMsg ServerConfig
type errMsg error
type tickMsg time.Time
type keyCreatedMsg struct {
	plainKey string
	keyID    string
	keyName  string
}
type requestsMsg RequestsResponse

// handleCreateKeyInput обрабатывает ввод в форме создания ключа
func (m Model) handleCreateKeyInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		// Отмена создания
		m.creatingKey = false
		m.createForm = CreateKeyForm{}
		return m, nil

	case "tab":
		// Переход к следующему полю
		m.createForm.CurrentField = (m.createForm.CurrentField + 1) % 3
		return m, nil

	case "enter":
		// Отправка формы
		if m.createForm.Name == "" {
			m.errorMessage = "Имя ключа обязательно"
			return m, nil
		}

		// Отправляем запрос на создание
		m.creatingKey = false
		return m, createAPIKeyCmd(m.serverURL, m.adminToken, m.createForm)

	case "backspace":
		// Удаление символа
		switch m.createForm.CurrentField {
		case 0:
			if len(m.createForm.Name) > 0 {
				m.createForm.Name = m.createForm.Name[:len(m.createForm.Name)-1]
			}
		case 1:
			if len(m.createForm.Description) > 0 {
				m.createForm.Description = m.createForm.Description[:len(m.createForm.Description)-1]
			}
		case 2:
			if len(m.createForm.Models) > 0 {
				m.createForm.Models = m.createForm.Models[:len(m.createForm.Models)-1]
			}
		}
		return m, nil

	default:
		// Добавление символа
		if len(msg.String()) == 1 {
			switch m.createForm.CurrentField {
			case 0:
				m.createForm.Name += msg.String()
			case 1:
				m.createForm.Description += msg.String()
			case 2:
				m.createForm.Models += msg.String()
			}
		}
		return m, nil
	}
}

// fetchStatsCmd загружает статистику с сервера
func fetchStatsCmd(serverURL string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{
			Timeout: 3 * time.Second,
		}

		resp, err := client.Get(serverURL + "/api/stats")
		if err != nil {
			return errMsg(fmt.Errorf("не удалось подключиться к серверу: %v", err))
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return errMsg(fmt.Errorf("сервер вернул ошибку %d: %s", resp.StatusCode, string(body)))
		}

		var stats ServerStats
		if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
			return errMsg(fmt.Errorf("ошибка парсинга ответа: %v", err))
		}

		return statsMsg(stats)
	}
}

// fetchMetricsCmd загружает Prometheus метрики с сервера
func fetchMetricsCmd(serverURL string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{
			Timeout: 3 * time.Second,
		}

		resp, err := client.Get(serverURL + "/metrics")
		if err != nil {
			// Не показываем ошибку - метрики опциональны
			return nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil
		}

		metricsSnapshot, err := ParsePrometheusMetrics(resp.Body)
		if err != nil {
			// Не показываем ошибку - метрики опциональны
			return nil
		}

		return metricsMsg(metricsSnapshot)
	}
}

// fetchConfigCmd загружает конфигурацию с сервера
func fetchConfigCmd(serverURL string) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{
			Timeout: 3 * time.Second,
		}

		resp, err := client.Get(serverURL + "/api/config")
		if err != nil {
			// Не показываем ошибку - конфигурация опциональна
			return nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil
		}

		var config ServerConfig
		if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
			// Не показываем ошибку - конфигурация опциональна
			return nil
		}

		return configMsg(config)
	}
}

// fetchRequestsCmd загружает список запросов с сервера (TUI-04)
func fetchRequestsCmd(serverURL string, state RequestsState) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{
			Timeout: 3 * time.Second,
		}

		// Формируем query parameters
		url := fmt.Sprintf("%s/api/requests?page=%d&per_page=%d&sort=%s&order=%s",
			serverURL, state.page, state.perPage, state.sortField, state.sortOrder)

		// Добавляем фильтры
		if state.statusFilter != "" && state.statusFilter != "all" {
			url += "&status=" + state.statusFilter
		}

		resp, err := client.Get(url)
		if err != nil {
			// Не показываем ошибку - запросы опциональны
			return nil
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil
		}

		var requests RequestsResponse
		if err := json.NewDecoder(resp.Body).Decode(&requests); err != nil {
			return nil
		}

		return requestsMsg(requests)
	}
}

// createAPIKeyCmd отправляет запрос на создание API ключа
func createAPIKeyCmd(serverURL string, adminToken string, form CreateKeyForm) tea.Cmd {
	return func() tea.Msg {
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		// Формируем модели (по умолчанию "*")
		models := []string{"*"}
		if form.Models != "" {
			models = []string{}
			for _, m := range splitAndTrim(form.Models) {
				if m != "" {
					models = append(models, m)
				}
			}
		}

		// Формируем тело запроса
		requestBody := map[string]interface{}{
			"name":        form.Name,
			"description": form.Description,
			"models":      models,
			"permissions": []string{"chat", "models"}, // Базовые разрешения
		}

		bodyBytes, err := json.Marshal(requestBody)
		if err != nil {
			return errMsg(fmt.Errorf("ошибка формирования запроса: %v", err))
		}

		// Отправляем запрос (используем admin ключ из конфига)
		req, err := http.NewRequest("POST", serverURL+"/admin/api-keys", bytes.NewBuffer(bodyBytes))
		if err != nil {
			return errMsg(fmt.Errorf("ошибка создания запроса: %v", err))
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+adminToken)

		resp, err := client.Do(req)
		if err != nil {
			return errMsg(fmt.Errorf("ошибка отправки запроса: %v", err))
		}
		defer resp.Body.Close()

		// Проверяем успешные статусы (200 OK или 201 Created)
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			body, _ := io.ReadAll(resp.Body)
			return errMsg(fmt.Errorf("ошибка создания ключа (status %d): %s", resp.StatusCode, string(body)))
		}

		var response struct {
			APIKey struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"api_key"`
			PlainKey string `json:"plain_key"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return errMsg(fmt.Errorf("ошибка парсинга ответа: %v", err))
		}

		return keyCreatedMsg{
			plainKey: response.PlainKey,
			keyID:    response.APIKey.ID,
			keyName:  response.APIKey.Name,
		}
	}
}

// splitAndTrim разбивает строку по запятой и убирает пробелы
func splitAndTrim(s string) []string {
	parts := []string{}
	for _, p := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// renderHelpScreen отрисовывает экран помощи
func (m Model) renderHelpScreen() string {
	header := titleStyle.Render("🦙 Ollama-OpenAI Proxy TUI - Помощь") + "\n\n"

	sections := []struct {
		title string
		items []struct {
			key  string
			desc string
		}
	}{
		{
			title: "📊 НАВИГАЦИЯ",
			items: []struct {
				key  string
				desc string
			}{
				{"1", "Dashboard - Общая информация и статистика"},
				{"2", "Requests - Мониторинг запросов (в разработке)"},
				{"3", "Models - Список доступных моделей"},
				{"4", "API Keys - Управление API ключами"},
				{"5", "Config - Конфигурация сервера"},
				{"6", "Logs - Просмотр логов сервера"},
				{"7", "Control - Управление сервером"},
			},
		},
		{
			title: "⌨️  КЛАВИШИ",
			items: []struct {
				key  string
				desc string
			}{
				{"↑↓ или j/k", "Прокрутка контента"},
				{"PgUp/PgDn", "Быстрая прокрутка (10 строк)"},
				{"Home/End", "В начало/конец"},
				{"h или ?", "Показать/скрыть эту помощь"},
				{"r", "Обновить данные"},
				{"Esc", "Закрыть ошибку/отменить действие"},
				{"q или Ctrl+C", "Выход из TUI"},
			},
		},
		{
			title: "🔑 API KEYS",
			items: []struct {
				key  string
				desc string
			}{
				{"n", "Создать новый API ключ (на экране Keys)"},
				{"Tab", "Переключение между полями формы"},
				{"Enter", "Подтвердить создание ключа"},
				{"c", "Скопировать созданный ключ в буфер обмена"},
				{"Esc", "Отменить создание / закрыть окно ключа"},
			},
		},
		{
			title: "🖱️  МЫШЬ",
			items: []struct {
				key  string
				desc string
			}{
				{"Колесо мыши", "Прокрутка контента (быстрее клавиатуры)"},
				{"Клик на вкладке", "Переключение между экранами"},
			},
		},
		{
			title: "💡 СОВЕТЫ",
			items: []struct {
				key  string
				desc string
			}{
				{"", "• Данные обновляются автоматически каждые 2 секунды"},
				{"", "• Используйте 'r' для принудительного обновления"},
				{"", "• Ошибки остаются на экране - нажмите Esc чтобы закрыть"},
				{"", "• Прокрутка сбрасывается при смене экрана"},
				{"", "• Ctrl+C безопасно завершает TUI"},
			},
		},
	}

	var content strings.Builder

	for _, section := range sections {
		content.WriteString(titleStyle.Render(section.title) + "\n")
		for _, item := range section.items {
			if item.key != "" {
				content.WriteString(statusStyle.Render(fmt.Sprintf("  %-15s", item.key)))
				content.WriteString(helpStyle.Render(" → " + item.desc + "\n"))
			} else {
				content.WriteString(helpStyle.Render("  " + item.desc + "\n"))
			}
		}
		content.WriteString("\n")
	}

	footer := "\n" + warningStyle.Render("Нажмите 'h' или '?' чтобы закрыть справку")

	return header + content.String() + footer
}

// tickCmd создает периодический тик для обновления данных
func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// === AUTH-04: Enhanced API Key Management Functions ===

// updateSelectedKeyID обновляет ID выбранного ключа
func (m *Model) updateSelectedKeyID() {
	if m.selectedKeyIndex >= 0 && m.selectedKeyIndex < len(m.stats.APIKeys.Keys) {
		m.selectedKeyID = m.stats.APIKeys.Keys[m.selectedKeyIndex].ID
	}
}

// startEditingKey начинает редактирование выбранного ключа
func (m Model) startEditingKey() tea.Cmd {
	if m.selectedKeyIndex >= len(m.stats.APIKeys.Keys) {
		return nil
	}

	key := m.stats.APIKeys.Keys[m.selectedKeyIndex]

	// Заполняем форму текущими данными ключа
	m.editForm = EditKeyForm{
		KeyID:        key.ID,
		Name:         key.Name,
		Description:  key.Description,
		Models:       strings.Join(key.Models, ", "),
		Permissions:  strings.Join(key.Permissions, ", "),
		CurrentField: 0,
	}
	m.editingKey = true
	m.errorMessage = ""

	return nil
}

// handleEditKeyInput обрабатывает ввод в форме редактирования
func (m Model) handleEditKeyInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Сохраняем изменения
		return m, m.saveEditedKey()
	case "esc":
		// Отменяем редактирование
		m.editingKey = false
		m.editForm = EditKeyForm{}
		return m, nil
	case "tab":
		// Переключение между полями
		m.editForm.CurrentField = (m.editForm.CurrentField + 1) % 4
		return m, nil
	case "backspace":
		// Удаление символа
		switch m.editForm.CurrentField {
		case 0:
			if len(m.editForm.Name) > 0 {
				m.editForm.Name = m.editForm.Name[:len(m.editForm.Name)-1]
			}
		case 1:
			if len(m.editForm.Description) > 0 {
				m.editForm.Description = m.editForm.Description[:len(m.editForm.Description)-1]
			}
		case 2:
			if len(m.editForm.Models) > 0 {
				m.editForm.Models = m.editForm.Models[:len(m.editForm.Models)-1]
			}
		case 3:
			if len(m.editForm.Permissions) > 0 {
				m.editForm.Permissions = m.editForm.Permissions[:len(m.editForm.Permissions)-1]
			}
		}
		return m, nil
	default:
		// Добавление символа
		if len(msg.String()) == 1 {
			switch m.editForm.CurrentField {
			case 0:
				m.editForm.Name += msg.String()
			case 1:
				m.editForm.Description += msg.String()
			case 2:
				m.editForm.Models += msg.String()
			case 3:
				m.editForm.Permissions += msg.String()
			}
		}
		return m, nil
	}
}

// saveEditedKey сохраняет изменения ключа через API
func (m Model) saveEditedKey() tea.Cmd {
	return func() tea.Msg {
		// Подготавливаем данные
		models := []string{}
		if m.editForm.Models != "" && m.editForm.Models != "*" {
			for _, model := range strings.Split(m.editForm.Models, ",") {
				models = append(models, strings.TrimSpace(model))
			}
		} else {
			models = []string{"*"}
		}

		permissions := []string{}
		if m.editForm.Permissions != "" {
			for _, perm := range strings.Split(m.editForm.Permissions, ",") {
				permissions = append(permissions, strings.TrimSpace(perm))
			}
		}

		// Отправляем PATCH запрос
		reqBody := map[string]interface{}{
			"name":        m.editForm.Name,
			"description": m.editForm.Description,
			"models":      models,
			"permissions": permissions,
		}

		bodyBytes, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("PUT", m.serverURL+"/api/admin/keys/"+m.editForm.KeyID, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return errMsg(fmt.Errorf("failed to create request: %w", err))
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+m.adminToken)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return errMsg(fmt.Errorf("failed to update key: %w", err))
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return errMsg(fmt.Errorf("failed to update key: %s", string(bodyBytes)))
		}

		// Успешно обновлено
		m.editingKey = false
		m.editForm = EditKeyForm{}
		m.successMessage = "✅ API ключ успешно обновлен!"

		// Обновляем список ключей
		return fetchStatsCmd(m.serverURL)()
	}
}

// executeConfirmedAction выполняет подтвержденное действие (revoke/enable)
func (m Model) executeConfirmedAction() tea.Cmd {
	if m.selectedKeyIndex >= len(m.stats.APIKeys.Keys) {
		return nil
	}

	key := m.stats.APIKeys.Keys[m.selectedKeyIndex]
	action := m.confirmAction

	return func() tea.Msg {
		var endpoint string
		var reqBody map[string]interface{}

		if action == "revoke" {
			endpoint = m.serverURL + "/api/admin/keys/" + key.ID + "/revoke"
			reqBody = map[string]interface{}{
				"reason": "Revoked via TUI",
			}
		} else if action == "enable" {
			endpoint = m.serverURL + "/api/admin/keys/" + key.ID + "/enable"
			reqBody = map[string]interface{}{}
		} else {
			return errMsg(fmt.Errorf("unknown action: %s", action))
		}

		bodyBytes, _ := json.Marshal(reqBody)

		req, err := http.NewRequest("PATCH", endpoint, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return errMsg(fmt.Errorf("failed to create request: %w", err))
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+m.adminToken)

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return errMsg(fmt.Errorf("failed to %s key: %w", action, err))
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return errMsg(fmt.Errorf("failed to %s key: %s", action, string(bodyBytes)))
		}

		// Успешно выполнено
		m.showConfirm = false
		m.confirmAction = ""

		actionText := "отозван"
		if action == "enable" {
			actionText = "активирован"
		}
		m.successMessage = fmt.Sprintf("✅ API ключ успешно %s!", actionText)

		// Обновляем список ключей
		return fetchStatsCmd(m.serverURL)()
	}
}

// Admin token теперь загружается из конфигурации и передается через Model.adminToken

// renderEditKeyForm отрисовывает форму редактирования API ключа (AUTH-04)
func (m Model) renderEditKeyForm() string {
	content := "✏️  РЕДАКТИРОВАНИЕ API КЛЮЧА\n\n"

	// Поле имени
	nameLabel := "Имя ключа:"
	if m.editForm.CurrentField == 0 {
		nameLabel = statusStyle.Render("▶ Имя ключа:")
	}
	content += nameLabel + "\n"
	content += borderStyle.Render(m.editForm.Name+"_") + "\n\n"

	// Поле описания
	descLabel := "Описание:"
	if m.editForm.CurrentField == 1 {
		descLabel = statusStyle.Render("▶ Описание:")
	}
	content += descLabel + "\n"
	desc := m.editForm.Description
	if desc == "" {
		desc = "-"
	}
	if m.editForm.CurrentField == 1 {
		desc += "_"
	}
	content += borderStyle.Render(desc) + "\n\n"

	// Поле моделей
	modelsLabel := "Модели (через запятую, * для всех):"
	if m.editForm.CurrentField == 2 {
		modelsLabel = statusStyle.Render("▶ Модели (через запятую, * для всех):")
	}
	content += modelsLabel + "\n"
	models := m.editForm.Models
	if models == "" {
		models = "*"
	}
	if m.editForm.CurrentField == 2 {
		models += "_"
	}
	content += borderStyle.Render(models) + "\n\n"

	// Поле permissions
	permsLabel := "Permissions (через запятую):"
	if m.editForm.CurrentField == 3 {
		permsLabel = statusStyle.Render("▶ Permissions (через запятую):")
	}
	content += permsLabel + "\n"
	perms := m.editForm.Permissions
	if perms == "" {
		perms = "chat, models"
	}
	if m.editForm.CurrentField == 3 {
		perms += "_"
	}
	content += borderStyle.Render(perms) + "\n\n"

	// Подсказки
	content += helpStyle.Render("Tab - переключение полей | Enter - сохранить | Esc - отменить")

	return borderStyle.Render(content)
}

// renderConfirmDialog отрисовывает модальное окно подтверждения (AUTH-04)
func (m Model) renderConfirmDialog() string {
	if m.selectedKeyIndex >= len(m.stats.APIKeys.Keys) {
		return ""
	}

	key := m.stats.APIKeys.Keys[m.selectedKeyIndex]

	var content strings.Builder
	content.WriteString("\n\n")

	// Рамка вокруг диалога
	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(lipgloss.Color("#FFD700")).
		Padding(1, 2).
		Width(60)

	var message string
	if m.confirmAction == "revoke" {
		message = fmt.Sprintf("⚠️  ПОДТВЕРЖДЕНИЕ ОТЗЫВА КЛЮЧА\n\n"+
			"Вы уверены что хотите ОТОЗВАТЬ ключ:\n\n"+
			"  Имя: %s\n"+
			"  ID:  %s\n\n"+
			"После отзыва ключ станет неактивным и не сможет\n"+
			"использоваться для аутентификации.\n\n"+
			"Вы сможете активировать его позже (клавиша 'x').\n\n"+
			"%s",
			key.Name,
			key.ID,
			helpStyle.Render("Y - Да, отозвать | N/Esc - Отменить"))
	} else {
		message = fmt.Sprintf("✅ ПОДТВЕРЖДЕНИЕ АКТИВАЦИИ КЛЮЧА\n\n"+
			"Вы уверены что хотите АКТИВИРОВАТЬ ключ:\n\n"+
			"  Имя: %s\n"+
			"  ID:  %s\n\n"+
			"После активации ключ снова сможет использоваться\n"+
			"для аутентификации запросов.\n\n"+
			"%s",
			key.Name,
			key.ID,
			helpStyle.Render("Y - Да, активировать | N/Esc - Отменить"))
	}

	content.WriteString(dialogStyle.Render(message))
	content.WriteString("\n\n")

	return content.String()
}

