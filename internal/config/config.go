package config

import (
	"fmt"
	"os"

	datadb "ferryman-agent/internal/data/db"
	"ferryman-agent/internal/llm/provider"
	"ferryman-agent/internal/service/prompt"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(DatabaseConfig, WorkingDir, Prompt)

type MCPType string

const (
	MCPStdio                 MCPType = "stdio"
	MCPSse                   MCPType = "sse"
	DefaultDataDirectory             = ".ferryer"
	MaxTokensFallbackDefault         = 4096
)

var cfg *Config

type MCPServer struct {
	Command string            `json:"command"`
	Env     []string          `json:"env"`
	Args    []string          `json:"args"`
	Type    MCPType           `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type Config struct {
	WorkingDir         string                  `json:"workingDir,omitempty"`
	Database           datadb.DatabaseConfig   `json:"database,omitempty"`
	MCPServers         map[string]MCPServer    `json:"mcpServers,omitempty"`
	Provider           provider.ProviderConfig `json:"provider,omitempty"`
	TitleProvider      provider.ProviderConfig `json:"titleProvider"`
	SummarizerProvider provider.ProviderConfig `json:"summarizerProvider,omitempty"`
	Debug              bool                    `json:"debug,omitempty"`
	AutoCompact        bool                    `json:"autoCompact,omitempty"`
	Prompt             prompt.PromptConfig     `json:"prompt,omitempty"`
	PromptConfigPath   string                  `json:"promptConfigPath,omitempty"`
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

func Prompt(config *Config) prompt.PromptConfig {
	if config == nil {
		return prompt.PromptConfig{}
	}
	if config.Prompt.Type != "" {
		return config.Prompt
	}
	if config.PromptConfigPath != "" {
		return prompt.PromptConfig{
			Type: prompt.PromptConfigPath,
			Path: config.PromptConfigPath,
		}
	}
	return prompt.PromptConfig{}
}

func Current() *Config {
	return Get()
}

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
	switch config.Prompt.Type {
	case "", prompt.PromptConfigPath, prompt.PromptConfigValue:
	default:
		return fmt.Errorf("unsupported prompt config type: %s", config.Prompt.Type)
	}
	return nil
}

func validateOptionalProviderConfig(name string, providerCfg provider.ProviderConfig) error {
	if providerCfg.Provider == "" && providerCfg.ModelConfig.Model == "" && providerCfg.APIKey == "" && providerCfg.BaseURL == "" {
		return nil
	}
	return validateProviderConfig(name, providerCfg)
}

func validateProviderConfig(name string, providerCfg provider.ProviderConfig) error {
	if providerCfg.Provider == "" && providerCfg.ModelConfig.Model == "" {
		return nil
	}
	if providerCfg.Provider == "" {
		return fmt.Errorf("%s provider is required", name)
	}
	if providerCfg.ModelConfig.Model == "" {
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
