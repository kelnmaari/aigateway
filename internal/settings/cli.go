// Package settings - CLI commands for settings management
// Version: v3.0.9 - Phase 5
package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"aigateway/internal/config"
)

// MigrateCommand migrates settings from YAML to database
func MigrateCommand(ctx context.Context, cfg *config.Config, storage Storage, logger *logrus.Logger, categoryFilter string, dryRun bool) error {
	if dryRun {
		logger.Info("🔍 DRY RUN MODE: No changes will be applied")
		fmt.Println("🔍 DRY RUN MODE: Preview only, no changes will be applied")
		fmt.Println()
	} else {
		logger.Info("Starting settings migration from YAML to database...")
	}

	seeder := NewConfigSeeder(storage, logger)

	// Get existing settings
	existing, err := storage.GetAllSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to check existing settings: %w", err)
	}

	// Get expected settings from YAML
	expectedSettings := seeder.MapConfigToSettings(cfg)

	if len(existing) > 0 && !dryRun {
		fmt.Printf("⚠️  Warning: Database already contains %d settings\n", len(existing))
		fmt.Print("Overwrite existing settings? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		if response != "yes" {
			fmt.Println("❌ Migration cancelled")
			return nil
		}
	}

	// Dry-run: Show what would be changed
	if dryRun {
		return previewMigration(existing, expectedSettings)
	}

	// Force seed (will overwrite existing)
	count, err := seeder.SeedFromYAMLForce(ctx, cfg, len(existing) > 0)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	fmt.Printf("✅ Successfully migrated %d settings from YAML to database\n", count)
	return nil
}

// previewMigration shows what would be changed in dry-run mode
func previewMigration(existing []*Setting, expected []Setting) error {
	// Build lookup maps
	existingMap := make(map[string]*Setting)
	for _, s := range existing {
		existingMap[s.ID] = s
	}

	expectedMap := make(map[string]Setting)
	for _, s := range expected {
		expectedMap[s.ID] = s
	}

	var toCreate []string
	var toUpdate []string
	var unchanged []string

	// Check what would be created or updated
	for id, expected := range expectedMap {
		if current, exists := existingMap[id]; exists {
			if current.Value != expected.Value {
				toUpdate = append(toUpdate, fmt.Sprintf("%s: '%s' → '%s'", id, current.Value, expected.Value))
			} else {
				unchanged = append(unchanged, id)
			}
		} else {
			toCreate = append(toCreate, fmt.Sprintf("%s = '%s'", id, expected.Value))
		}
	}

	// Print summary
	fmt.Println("📊 Migration Preview:")
	fmt.Printf("Total settings in YAML: %d\n", len(expected))
	fmt.Printf("Total settings in DB: %d\n", len(existing))
	fmt.Println()

	if len(toCreate) > 0 {
		fmt.Printf("✨ Would CREATE (%d):\n", len(toCreate))
		for _, item := range toCreate {
			fmt.Printf("  + %s\n", item)
		}
		fmt.Println()
	}

	if len(toUpdate) > 0 {
		fmt.Printf("🔄 Would UPDATE (%d):\n", len(toUpdate))
		for _, item := range toUpdate {
			fmt.Printf("  ~ %s\n", item)
		}
		fmt.Println()
	}

	if len(unchanged) > 0 {
		fmt.Printf("✓ Unchanged (%d settings)\n", len(unchanged))
		fmt.Println()
	}

	fmt.Println("💡 Run without --dry-run to apply these changes")
	return nil
}

// ExportCommand exports database settings to file (YAML or JSON)
func ExportCommand(ctx context.Context, storage Storage, logger *logrus.Logger, outputPath string, format string) error {
	// Normalize format
	format = strings.ToLower(format)
	if format != "yaml" && format != "json" {
		return fmt.Errorf("unsupported format '%s': must be 'yaml' or 'json'", format)
	}

	logger.WithField("format", format).Info("Exporting database settings...")

	// Get all settings from database
	allSettings, err := storage.GetAllSettings(ctx)
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	if len(allSettings) == 0 {
		return fmt.Errorf("no settings found in database")
	}

	// Convert to export-friendly structure
	exportData := make(map[string]interface{})
	exportData["exported_at"] = time.Now().Format(time.RFC3339)
	exportData["total_settings"] = len(allSettings)
	exportData["format"] = format

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

	// Write to file based on format
	var fileData []byte
	if format == "json" {
		fileData, err = json.MarshalIndent(exportData, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
	} else {
		fileData, err = yaml.Marshal(exportData)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML: %w", err)
		}
	}

	if err := os.WriteFile(outputPath, fileData, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	fmt.Printf("✅ Successfully exported %d settings to %s (%s format)\n", len(allSettings), outputPath, strings.ToUpper(format))
	return nil
}

// ImportCommand imports settings from exported YAML/JSON file
func ImportCommand(ctx context.Context, storage Storage, logger *logrus.Logger, inputPath string) error {
	logger.WithField("file", inputPath).Info("Importing settings from file...")

	// Read file
	fileData, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Detect format (YAML or JSON)
	var importData map[string]interface{}
	
	// Try JSON first
	if err := json.Unmarshal(fileData, &importData); err != nil {
		// Try YAML
		if err := yaml.Unmarshal(fileData, &importData); err != nil {
			return fmt.Errorf("failed to parse file as JSON or YAML: %w", err)
		}
	}

	// Extract settings from import data
	settingsData, ok := importData["settings"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid import file: missing 'settings' field")
	}

	// Parse settings by category
	var settingsToImport []*Setting
	for categoryName, categoryData := range settingsData {
		categorySettings, ok := categoryData.([]interface{})
		if !ok {
			logger.WithField("category", categoryName).Warn("Invalid category format, skipping")
			continue
		}

		for _, settingData := range categorySettings {
			settingMap, ok := settingData.(map[string]interface{})
			if !ok {
				continue
			}

			setting := &Setting{
				ID:             getString(settingMap, "id"),
				Category:       SettingCategory(categoryName),
				Key:            getString(settingMap, "key"),
				Value:          getString(settingMap, "value"),
				Type:           SettingType(getString(settingMap, "type")),
				DefaultValue:   getString(settingMap, "default_value"),
				Description:    getString(settingMap, "description"),
				IsEditable:     getBool(settingMap, "is_editable"),
				RequiresRestart: getBool(settingMap, "requires_restart"),
				ValidationRule: getString(settingMap, "validation_rule"),
			}

			if setting.ID != "" {
				settingsToImport = append(settingsToImport, setting)
			}
		}
	}

	if len(settingsToImport) == 0 {
		return fmt.Errorf("no valid settings found in import file")
	}

	// Confirm import
	fmt.Printf("⚠️  About to import %d settings from %s\n", len(settingsToImport), inputPath)
	fmt.Print("This will overwrite existing settings. Continue? (yes/no): ")
	var response string
	fmt.Scanln(&response)
	if response != "yes" {
		fmt.Println("❌ Import cancelled")
		return nil
	}

	// Import settings
	imported := 0
	failed := 0
	for _, setting := range settingsToImport {
		if err := storage.UpsertSetting(ctx, setting); err != nil {
			logger.WithError(err).WithField("id", setting.ID).Warn("Failed to import setting")
			failed++
			continue
		}
		imported++
	}

	fmt.Printf("✅ Successfully imported %d settings (%d failed)\n", imported, failed)
	if failed > 0 {
		fmt.Printf("⚠️  %d settings failed to import (check logs for details)\n", failed)
	}

	return nil
}

// Helper functions for parsing import data
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

// ValidateCommand validates database settings against YAML config
func ValidateCommand(ctx context.Context, cfg *config.Config, storage Storage, logger *logrus.Logger, diffMode bool) error {
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
	var valueMismatches []ValueMismatch

	// Check for settings in YAML but not in DB
	for id, expected := range expectedMap {
		if _, exists := dbMap[id]; !exists {
			missingInDB = append(missingInDB, id)
		} else {
			// Check value mismatch
			if dbMap[id].Value != expected.Value {
				valueMismatches = append(valueMismatches, ValueMismatch{
					ID:        id,
					DBValue:   dbMap[id].Value,
					YAMLValue: expected.Value,
				})
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
	if diffMode {
		return printDetailedDiff(dbSettings, expectedSettings, dbMap, expectedMap, missingInDB, missingInYAML, valueMismatches)
	}

	return printSimpleSummary(dbSettings, expectedSettings, missingInDB, missingInYAML, valueMismatches)
}

// ValueMismatch represents a setting with different values
type ValueMismatch struct {
	ID        string
	DBValue   string
	YAMLValue string
}

// printSimpleSummary prints a simple validation summary
func printSimpleSummary(dbSettings []*Setting, expectedSettings []Setting, missingInDB, missingInYAML []string, valueMismatches []ValueMismatch) error {
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
			fmt.Printf("  - %s (DB: %s, YAML: %s)\n", mismatch.ID, mismatch.DBValue, mismatch.YAMLValue)
		}
		fmt.Println()
	}

	if len(missingInDB) == 0 && len(valueMismatches) == 0 {
		fmt.Println("✅ All settings are in sync!")
	}

	return nil
}

// printDetailedDiff prints a detailed side-by-side comparison
func printDetailedDiff(dbSettings []*Setting, expectedSettings []Setting, dbMap map[string]*Setting, expectedMap map[string]Setting, missingInDB, missingInYAML []string, valueMismatches []ValueMismatch) error {
	fmt.Println("\n📊 Detailed Validation Report (Diff Mode)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Total in DB: %d  |  Total in YAML: %d\n", len(dbSettings), len(expectedSettings))
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Print value mismatches with side-by-side comparison
	if len(valueMismatches) > 0 {
		fmt.Printf("🔄 VALUE MISMATCHES (%d):\n\n", len(valueMismatches))
		
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SETTING\tDATABASE\t→\tYAML")
		fmt.Fprintln(w, "---\t---\t---\t---")
		
		for _, mismatch := range valueMismatches {
			dbVal := truncate(mismatch.DBValue, 40)
			yamlVal := truncate(mismatch.YAMLValue, 40)
			fmt.Fprintf(w, "%s\t%s\t→\t%s\n", mismatch.ID, dbVal, yamlVal)
		}
		
		w.Flush()
		fmt.Println()
	}

	// Print missing in DB
	if len(missingInDB) > 0 {
		fmt.Printf("❌ MISSING IN DATABASE (%d):\n", len(missingInDB))
		for _, id := range missingInDB {
			yamlVal := truncate(expectedMap[id].Value, 60)
			fmt.Printf("  + %s = '%s'\n", id, yamlVal)
		}
		fmt.Println()
	}

	// Print missing in YAML (extra in DB)
	if len(missingInYAML) > 0 {
		fmt.Printf("⚠️  NOT IN YAML (extra in DB) (%d):\n", len(missingInYAML))
		for _, id := range missingInYAML {
			dbVal := truncate(dbMap[id].Value, 60)
			fmt.Printf("  - %s = '%s'\n", id, dbVal)
		}
		fmt.Println()
	}

	// Print summary
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if len(missingInDB) == 0 && len(valueMismatches) == 0 && len(missingInYAML) == 0 {
		fmt.Println("✅ RESULT: All settings are in perfect sync!")
	} else {
		fmt.Println("⚠️  RESULT: Settings are out of sync")
		fmt.Println("\n💡 Actions:")
		if len(missingInDB) > 0 || len(valueMismatches) > 0 {
			fmt.Println("   - Run -migrate-config to sync DB with YAML")
		}
		if len(missingInYAML) > 0 {
			fmt.Println("   - Review extra DB settings (may be manually added)")
		}
	}
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	return nil
}

// truncate truncates a string to maxLen characters
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
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

// DeleteCommand deletes settings by ID (for removing deprecated settings)
func DeleteCommand(ctx context.Context, storage Storage, logger *logrus.Logger, settingIDs []string, force bool) error {
	if len(settingIDs) == 0 {
		return fmt.Errorf("no setting IDs provided")
	}
	
	logger.WithField("count", len(settingIDs)).Info("Starting settings deletion...")
	
	// Validate settings exist
	var toDelete []*Setting
	for _, id := range settingIDs {
		setting, err := storage.GetSetting(ctx, id)
		if err != nil {
			fmt.Printf("⚠️  Warning: Setting '%s' not found, skipping\n", id)
			logger.WithField("id", id).Warn("Setting not found")
			continue
		}
		toDelete = append(toDelete, setting)
	}
	
	if len(toDelete) == 0 {
		fmt.Println("❌ No valid settings found to delete")
		return fmt.Errorf("no valid settings found")
	}
	
	// Display settings to be deleted
	fmt.Printf("\n📋 Settings to be deleted (%d):\n\n", len(toDelete))
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tCategory\tKey\tValue\tMigrated")
	fmt.Fprintln(w, "──\t────────\t───\t─────\t────────")
	
	for _, s := range toDelete {
		migrated := "No"
		if s.IsMigrated {
			migrated = "Yes"
		}
		value := truncate(s.Value, 30)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", s.ID, s.Category, s.Key, value, migrated)
	}
	w.Flush()
	fmt.Println()
	
	// Confirmation prompt
	if !force {
		fmt.Print("⚠️  Are you sure you want to DELETE these settings? (yes/no): ")
		var response string
		fmt.Scanln(&response)
		
		if strings.ToLower(strings.TrimSpace(response)) != "yes" {
			fmt.Println("❌ Deletion cancelled")
			return nil
		}
	}
	
	// Delete settings
	deleted := 0
	failed := 0
	
	for _, setting := range toDelete {
		if err := storage.DeleteSetting(ctx, setting.ID); err != nil {
			fmt.Printf("❌ Failed to delete '%s': %v\n", setting.ID, err)
			logger.WithError(err).WithField("id", setting.ID).Error("Failed to delete setting")
			failed++
		} else {
			fmt.Printf("✅ Deleted: %s\n", setting.ID)
			deleted++
		}
	}
	
	fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("📊 Deletion Summary:\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	fmt.Printf("✅ Deleted:  %d\n", deleted)
	if failed > 0 {
		fmt.Printf("❌ Failed:   %d\n", failed)
	}
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	
	logger.WithFields(logrus.Fields{
		"deleted": deleted,
		"failed":  failed,
	}).Info("Settings deletion completed")
	
	if failed > 0 {
		return fmt.Errorf("failed to delete %d settings", failed)
	}
	
	return nil
}

