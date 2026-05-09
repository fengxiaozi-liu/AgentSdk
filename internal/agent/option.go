package agent

import (
	mcptools "ferryman-agent/internal/capability/mcp"
	datadb "ferryman-agent/internal/data/db"
	"ferryman-agent/internal/data/llm/models"
	"ferryman-agent/internal/prompt"
	providersvc "ferryman-agent/internal/provider"
	toolcore "ferryman-agent/internal/tools"
)

type AgentOption func(*agentOptions)

type agentOptions struct {
	tools               []toolcore.BaseTool
	enableAgentTool     bool
	enableWorkSpaceTool bool
	mcpServers          map[string]mcptools.MCPServer
	mcpToolLoader       mcptools.MCPToolLoader
	disableMCP          bool
	systemPrompt        string
	systemPromptSet     bool
	systemPromptRef     *systemPromptRef
	promptKey           string
	providers           []providersvc.ProviderConfig
	database            *datadb.DatabaseConfig
	modelID             models.ModelID
	provider            models.ModelProvider
}

type systemPromptRef struct {
	service prompt.Service
	key     string
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

func WithMCPServers(servers map[string]mcptools.MCPServer) AgentOption {
	return func(opts *agentOptions) {
		opts.mcpServers = servers
	}
}

func WithMCPToolLoader(loader mcptools.MCPToolLoader) AgentOption {
	return func(opts *agentOptions) {
		opts.mcpToolLoader = loader
	}
}

func DisableMCP() AgentOption {
	return func(opts *agentOptions) {
		opts.disableMCP = true
	}
}

func WithPromptKey(key string) AgentOption {
	return func(opts *agentOptions) {
		opts.promptKey = key
	}
}

func WithSystemPrompt(value string) AgentOption {
	return func(opts *agentOptions) {
		opts.systemPrompt = value
		opts.systemPromptSet = true
	}
}

func WithSystemPromptFrom(service prompt.Service, key string) AgentOption {
	return func(opts *agentOptions) {
		opts.systemPromptRef = &systemPromptRef{service: service, key: key}
	}
}

func WithProviders(providers ...providersvc.ProviderConfig) AgentOption {
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
