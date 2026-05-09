package agent

import (
	"context"

	mcptools "ferryman-agent/internal/capability/mcp"
	datadb "ferryman-agent/internal/data/db"
	"ferryman-agent/internal/data/llm/models"
	"ferryman-agent/internal/data/repo"
	"ferryman-agent/internal/memory/history"
	"ferryman-agent/internal/memory/message"
	"ferryman-agent/internal/memory/session"
	"ferryman-agent/internal/prompt"
	providersvc "ferryman-agent/internal/provider"
	"ferryman-agent/internal/security/permission"
	toolcore "ferryman-agent/internal/tools"
)

const PromptKeyDefault = "default"

type Option func(*AgentConfig) error

func WithWorkingDir(path string) Option {
	return func(cfg *AgentConfig) error {
		cfg.WorkingDir = path
		return nil
	}
}

func WithDatabaseConfig(database datadb.DatabaseConfig) Option {
	return func(cfg *AgentConfig) error {
		memory, err := memoryFromDatabase(database)
		if err != nil {
			return err
		}
		cfg.Memory = memory
		return nil
	}
}

func WithMemoryRepos(sessionRepo repo.SessionRepo, messageRepo repo.MessageRepo, historyRepo repo.HistoryRepo) Option {
	return func(cfg *AgentConfig) error {
		cfg.Memory.Session = session.NewService(sessionRepo)
		cfg.Memory.Messages = message.NewService(messageRepo)
		cfg.Memory.History = history.NewService(historyRepo)
		return nil
	}
}

func WithMemoryServices(sessions session.Service, messages message.Service, history history.Service) Option {
	return func(cfg *AgentConfig) error {
		cfg.Memory.Session = sessions
		cfg.Memory.Messages = messages
		cfg.Memory.History = history
		return nil
	}
}

func WithProviderConfig(configs ...providersvc.ProviderConfig) Option {
	return func(cfg *AgentConfig) error {
		router, err := providersvc.NewDefaultRouter(configs...)
		if err != nil {
			return err
		}
		cfg.Provider.Router = router
		if cfg.Provider.AgentProvider.ModelID == "" || cfg.Provider.AgentProvider.Provider == "" {
			if model, ok := firstModelRef(configs); ok {
				cfg.Provider.AgentProvider = model
			}
		}
		return nil
	}
}

func WithProviderRouter(router providersvc.Router) Option {
	return func(cfg *AgentConfig) error {
		cfg.Provider.Router = router
		return nil
	}
}

func WithModel(provider models.ModelProvider, modelID models.ModelID) Option {
	return func(cfg *AgentConfig) error {
		cfg.Provider.AgentProvider = ModelRef{Provider: provider, ModelID: modelID}
		return nil
	}
}

func WithTools(tools ...toolcore.BaseTool) Option {
	return func(cfg *AgentConfig) error {
		cfg.Tools = append(cfg.Tools, tools...)
		return nil
	}
}

func WithMCPServers(servers map[string]mcptools.MCPServer) Option {
	return func(cfg *AgentConfig) error {
		tools, err := mcptools.LoadTools(context.Background(), servers, permission.NewServiceWithWorkingDir(cfg.WorkingDir), cfg.WorkingDir)
		if err != nil {
			return err
		}
		cfg.Tools = append(cfg.Tools, tools...)
		return nil
	}
}

func WithMCPToolLoader(loader mcptools.MCPToolLoader) Option {
	return func(cfg *AgentConfig) error {
		if loader == nil {
			return nil
		}
		servers, err := loader.Load(context.Background())
		if err != nil {
			return err
		}
		tools, err := mcptools.LoadTools(context.Background(), servers, permission.NewServiceWithWorkingDir(cfg.WorkingDir), cfg.WorkingDir)
		if err != nil {
			return err
		}
		cfg.Tools = append(cfg.Tools, tools...)
		return nil
	}
}

func WithPrompt(service prompt.Service) Option {
	return func(cfg *AgentConfig) error {
		cfg.Prompt.Prompt = service
		return nil
	}
}

func WithSystemValue(value string) Option {
	return func(cfg *AgentConfig) error {
		if cfg.Prompt.Prompt == nil {
			cfg.Prompt.Prompt = prompt.NewDefault()
		}
		cfg.Prompt.Prompt.SetPrompt(PromptKeyDefault, value)
		cfg.Prompt.AgentSystemKey = PromptKeyDefault
		return nil
	}
}

func WithAgentSystemKey(key string) Option {
	return func(cfg *AgentConfig) error {
		cfg.Prompt.AgentSystemKey = key
		return nil
	}
}

func WithDebug(enabled bool) Option {
	return func(cfg *AgentConfig) error {
		cfg.Debug = enabled
		return nil
	}
}

func WithAutoCompact(enabled bool) Option {
	return func(cfg *AgentConfig) error {
		cfg.AutoCompact = enabled
		return nil
	}
}

func firstModelRef(configs []providersvc.ProviderConfig) (ModelRef, bool) {
	for _, cfg := range configs {
		if cfg.Disabled {
			continue
		}
		model, ok := cfg.PrimaryModelConfig()
		if !ok {
			continue
		}
		return ModelRef{Provider: cfg.Provider, ModelID: model.ModelID}, true
	}
	return ModelRef{}, false
}
