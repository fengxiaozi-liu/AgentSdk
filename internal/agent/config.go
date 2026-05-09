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
	Provider    AgentProviderRouter
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

type AgentProviderRouter struct {
	Router        provider.Router
	AgentProvider AgentProvider
}

type AgentProvider struct {
	Provider models.ModelProvider
	ModelID  models.ModelID
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
	if cfg.Provider.AgentProvider.Provider == "" || cfg.Provider.AgentProvider.ModelID == "" {
		return fmt.Errorf("agent provider and model are required")
	}
	if cfg.Prompt.Prompt == nil {
		return fmt.Errorf("prompt service is required")
	}
	return nil
}
