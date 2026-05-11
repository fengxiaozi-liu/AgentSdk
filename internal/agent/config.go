package agent

import (
	"fmt"
	"os"
	"strings"

	"ferryman-agent/internal/data/llm/models"
	"ferryman-agent/internal/memory/history"
	"ferryman-agent/internal/memory/message"
	"ferryman-agent/internal/memory/session"
	"ferryman-agent/internal/prompt"
	"ferryman-agent/internal/provider"
	toolcore "ferryman-agent/internal/tools"
)

type AgentConfig struct {
	WorkingDir  string
	Memory      MemoryConfig
	Prompt      PromptConfig
	Provider    ProviderRoutingConfig
	Tools       []toolcore.BaseTool
	Debug       bool
	AutoCompact bool
}

type MemoryConfig struct {
	Session  session.Service
	Messages message.Service
	History  history.Service
}

type PromptConfig struct {
	Prompt         prompt.Service
	AgentSystemKey string
}

type ProviderRoutingConfig struct {
	Router       provider.Router
	DefaultModel ModelTarget
}

type ModelTarget struct {
	Provider models.ModelProvider
	ModelID  models.ModelID
}

type ProviderConfig struct {
	Provider models.ModelProvider `json:"provider"`
	APIKey   string               `json:"apiKey,omitempty"`
	BaseURL  string               `json:"baseURL,omitempty"`
	Models   []ModelConfig        `json:"models"`
	Disabled bool                 `json:"disabled,omitempty"`
}

type ModelConfig struct {
	ModelID         models.ModelID `json:"model_id"`
	APIModel        string         `json:"api_model,omitempty"`
	MaxTokens       int64          `json:"maxTokens,omitempty"`
	ReasoningEffort string         `json:"reasoning_effort,omitempty"`
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

func DefaultAgentConfig() AgentConfig {
	promptSvc := prompt.NewDefault()
	return AgentConfig{
		Prompt: PromptConfig{
			Prompt:         promptSvc,
			AgentSystemKey: prompt.KeyCoder,
		},
	}
}

func normalizeConfig(cfg *AgentConfig) error {
	if strings.TrimSpace(cfg.WorkingDir) == "" {
		if wd, err := os.Getwd(); err == nil {
			cfg.WorkingDir = wd
		}
	}
	if cfg.Prompt.Prompt == nil {
		cfg.Prompt.Prompt = prompt.NewDefault()
	}
	if strings.TrimSpace(cfg.Prompt.AgentSystemKey) == "" {
		cfg.Prompt.AgentSystemKey = prompt.KeyCoder
	}
	return nil
}

func validateConfig(cfg AgentConfig) error {
	if cfg.Memory.Session == nil || cfg.Memory.Messages == nil || cfg.Memory.History == nil {
		return fmt.Errorf("memory session, messages, and history services must be configured together")
	}
	if cfg.Provider.Router == nil {
		return fmt.Errorf("provider router is required")
	}
	if cfg.Provider.DefaultModel.Provider == "" || cfg.Provider.DefaultModel.ModelID == "" {
		return fmt.Errorf("agent provider and model are required")
	}
	if cfg.Prompt.Prompt == nil {
		return fmt.Errorf("prompt service is required")
	}
	return nil
}
