// Package settings provides database-backed configuration management
// Version: v3.0.9 - Configuration migration from YAML to PostgreSQL
package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// SettingCategory represents configuration category
type SettingCategory string

const (
	CategoryServer        SettingCategory = "server"
	CategoryInference     SettingCategory = "inference"
	CategoryAuth          SettingCategory = "auth"
	CategoryDatabase      SettingCategory = "database"
	CategoryLogging       SettingCategory = "logging"
	CategoryMetrics       SettingCategory = "metrics"
	CategoryTLS           SettingCategory = "tls"
	CategoryRAG           SettingCategory = "rag"
	CategoryHuggingFace   SettingCategory = "huggingface"
	CategoryObservability SettingCategory = "observability" // v3.0.9+: Tracing & Performance
	CategoryModelRegistry SettingCategory = "model_registry" // v3.0.9+: Model Registry
	CategoryDevelopment   SettingCategory = "development"    // v3.0.9+: Development settings
)

// SettingType represents the data type of a setting
type SettingType string

const (
	TypeString   SettingType = "string"
	TypeInt      SettingType = "int"
	TypeBool     SettingType = "bool"
	TypeFloat    SettingType = "float"
	TypeJSON     SettingType = "json"     // For complex objects
	TypeArray    SettingType = "array"    // For string arrays
	TypeDuration SettingType = "duration" // For time.Duration (stored as string)
)

// Setting represents a single configuration setting
type Setting struct {
	ID             string          `json:"id" db:"id"`                                     // Unique key (e.g., "server.port")
	Category       SettingCategory `json:"category" db:"category"`                         // Configuration category
	Key            string          `json:"key" db:"key"`                                   // Setting key within category
	Value          string          `json:"value" db:"value"`                               // Current value (as string)
	Type           SettingType     `json:"type" db:"type"`                                 // Data type
	DefaultValue   string          `json:"default_value" db:"default_value"`               // Default from YAML
	Description    string          `json:"description" db:"description"`                   // Human-readable description
	IsEditable     bool            `json:"is_editable" db:"is_editable"`                   // Can be changed via UI
	IsRequired     bool            `json:"is_required" db:"is_required"`                   // Required for system operation
	IsMigrated     bool            `json:"is_migrated" db:"is_migrated"`                   // Migrated from YAML to DB
	RequiresRestart bool           `json:"requires_restart" db:"requires_restart"`         // Requires server restart (Phase 4)
	ValidationRule string          `json:"validation_rule,omitempty" db:"validation_rule"` // Validation pattern/rule
	UpdatedAt      time.Time       `json:"updated_at" db:"updated_at"`
	UpdatedBy      string          `json:"updated_by,omitempty" db:"updated_by"` // User who updated
}

// SettingValue represents a parsed setting value with type safety
type SettingValue struct {
	StringValue   *string         `json:"string_value,omitempty"`
	IntValue      *int            `json:"int_value,omitempty"`
	BoolValue     *bool           `json:"bool_value,omitempty"`
	FloatValue    *float64        `json:"float_value,omitempty"`
	JSONValue     json.RawMessage `json:"json_value,omitempty"`
	ArrayValue    []string        `json:"array_value,omitempty"`
	DurationValue *time.Duration  `json:"duration_value,omitempty"`
}

// Manager manages configuration settings in database
type Manager struct {
	storage       Storage
	cache         map[string]*Setting // In-memory cache
	mu            sync.RWMutex
	logger        *logrus.Logger
	reloadHandlers map[string]ReloadHandler // Live reload callbacks by setting ID
}

// ReloadHandler is a callback function invoked when a setting changes
type ReloadHandler func(ctx context.Context, setting *Setting) error

// Storage interface for settings persistence
type Storage interface {
	// GetSetting retrieves a setting by ID
	GetSetting(ctx context.Context, id string) (*Setting, error)

	// GetSettingsByCategory retrieves all settings in a category
	GetSettingsByCategory(ctx context.Context, category SettingCategory) ([]*Setting, error)

	// GetAllSettings retrieves all settings
	GetAllSettings(ctx context.Context) ([]*Setting, error)

	// UpsertSetting creates or updates a setting
	UpsertSetting(ctx context.Context, setting *Setting) error

	// UpdateSettingValue updates only the value field
	UpdateSettingValue(ctx context.Context, id, value, updatedBy string) error

	// DeleteSetting deletes a setting (admin only)
	DeleteSetting(ctx context.Context, id string) error

	// BulkUpsertSettings upserts multiple settings in a transaction
	BulkUpsertSettings(ctx context.Context, settings []*Setting) error
}

// NewManager creates a new settings manager
func NewManager(storage Storage, logger *logrus.Logger) *Manager {
	if logger == nil {
		logger = logrus.New()
	}

	return &Manager{
		storage:        storage,
		cache:          make(map[string]*Setting),
		logger:         logger,
		reloadHandlers: make(map[string]ReloadHandler),
	}
}

// RegisterReloadHandler registers a callback for live reload when a setting changes
func (m *Manager) RegisterReloadHandler(settingID string, handler ReloadHandler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.reloadHandlers[settingID] = handler
	m.logger.WithField("setting_id", settingID).Debug("Registered reload handler")
}

// UnregisterReloadHandler removes a reload callback
func (m *Manager) UnregisterReloadHandler(settingID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.reloadHandlers, settingID)
	m.logger.WithField("setting_id", settingID).Debug("Unregistered reload handler")
}

// GetString retrieves a string setting value
func (m *Manager) GetString(ctx context.Context, id string) (string, error) {
	setting, err := m.GetSetting(ctx, id)
	if err != nil {
		return "", err
	}
	return setting.Value, nil
}

// GetInt retrieves an int setting value
func (m *Manager) GetInt(ctx context.Context, id string) (int, error) {
	setting, err := m.GetSetting(ctx, id)
	if err != nil {
		return 0, err
	}

	var value int
	if _, err := fmt.Sscanf(setting.Value, "%d", &value); err != nil {
		return 0, fmt.Errorf("invalid int value for %s: %w", id, err)
	}
	return value, nil
}

// GetBool retrieves a bool setting value
func (m *Manager) GetBool(ctx context.Context, id string) (bool, error) {
	setting, err := m.GetSetting(ctx, id)
	if err != nil {
		return false, err
	}

	var value bool
	if _, err := fmt.Sscanf(setting.Value, "%t", &value); err != nil {
		return false, fmt.Errorf("invalid bool value for %s: %w", id, err)
	}
	return value, nil
}

// GetDuration retrieves a duration setting value
func (m *Manager) GetDuration(ctx context.Context, id string) (time.Duration, error) {
	setting, err := m.GetSetting(ctx, id)
	if err != nil {
		return 0, err
	}

	duration, err := time.ParseDuration(setting.Value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration value for %s: %w", id, err)
	}
	return duration, nil
}

// GetJSON retrieves a JSON setting value
func (m *Manager) GetJSON(ctx context.Context, id string, dest interface{}) error {
	setting, err := m.GetSetting(ctx, id)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(setting.Value), dest); err != nil {
		return fmt.Errorf("invalid JSON value for %s: %w", id, err)
	}
	return nil
}

// SetString sets a string setting value
func (m *Manager) SetString(ctx context.Context, id, value, updatedBy string) error {
	return m.updateValue(ctx, id, value, updatedBy)
}

// SetInt sets an int setting value
func (m *Manager) SetInt(ctx context.Context, id string, value int, updatedBy string) error {
	return m.updateValue(ctx, id, fmt.Sprintf("%d", value), updatedBy)
}

// SetBool sets a bool setting value
func (m *Manager) SetBool(ctx context.Context, id string, value bool, updatedBy string) error {
	return m.updateValue(ctx, id, fmt.Sprintf("%t", value), updatedBy)
}

// updateValue is internal helper to update setting value through storage
func (m *Manager) updateValue(ctx context.Context, id, value, updatedBy string) error {
	// Validate setting exists and is editable
	setting, err := m.GetSetting(ctx, id)
	if err != nil {
		return fmt.Errorf("setting not found: %w", err)
	}
	
	if !setting.IsEditable {
		return fmt.Errorf("setting '%s' is not editable (requires server restart)", id)
	}
	
	// Update through storage
	if err := m.storage.UpdateSettingValue(ctx, id, value, updatedBy); err != nil {
		return fmt.Errorf("failed to update setting: %w", err)
	}
	
	// Invalidate cache for this setting
	m.mu.Lock()
	delete(m.cache, id)
	m.mu.Unlock()
	
	// Phase 4: Trigger live reload hooks for hot-reloadable settings
	if !setting.RequiresRestart {
		// Create updated setting object with new value
		updatedSetting := *setting
		updatedSetting.Value = value
		updatedSetting.UpdatedAt = time.Now()
		updatedSetting.UpdatedBy = updatedBy
		
		// Call registered reload handler
		m.mu.RLock()
		handler, exists := m.reloadHandlers[id]
		m.mu.RUnlock()
		
		if exists {
			if err := handler(ctx, &updatedSetting); err != nil {
				m.logger.WithError(err).
					WithField("setting_id", id).
					Warn("Reload handler failed, but update was saved")
				// Don't fail the update if reload handler fails
			} else {
				m.logger.WithField("setting_id", id).Info("Live reload applied successfully")
			}
		}
	}
	
	return nil
}

// GetAllSettings retrieves all settings grouped by category
func (m *Manager) GetAllSettings(ctx context.Context) (map[SettingCategory][]*Setting, error) {
	settings, err := m.storage.GetAllSettings(ctx)
	if err != nil {
		return nil, err
	}

	grouped := make(map[SettingCategory][]*Setting)
	for _, setting := range settings {
		grouped[setting.Category] = append(grouped[setting.Category], setting)
	}

	return grouped, nil
}

// getSetting retrieves from cache or storage
// GetSetting retrieves a single setting by ID (with caching)
func (m *Manager) GetSetting(ctx context.Context, id string) (*Setting, error) {
	// Check cache first
	m.mu.RLock()
	if cached, ok := m.cache[id]; ok {
		m.mu.RUnlock()
		return cached, nil
	}
	m.mu.RUnlock()

	// Load from storage
	setting, err := m.storage.GetSetting(ctx, id)
	if err != nil {
		return nil, err
	}

	// Update cache
	m.mu.Lock()
	m.cache[id] = setting
	m.mu.Unlock()

	return setting, nil
}

// DeleteSetting deletes a setting from storage and cache
func (m *Manager) DeleteSetting(ctx context.Context, id string) error {
	// Delete from storage
	if err := m.storage.DeleteSetting(ctx, id); err != nil {
		return fmt.Errorf("failed to delete setting: %w", err)
	}
	
	// Remove from cache
	m.mu.Lock()
	delete(m.cache, id)
	m.mu.Unlock()
	
	// Unregister reload handler if exists
	m.UnregisterReloadHandler(id)
	
	m.logger.WithField("setting_id", id).Info("Setting deleted successfully")
	return nil
}

// InvalidateCache clears the settings cache
func (m *Manager) InvalidateCache() {
	m.mu.Lock()
	m.cache = make(map[string]*Setting)
	m.mu.Unlock()

	m.logger.Info("Settings cache invalidated")
}
