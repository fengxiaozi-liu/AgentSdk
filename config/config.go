package config

import (
	"fmt"
	"os"
	"path/filepath"

	datadb "ferryman-agent/data/db"
	"ferryman-agent/llm/models"
)

type MCPType string

const (
	MCPStdio MCPType = "stdio"
	MCPSse   MCPType = "sse"

	DefaultDataDirectory = ".ferryer"

	MaxTokensFallbackDefault = 4096
)

type MCPServer struct {
	Command string            `json:"command"`
	Env     []string          `json:"env"`
	Args    []string          `json:"args"`
	Type    MCPType           `json:"type"`
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers"`
}

type Provider struct {
	APIKey   string `json:"apiKey"`
	Disabled bool   `json:"disabled"`
}

type Data struct {
	Directory string `json:"directory,omitempty"`
}

type ModelConfig struct {
	Provider        models.ModelProvider `json:"provider"`
	Model           models.ModelID       `json:"model"`
	PromptKey       string               `json:"promptKey,omitempty"`
	MaxTokens       int64                `json:"maxTokens,omitempty"`
	ReasoningEffort string               `json:"reasoningEffort,omitempty"`
}

type Config struct {
	Data             Data                              `json:"data"`
	Database         datadb.DatabaseConfig             `json:"database,omitempty"`
	WorkingDir       string                            `json:"wd,omitempty"`
	MCPServers       map[string]MCPServer              `json:"mcpServers,omitempty"`
	Providers        map[models.ModelProvider]Provider `json:"providers,omitempty"`
	Model            ModelConfig                       `json:"model,omitempty"`
	ModelProfiles    map[string]ModelConfig            `json:"modelProfiles,omitempty"`
	Debug            bool                              `json:"debug,omitempty"`
	AutoCompact      bool                              `json:"autoCompact,omitempty"`
	PromptConfigPath string                            `json:"promptConfigPath,omitempty"`
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
	if config.Data.Directory == "" {
		config.Data.Directory = DefaultDataDirectory
	}
	if config.Database.Type == "" {
		config.Database.Type = datadb.DatabaseSQLite
	}
	if config.Database.Type == datadb.DatabaseSQLite && config.Database.Path == "" && config.Database.DSN == "" {
		config.Database.Path = filepath.Join(config.Data.Directory, "agent.db")
	}
	if config.Providers == nil {
		config.Providers = make(map[models.ModelProvider]Provider)
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
	if config.ModelProfiles == nil {
		config.ModelProfiles = make(map[string]ModelConfig)
	}
	return config
}

func Validate(config Config) error {
	for name, profile := range config.ModelProfiles {
		if profile.Provider == "" {
			return fmt.Errorf("model profile %s provider is required", name)
		}
		if profile.Model == "" {
			return fmt.Errorf("model profile %s model is required", name)
		}
	}
	if config.Model.Model != "" && config.Model.Provider == "" {
		return fmt.Errorf("model provider is required")
	}
	if config.Model.Provider != "" && config.Model.Model == "" {
		return fmt.Errorf("model id is required")
	}
	switch config.Database.Type {
	case "", datadb.DatabaseSQLite, datadb.DatabaseMySQL:
	default:
		return fmt.Errorf("unsupported database type: %s", config.Database.Type)
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

func ModelProfile(key string) (ModelConfig, bool) {
	cfg := Get()
	if key != "" {
		if profile, ok := cfg.ModelProfiles[key]; ok {
			if profile.PromptKey == "" {
				profile.PromptKey = key
			}
			return profile, true
		}
	}
	if cfg.Model.Model == "" || cfg.Model.Provider == "" {
		return ModelConfig{}, false
	}
	profile := cfg.Model
	if profile.PromptKey == "" {
		profile.PromptKey = key
	}
	return profile, true
}
