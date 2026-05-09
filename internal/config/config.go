package config

import (
	mcptools "ferryman-agent/internal/capability/mcp"
	datadb "ferryman-agent/internal/data/db"
	"ferryman-agent/internal/data/llm/models"
	"fmt"
	"os"

	"ferryman-agent/internal/prompt"
	"ferryman-agent/internal/provider"
)

const (
	DefaultDataDirectory     = ".ferryer"
	MaxTokensFallbackDefault = 200000
	DatabaseSQLite           = datadb.DatabaseSQLite
	DatabaseMySQL            = datadb.DatabaseMySQL
)

var cfg *Config

type DatabaseConfig = datadb.DatabaseConfig
type DatabaseType = datadb.DatabaseType
type MCPServer = mcptools.MCPServer
type MCPType = mcptools.MCPType

const (
	MCPStdio = mcptools.MCPStdio
	MCPSse   = mcptools.MCPSse
)

type AgentModelConfig struct {
	ModelID  models.ModelID       `json:"model_id,omitempty"`
	Provider models.ModelProvider `json:"provider,omitempty"`
}

type Config struct {
	WorkingDir       string                    `json:"workingDir,omitempty"`
	Database         DatabaseConfig            `json:"database,omitempty"`
	MCPServers       map[string]MCPServer      `json:"mcpServers,omitempty"`
	Providers        []provider.ProviderConfig `json:"providers,omitempty"`
	Agent            AgentModelConfig          `json:"agent,omitempty"`
	TitleAgent       AgentModelConfig          `json:"titleAgent,omitempty"`
	SummarizeAgent   AgentModelConfig          `json:"summarizeAgent,omitempty"`
	Debug            bool                      `json:"debug,omitempty"`
	AutoCompact      bool                      `json:"autoCompact,omitempty"`
	Prompt           prompt.PromptConfig       `json:"prompt,omitempty"`
	PromptConfigPath string                    `json:"promptConfigPath,omitempty"`
}

func ProvideDatabaseConfig(config *Config) DatabaseConfig {
	if config == nil {
		return DatabaseConfig{}
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
		config.Database.Type = DatabaseSQLite
	}
	if config.MCPServers == nil {
		config.MCPServers = make(map[string]MCPServer)
	}
	config.Agent = defaultAgentModelConfig(config.Agent, primaryProviderConfig(config))
	config.TitleAgent = defaultAgentModelConfig(config.TitleAgent, primaryProviderConfig(config))
	config.SummarizeAgent = defaultAgentModelConfig(config.SummarizeAgent, primaryProviderConfig(config))
	for name, server := range config.MCPServers {
		if server.Type == "" {
			server.Type = mcptools.MCPStdio
			config.MCPServers[name] = server
		}
	}
	return config
}

func ProviderConfigs(config Config) []provider.ProviderConfig {
	return config.Providers
}

func primaryProviderConfig(config Config) provider.ProviderConfig {
	if len(config.Providers) > 0 {
		return config.Providers[0]
	}
	return provider.ProviderConfig{}
}

func defaultAgentModelConfig(agentCfg AgentModelConfig, providerCfg provider.ProviderConfig) AgentModelConfig {
	if agentCfg.ModelID == "" {
		if modelCfg, ok := providerCfg.PrimaryModelConfig(); ok {
			agentCfg.ModelID = modelCfg.ModelID
		}
	}
	if agentCfg.Provider == "" {
		agentCfg.Provider = providerCfg.Provider
	}
	return agentCfg
}

func Validate(config Config) error {
	for i, providerCfg := range config.Providers {
		if err := validateProviderConfig(fmt.Sprintf("providers[%d]", i), providerCfg); err != nil {
			return err
		}
	}
	switch config.Database.Type {
	case "", DatabaseSQLite, DatabaseMySQL:
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

func validateProviderConfig(name string, providerCfg provider.ProviderConfig) error {
	if !providerCfg.Configured() {
		return nil
	}
	if providerCfg.Provider == "" {
		return fmt.Errorf("%s provider is required", name)
	}
	if len(providerCfg.Models) == 0 {
		return fmt.Errorf("%s models is required", name)
	}
	for i, modelCfg := range providerCfg.Models {
		if modelCfg.ModelID == "" {
			return fmt.Errorf("%s models[%d].model_id is required", name, i)
		}
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
