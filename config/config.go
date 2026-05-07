package config

import (
	"fmt"
	"os"

	datadb "ferryman-agent/data/db"
	"ferryman-agent/llm/models"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(DatabaseConfig, WorkingDir)

type MCPType string

const (
	MCPStdio                 MCPType = "stdio"
	MCPSse                   MCPType = "sse"
	DefaultDataDirectory             = ".ferryer"
	MaxTokensFallbackDefault         = 4096
)

type MCPServer struct {
	Command string            `json:"command"`
	Env     []string          `json:"env"`
	Args    []string          `json:"args"`
	Type    MCPType           `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type ProviderConfig struct {
	Provider    models.ModelProvider `json:"provider"`
	APIKey      string               `json:"apiKey"`
	BaseURL     string               `json:"baseURL"`
	ModelConfig ModelConfig          `json:"modelConfig"`
	Prompt      string               `json:"prompt"`
	Disabled    bool                 `json:"disabled"`
}

type ModelConfig struct {
	Model           models.ModelID `json:"model"`
	MaxTokens       int64          `json:"maxTokens,omitempty"`
	ReasoningEffort string         `json:"reasoningEffort,omitempty"`
}
type Config struct {
	WorkingDir         string                `json:"workingDir,omitempty"`
	Database           datadb.DatabaseConfig `json:"database,omitempty"`
	MCPServers         map[string]MCPServer  `json:"mcpServers,omitempty"`
	Provider           ProviderConfig        `json:"provider,omitempty"`
	TitleProvider      ProviderConfig        `json:"titleProvider"`
	SummarizerProvider ProviderConfig        `json:"summarizerProvider,omitempty"`
	Debug              bool                  `json:"debug,omitempty"`
	AutoCompact        bool                  `json:"autoCompact,omitempty"`
	PromptConfigPath   string                `json:"promptConfigPath,omitempty"`
}

func DatabaseConfig(config *Config) datadb.DatabaseConfig {
	if config == nil {
		return datadb.DatabaseConfig{}
	}
	return config.Database
}

func WorkingDir(config *Config) string {
	if config == nil {
		return ""
	}
	return config.WorkingDir
}

func Current() *Config {
	return Get()
}

var cfg *Config

func Use(config Config) (*Config, error) {
	config = WithDefaults(config)
	if err := Validate(config); err != nil {
		return nil, err
	}
	cfg = &config
	return cfg, nil
}

func WithDefaults(config Config) Config {
	if config.WorkingDir == "" {
		if wd, err := os.Getwd(); err == nil {
			config.WorkingDir = wd
		}
	}
	if config.Database.Type == "" {
		config.Database.Type = datadb.DatabaseSQLite
	}
	if config.MCPServers == nil {
		config.MCPServers = make(map[string]MCPServer)
	}
	for name, server := range config.MCPServers {
		if server.Type == "" {
			server.Type = MCPStdio
			config.MCPServers[name] = server
		}
	}
	return config
}

func Validate(config Config) error {
	if err := validateProviderConfig("provider", config.Provider); err != nil {
		return err
	}
	if err := validateOptionalProviderConfig("titleProvider", config.TitleProvider); err != nil {
		return err
	}
	if err := validateOptionalProviderConfig("summarizerProvider", config.SummarizerProvider); err != nil {
		return err
	}
	switch config.Database.Type {
	case "", datadb.DatabaseSQLite, datadb.DatabaseMySQL:
	default:
		return fmt.Errorf("unsupported database type: %s", config.Database.Type)
	}
	return nil
}

func validateOptionalProviderConfig(name string, provider ProviderConfig) error {
	if provider.Provider == "" && provider.ModelConfig.Model == "" && provider.APIKey == "" && provider.BaseURL == "" && provider.Prompt == "" {
		return nil
	}
	return validateProviderConfig(name, provider)
}

func validateProviderConfig(name string, provider ProviderConfig) error {
	if provider.Provider == "" && provider.ModelConfig.Model == "" {
		return nil
	}
	if provider.Provider == "" {
		return fmt.Errorf("%s provider is required", name)
	}
	if provider.ModelConfig.Model == "" {
		return fmt.Errorf("%s model is required", name)
	}
	return nil
}

func Get() *Config {
	if cfg == nil {
		defaultCfg, _ := Use(Config{})
		return defaultCfg
	}
	return cfg
}

func WorkingDirectory() string {
	return Get().WorkingDir
}
