package agent

import (
	datadb "ferryman-agent/internal/data/db"
	"ferryman-agent/internal/data/llm/models"
	providersvc "ferryman-agent/internal/provider"
	toolcore "ferryman-agent/internal/tools"
)

type AgentOption func(*agentOptions)

type agentOptions struct {
	tools               []toolcore.BaseTool
	enableAgentTool     bool
	enableMcpTool       bool
	enableWorkSpaceTool bool
	promptKey           string
	providers           []providersvc.ProviderRegister
	database            *datadb.DatabaseConfig
	modelID             models.ModelID
	provider            models.ModelProvider
}

func WithTools(tools ...toolcore.BaseTool) AgentOption {
	return func(opts *agentOptions) {
		opts.tools = append(opts.tools, tools...)
	}
}

func WithAgentTool() AgentOption {
	return func(opts *agentOptions) {
		opts.enableAgentTool = true
	}
}

func WithWorkSpaceTool() AgentOption {
	return func(opts *agentOptions) {
		opts.enableWorkSpaceTool = true
	}
}

func WithMcpTool() AgentOption {
	return func(opts *agentOptions) {
		opts.enableMcpTool = true
	}
}

func WithMCPTool() AgentOption {
	return WithMcpTool()
}

func WithPromptKey(key string) AgentOption {
	return func(opts *agentOptions) {
		opts.promptKey = key
	}
}

func WithProviders(providers ...providersvc.ProviderRegister) AgentOption {
	return func(opts *agentOptions) {
		opts.providers = append(opts.providers, providers...)
	}
}

func WithModel(provider models.ModelProvider, modelID models.ModelID) AgentOption {
	return func(opts *agentOptions) {
		opts.provider = provider
		opts.modelID = modelID
	}
}

func WithDatabase(database datadb.DatabaseConfig) AgentOption {
	return func(opts *agentOptions) {
		opts.database = &database
	}
}

func applyAgentOptions(opts ...AgentOption) agentOptions {
	agentOpts := agentOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(&agentOpts)
		}
	}
	return agentOpts
}
