package agent

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// AgentConfig holds configuration for an agent node.
type AgentConfig struct {
	ListenAddr   string         `mapstructure:"listen_addr"`
	APIKey       string         `mapstructure:"api_key"`
	NodeName     string         `mapstructure:"node_name"`
	NodeType     NodeType       `mapstructure:"node_type"`
	HFCacheDir   string         `mapstructure:"hf_cache_dir"`
	GGUFCacheDir string         `mapstructure:"gguf_cache_dir"`
	HFToken      string         `mapstructure:"hf_token"`
	TLS          AgentTLSConfig `mapstructure:"tls"`

	// Timeouts
	HealthCheckTimeout time.Duration `mapstructure:"health_check_timeout"`
	StartupTimeout     time.Duration `mapstructure:"startup_timeout"`

	// Docker
	DockerBin          string             `mapstructure:"docker_bin"`
	MaxRunningModels   int                `mapstructure:"max_running_models"`
	DefaultProvider    string             `mapstructure:"default_provider"`
	DockerRegistries   []RegistryAuth     `mapstructure:"docker_registries"`
}

// RegistryAuth holds credentials for a private Docker registry.
type RegistryAuth struct {
	Registry string `mapstructure:"registry"` // e.g. "registry.gitlab.com", "harbor.example.com"
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// AgentTLSConfig holds TLS settings for the agent HTTP server.
type AgentTLSConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

// LoadAgentConfig reads agent config from a YAML file.
func LoadAgentConfig(path string) (*AgentConfig, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// Defaults
	v.SetDefault("agent.listen_addr", "0.0.0.0:9090")
	v.SetDefault("agent.node_type", "gpu")
	v.SetDefault("agent.hf_cache_dir", "./data/models/hf")
	v.SetDefault("agent.gguf_cache_dir", "./data/models/gguf")
	v.SetDefault("agent.health_check_timeout", "60s")
	v.SetDefault("agent.startup_timeout", "10m")
	v.SetDefault("agent.docker_bin", "docker")
	v.SetDefault("agent.max_running_models", 2)
	v.SetDefault("agent.default_provider", "vllm")

	// Environment override
	v.SetEnvPrefix("AIGATEWAY_AGENT")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read agent config: %w", err)
	}

	var cfg AgentConfig
	if err := v.UnmarshalKey("agent", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal agent config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate checks required fields.
func (c *AgentConfig) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("agent.api_key is required")
	}
	if c.NodeName == "" {
		hostname, _ := os.Hostname()
		c.NodeName = hostname
	}
	if c.ListenAddr == "" {
		c.ListenAddr = "0.0.0.0:9090"
	}
	if c.NodeType == "" {
		c.NodeType = NodeTypeGPU
	}
	if c.NodeType != NodeTypeGPU && c.NodeType != NodeTypeCPU && c.NodeType != NodeTypeMixed {
		return fmt.Errorf("agent.node_type must be gpu, cpu, or mixed (got %q)", c.NodeType)
	}
	return nil
}
