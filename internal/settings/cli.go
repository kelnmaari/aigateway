// Package settings - CLI commands for settings management
// Version: v3.0.9 - Phase 5
package settings

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"aigateway/internal/config"
)

// MigrateCommand migrates settings from YAML to database
func MigrateCommand(ctx context.Context, cfg *config.Config, storage Storage, logger *logrus.Logger, categoryFilter string) error {
	logger.Info("Starting settings migration from YAML to database...")

	seeder := NewConfigSeeder(storage, logger)

	// Get existing settings
	existing, err := storage.GetAllSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to check existing settings: %w", err)
	}

	if len(existing) > 0 {
		fmt.Printf("⚠️  Warning: Database already contains %d settings\n", len(existing))
		fmt.Print("Overwrite existing settings? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			fmt.Println("❌ Migration cancelled")
			return nil
		}
	}

	// Force seed (will overwrite existing)
	count, err := seeder.SeedFromYAMLForce(ctx, cfg, len(existing) > 0)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Printf("✅ Successfully migrated %d settings from YAML to database\n", count)
	return nil
}

// ExportCommand exports database settings to YAML file
func ExportCommand(ctx context.Context, storage Storage, logger *logrus.Logger, outputPath string) error {
	logger.Info("Exporting database settings to YAML...")

	// Get all settings from database
	allSettings, err := storage.GetAllSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	if len(allSettings) == 0 {
		return fmt.Errorf("no settings found in database")
	}

	// Convert to YAML-friendly structure
	exportData := make(map[string]interface{})
	exportData["exported_at"] = time.Now().Format(time.RFC3339)
	exportData["total_settings"] = len(allSettings)

	settingsByCategory := make(map[string][]map[string]interface{})
	for _, setting := range allSettings {
		category := string(setting.Category)
		if settingsByCategory[category] == nil {
			settingsByCategory[category] = []map[string]interface{}{}
		}

		settingMap := map[string]interface{}{
			"id":               setting.ID,
			"key":              setting.Key,
			"value":            setting.Value,
			"type":             string(setting.Type),
			"default_value":    setting.DefaultValue,
			"description":      setting.Description,
			"is_editable":      setting.IsEditable,
			"requires_restart": setting.RequiresRestart,
			"validation_rule":  setting.ValidationRule,
		}
		settingsByCategory[category] = append(settingsByCategory[category], settingMap)
	}

	exportData["settings"] = settingsByCategory

	// Write to YAML file
	yamlData, err := yaml.Marshal(exportData)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	if err := os.WriteFile(outputPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("✅ Successfully exported %d settings to %s\n", len(allSettings), outputPath)
	return nil
}

// ValidateCommand validates database settings against YAML config
func ValidateCommand(ctx context.Context, cfg *config.Config, storage Storage, logger *logrus.Logger) error {
	logger.Info("Validating database settings against YAML config...")

	// Get all settings from database
	dbSettings, err := storage.GetAllSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database settings: %w", err)
	}

	if len(dbSettings) == 0 {
		fmt.Println("⚠️  No settings found in database")
		return nil
	}

	// Create seeder to get expected settings from YAML
	seeder := NewConfigSeeder(storage, logger)
	expectedSettings := seeder.MapConfigToSettings(cfg)

	// Build lookup maps
	dbMap := make(map[string]*Setting)
	for _, s := range dbSettings {
		dbMap[s.ID] = s
	}

	expectedMap := make(map[string]Setting)
	for _, s := range expectedSettings {
		expectedMap[s.ID] = s
	}

	// Validate
	var missingInDB []string
	var missingInYAML []string
	var valueMismatches []string

	// Check for settings in YAML but not in DB
	for id, expected := range expectedMap {
		if _, exists := dbMap[id]; !exists {
			missingInDB = append(missingInDB, id)
		} else {
			// Check value mismatch
			if dbMap[id].Value != expected.Value {
				valueMismatches = append(valueMismatches, fmt.Sprintf("%s (DB: %s, YAML: %s)", id, dbMap[id].Value, expected.Value))
			}
		}
	}

	// Check for settings in DB but not in YAML
	for id := range dbMap {
		if _, exists := expectedMap[id]; !exists {
			missingInYAML = append(missingInYAML, id)
		}
	}

	// Print results
	fmt.Println("\n📊 Validation Report:")
	fmt.Printf("Total in DB: %d\n", len(dbSettings))
	fmt.Printf("Total in YAML: %d\n", len(expectedSettings))
	fmt.Println()

	if len(missingInDB) > 0 {
		fmt.Printf("❌ Missing in DB (%d):\n", len(missingInDB))
		for _, id := range missingInDB {
			fmt.Printf("  - %s\n", id)
		}
		fmt.Println()
	}

	if len(missingInYAML) > 0 {
		fmt.Printf("⚠️  Not in YAML config (%d):\n", len(missingInYAML))
		for _, id := range missingInYAML {
			fmt.Printf("  - %s\n", id)
		}
		fmt.Println()
	}

	if len(valueMismatches) > 0 {
		fmt.Printf("⚠️  Value mismatches (%d):\n", len(valueMismatches))
		for _, mismatch := range valueMismatches {
			fmt.Printf("  - %s\n", mismatch)
		}
		fmt.Println()
	}

	if len(missingInDB) == 0 && len(valueMismatches) == 0 {
		fmt.Println("✅ All settings are in sync!")
	}

	return nil
}

// ListCommand lists all settings in a table format
func ListCommand(ctx context.Context, storage Storage, categoryFilter string) error {
	allSettings, err := storage.GetAllSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	if len(allSettings) == 0 {
		fmt.Println("No settings found in database")
		return nil
	}

	// Filter by category if specified
	var filtered []*Setting
	if categoryFilter != "" {
		for _, s := range allSettings {
			if string(s.Category) == categoryFilter {
				filtered = append(filtered, s)
			}
		}
	} else {
		filtered = allSettings
	}

	if len(filtered) == 0 {
		fmt.Printf("No settings found in category: %s\n", categoryFilter)
		return nil
	}

	// Print table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tCATEGORY\tVALUE\tEDITABLE\tRESTART")
	fmt.Fprintln(w, "---\t---\t---\t---\t---")

	for _, s := range filtered {
		editable := "❌"
		if s.IsEditable {
			editable = "✅"
		}

		restart := "❌"
		if s.RequiresRestart {
			restart = "⚠️"
		}

		value := s.Value
		if len(value) > 40 {
			value = value[:37] + "..."
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", s.ID, s.Category, value, editable, restart)
	}

	w.Flush()
	fmt.Printf("\nTotal: %d settings\n", len(filtered))

	return nil
}

