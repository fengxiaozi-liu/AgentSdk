package agent

import (
	"ferryman-agent/internal/data/llm/models"
	"ferryman-agent/internal/memory/history"
	"ferryman-agent/internal/memory/message"
	"ferryman-agent/internal/memory/session"
	"ferryman-agent/internal/prompt"
	"ferryman-agent/internal/provider"
	toolcore "ferryman-agent/internal/tools"
)

type AgentConfig struct {
	WorkingDir string

	Memory   MemoryConfig
	Prompt   PromptConfig
	Provider ProviderConfig

	Tools []toolcore.BaseTool

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

type ProviderConfig struct {
	Router        provider.Router
	AgentProvider ModelRef
}

type ModelRef struct {
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
