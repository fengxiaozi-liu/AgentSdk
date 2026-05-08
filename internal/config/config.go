package config

import (
	"encoding/json"
	"ferryman-agent/internal/data/llm/models"
	"fmt"
	"os"

	"ferryman-agent/internal/prompt"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(ProvideDatabaseConfig, WorkingDir, Prompt)

type MCPType string

const (
	MCPStdio                 MCPType = "stdio"
	MCPSse                   MCPType = "sse"
	DefaultDataDirectory             = ".ferryer"
	MaxTokensFallbackDefault         = 200000
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

type DatabaseType string

const (
	DatabaseSQLite DatabaseType = "sqlite"
	DatabaseMySQL  DatabaseType = "mysql"
)

type DatabaseConfig struct {
	Type DatabaseType `json:"type"`
	DSN  string       `json:"dsn,omitempty"`

	Path string `json:"path,omitempty"`

	Host      string `json:"host,omitempty"`
	Port      int    `json:"port,omitempty"`
	Username  string `json:"username,omitempty"`
	Password  string `json:"password,omitempty"`
	Database  string `json:"database,omitempty"`
	Charset   string `json:"charset,omitempty"`
	ParseTime bool   `json:"parseTime,omitempty"`
	Loc       string `json:"loc,omitempty"`

	AutoMigrate         bool   `json:"autoMigrate,omitempty"`
	MaxOpenConns        int    `json:"maxOpenConns,omitempty"`
	MaxIdleConns        int    `json:"maxIdleConns,omitempty"`
	ConnMaxLifetimeSecs int    `json:"connMaxLifetimeSecs,omitempty"`
	LogLevel            string `json:"logLevel,omitempty"`
}

type ModelConfig struct {
	ModelId         models.ModelID `json:"model_id"`
	APIModel        string         `json:"api_model"`
	MaxTokens       int64          `json:"maxTokens,omitempty"`
	ReasoningEffort string         `json:"reasoning_effort,omitempty"`
	Weight          int            `json:"weight,omitempty"`
	Priority        int            `json:"priority,omitempty"`
}

func ApplyModelConfig(model models.Model, modelCfg ModelConfig) models.Model {
	if modelCfg.APIModel != "" {
		model.APIModel = modelCfg.APIModel
	}
	if modelCfg.MaxTokens > 0 {
		model.MaxTokens = modelCfg.MaxTokens
	}
	if modelCfg.ReasoningEffort != "" {
		model.ReasoningEffort = modelCfg.ReasoningEffort
	}
	return model
}

type ProviderConfig struct {
	Provider models.ModelProvider `json:"provider"`
	APIKey   string               `json:"apiKey"`
	BaseURL  string               `json:"baseURL"`
	Models   []ModelConfig        `json:"models"`
	Disabled bool                 `json:"disabled"`

	hasLegacyModelConfig bool
}

func (p *ProviderConfig) UnmarshalJSON(data []byte) error {
	type providerConfigAlias ProviderConfig
	var raw struct {
		providerConfigAlias
		ModelConfig json.RawMessage `json:"modelConfig"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*p = ProviderConfig(raw.providerConfigAlias)
	p.hasLegacyModelConfig = len(raw.ModelConfig) > 0 && string(raw.ModelConfig) != "null"
	return nil
}

func (p ProviderConfig) Configured() bool {
	return p.Provider != "" || p.APIKey != "" || p.BaseURL != "" || len(p.Models) > 0 || p.hasLegacyModelConfig
}

func (p ProviderConfig) PrimaryModelConfig() (ModelConfig, bool) {
	if len(p.Models) == 0 {
		return ModelConfig{}, false
	}
	return p.Models[0], true
}

type AgentModelConfig struct {
	ModelID  models.ModelID       `json:"model_id,omitempty"`
	Provider models.ModelProvider `json:"provider,omitempty"`
}

type Config struct {
	WorkingDir         string               `json:"workingDir,omitempty"`
	Database           DatabaseConfig       `json:"database,omitempty"`
	MCPServers         map[string]MCPServer `json:"mcpServers,omitempty"`
	Providers          []ProviderConfig     `json:"providers,omitempty"`
	Provider           ProviderConfig       `json:"provider,omitempty"`
	TitleProvider      ProviderConfig       `json:"titleProvider"`
	SummarizerProvider ProviderConfig       `json:"summarizerProvider,omitempty"`
	Agent              AgentModelConfig     `json:"agent,omitempty"`
	TitleAgent         AgentModelConfig     `json:"titleAgent,omitempty"`
	SummarizeAgent     AgentModelConfig     `json:"summarizeAgent,omitempty"`
	Debug              bool                 `json:"debug,omitempty"`
	AutoCompact        bool                 `json:"autoCompact,omitempty"`
	Prompt             prompt.PromptConfig  `json:"prompt,omitempty"`
	PromptConfigPath   string               `json:"promptConfigPath,omitempty"`
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
	if len(config.Providers) == 0 {
		config.Providers = legacyProviderConfigs(config)
	}
	config.Agent = defaultAgentModelConfig(config.Agent, primaryProviderConfig(config))
	config.TitleAgent = defaultAgentModelConfig(config.TitleAgent, config.TitleProvider)
	config.SummarizeAgent = defaultAgentModelConfig(config.SummarizeAgent, config.SummarizerProvider)
	for name, server := range config.MCPServers {
		if server.Type == "" {
			server.Type = MCPStdio
			config.MCPServers[name] = server
		}
	}
	return config
}

func ProviderConfigs(config Config) []ProviderConfig {
	if len(config.Providers) > 0 {
		return config.Providers
	}
	return legacyProviderConfigs(config)
}

func legacyProviderConfigs(config Config) []ProviderConfig {
	providerConfigs := []ProviderConfig{}
	if config.Provider.Configured() {
		providerConfigs = append(providerConfigs, config.Provider)
	}
	if config.TitleProvider.Configured() {
		providerConfigs = append(providerConfigs, config.TitleProvider)
	}
	if config.SummarizerProvider.Configured() {
		providerConfigs = append(providerConfigs, config.SummarizerProvider)
	}
	return providerConfigs
}

func primaryProviderConfig(config Config) ProviderConfig {
	if config.Provider.Configured() {
		return config.Provider
	}
	if len(config.Providers) > 0 {
		return config.Providers[0]
	}
	return ProviderConfig{}
}

func defaultAgentModelConfig(agentCfg AgentModelConfig, providerCfg ProviderConfig) AgentModelConfig {
	if agentCfg.ModelID == "" {
		if modelCfg, ok := providerCfg.PrimaryModelConfig(); ok {
			agentCfg.ModelID = modelCfg.ModelId
		}
	}
	if agentCfg.Provider == "" {
		agentCfg.Provider = providerCfg.Provider
	}
	return agentCfg
}

func Validate(config Config) error {
	if err := validateProviderConfig("provider", config.Provider); err != nil {
		return err
	}
	for i, providerCfg := range config.Providers {
		if err := validateProviderConfig(fmt.Sprintf("providers[%d]", i), providerCfg); err != nil {
			return err
		}
	}
	if err := validateOptionalProviderConfig("titleProvider", config.TitleProvider); err != nil {
		return err
	}
	if err := validateOptionalProviderConfig("summarizerProvider", config.SummarizerProvider); err != nil {
		return err
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

func validateOptionalProviderConfig(name string, providerCfg ProviderConfig) error {
	if !providerCfg.Configured() {
		return nil
	}
	return validateProviderConfig(name, providerCfg)
}

func validateProviderConfig(name string, providerCfg ProviderConfig) error {
	if providerCfg.hasLegacyModelConfig {
		return fmt.Errorf("%s modelConfig is not supported; use models", name)
	}
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
		if modelCfg.ModelId == "" {
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
