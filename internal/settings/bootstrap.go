// Package settings - Bootstrap config generator
// Version: v3.1.0 - Minimal bootstrap config generation
package settings

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"aigateway/internal/config"
)

// BootstrapConfig represents minimal config required to connect to database
type BootstrapConfig struct {
	Database DatabaseBootstrap `yaml:"database"`
	Logging  LoggingBootstrap  `yaml:"logging,omitempty"` // Optional: for startup logs
}

// DatabaseBootstrap minimal DB connection info
type DatabaseBootstrap struct {
	Type       string              `yaml:"type"`
	PostgreSQL PostgreSQLBootstrap `yaml:"postgresql,omitempty"`
	SQLite     SQLiteBootstrap     `yaml:"sqlite,omitempty"`
}

// PostgreSQLBootstrap minimal PostgreSQL connection
type PostgreSQLBootstrap struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	SSLMode  string `yaml:"ssl_mode,omitempty"`
}

// SQLiteBootstrap minimal SQLite connection
type SQLiteBootstrap struct {
	Path string `yaml:"path"`
}

// LoggingBootstrap minimal logging config
type LoggingBootstrap struct {
	Level  string `yaml:"level,omitempty"`
	Format string `yaml:"format,omitempty"`
}

// GenerateBootstrapCommand generates minimal bootstrap.yaml from full config
func GenerateBootstrapCommand(ctx context.Context, cfg *config.Config, logger *logrus.Logger, outputPath string) error {
	logger.Info("Generating minimal bootstrap configuration...")

	// Create bootstrap config with only database connection info
	bootstrap := BootstrapConfig{
		Database: DatabaseBootstrap{
			Type: string(cfg.Database.Type),
		},
		Logging: LoggingBootstrap{
			Level:  cfg.Logging.Level,
			Format: cfg.Logging.Format,
		},
	}

	// Set database-specific fields
	switch cfg.Database.Type {
	case config.DatabaseTypePostgreSQL:
		bootstrap.Database.PostgreSQL = PostgreSQLBootstrap{
			Host:     cfg.Database.PostgreSQL.Host,
			Port:     cfg.Database.PostgreSQL.Port,
			Database: cfg.Database.PostgreSQL.Database,
			User:     cfg.Database.PostgreSQL.User,
			Password: "${POSTGRES_PASSWORD}", // Use env var placeholder
			SSLMode:  cfg.Database.PostgreSQL.SSLMode,
		}
	case config.DatabaseTypeSQLite:
		bootstrap.Database.SQLite = SQLiteBootstrap{
			Path: cfg.Database.SQLite.Path,
		}
	default:
		return fmt.Errorf("unsupported database type: %s", cfg.Database.Type)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(bootstrap)
	if err != nil {
		return fmt.Errorf("failed to marshal bootstrap config: %w", err)
	}

	// Add header comment
	header := `# AIGateway Bootstrap Configuration
# Version: v3.1.0
# 
# This is a MINIMAL config file containing only database connection info.
# All other settings are loaded from the database on startup.
# 
# After initial migration, this replaces your full dev.yaml/production.yaml.
# 
# Environment Variables:
#   POSTGRES_PASSWORD - Database password (for security)
#
# To migrate settings from full config to database:
#   ./bin/server.exe -migrate-config
#
# Full configuration is then managed via:
#   - Web UI: http://localhost:8085/admin.html → Settings tab
#   - CLI: ./bin/server.exe -export-config / -import-config
#

`

	fullYAML := header + string(data)

	// Write to file
	if err := os.WriteFile(outputPath, []byte(fullYAML), 0644); err != nil {
		return fmt.Errorf("failed to write bootstrap config: %w", err)
	}

	logger.WithField("output", outputPath).Info("Bootstrap config generated successfully")

	fmt.Println()
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("✅ Bootstrap config generated successfully!")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("\n📁 Output: %s\n", outputPath)
	fmt.Println("\n📋 What's included:")
	fmt.Println("   • Database connection (host, port, db, user)")
	fmt.Println("   • Logging level (for startup)")
	fmt.Println("   • Environment variable placeholders")
	fmt.Println("\n🚫 What's NOT included (loaded from DB):")
	fmt.Println("   • Server settings (port, timeouts, TLS)")
	fmt.Println("   • Auth settings (JWT, OIDC, LDAP)")
	fmt.Println("   • Inference settings (Docker, models)")
	fmt.Println("   • Metrics, RAG, all other configs")
	fmt.Println("\n🔄 Next steps:")
	fmt.Println("   1. Review generated bootstrap.yaml")
	fmt.Println("   2. Set POSTGRES_PASSWORD environment variable")
	fmt.Println("   3. Rename your old dev.yaml → dev.yaml.backup")
	fmt.Println("   4. Use bootstrap.yaml: ./bin/server.exe -config=bootstrap.yaml")
	fmt.Println("   5. All settings loaded from database automatically!")
	fmt.Println()

	return nil
}

