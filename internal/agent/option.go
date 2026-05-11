package agent

import (
	"context"
	"fmt"

	"ferryman-agent/internal/prompt"

	mcptools "ferryman-agent/internal/capability/mcp"
	datadb "ferryman-agent/internal/data/db"
	clientfactory "ferryman-agent/internal/data/llm/client/factory"
	"ferryman-agent/internal/data/llm/models"
	"ferryman-agent/internal/data/repo"
	"ferryman-agent/internal/memory/history"
	"ferryman-agent/internal/memory/message"
	"ferryman-agent/internal/memory/session"
	"ferryman-agent/internal/provider"
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
		dbClient, err := datadb.NewDbClient(database)
		if err != nil {
			return err
		}
		if database.AutoMigrate {
			if err := dbClient.AutoMigrate(&repo.SessionRecord{}, &repo.MessageRecord{}, &repo.HistoryRecord{}); err != nil {
				return err
			}
		}
		sessionRepo := repo.NewSessionRepo(dbClient)
		messageRepo := repo.NewMessageRepo(dbClient)
		historyRepo := repo.NewHistoryRepo(dbClient)
		cfg.Memory.Session = session.NewService(sessionRepo)
		cfg.Memory.Messages = message.NewService(messageRepo)
		cfg.Memory.History = history.NewService(historyRepo)
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

func WithProviderConfigs(defaultModel ModelTarget, configs ...ProviderConfig) Option {
	return func(cfg *AgentConfig) error {
		targets := map[models.ModelProvider]map[models.ModelID]provider.ProviderClient{}
		for _, providerCfg := range configs {
			if providerCfg.Disabled {
				continue
			}
			if providerCfg.Provider == "" {
				return provider.ErrProviderNotConfigured
			}
			if len(providerCfg.Models) == 0 {
				return fmt.Errorf("%w: %s", provider.ErrModelNotConfigured, providerCfg.Provider)
			}

			vendorClient, err := clientfactory.NewClient(
				providerCfg.Provider,
				clientfactory.WithAPIKey(providerCfg.APIKey),
				clientfactory.WithBaseURL(providerCfg.BaseURL),
			)
			if err != nil {
				return err
			}

			for _, modelCfg := range providerCfg.Models {
				if modelCfg.ModelID == "" {
					return fmt.Errorf("%w: empty model_id for %s", provider.ErrModelNotConfigured, providerCfg.Provider)
				}

				model := ApplyModelConfig(models.ResolveModel(providerCfg.Provider, modelCfg.ModelID), modelCfg)
				if targets[providerCfg.Provider] == nil {
					targets[providerCfg.Provider] = map[models.ModelID]provider.ProviderClient{}
				}
				if _, exists := targets[providerCfg.Provider][model.ID]; exists {
					return fmt.Errorf("%w: %s/%s", provider.ErrProviderTargetExists, providerCfg.Provider, model.ID)
				}

				targets[providerCfg.Provider][model.ID] = provider.ProviderClient{
					Provider: providerCfg.Provider,
					Model:    model,
					Client:   vendorClient,
				}
			}
		}
		cfg.Provider.Router = provider.NewDefaultRouter(targets)
		cfg.Provider.DefaultModel = defaultModel
		return nil
	}
}

func WithAgentProviders(defaultModel ModelTarget, configs ...ProviderConfig) Option {
	return WithProviderConfigs(defaultModel, configs...)
}

func WithProviderRouting(providerRouting ProviderRoutingConfig) Option {
	return func(cfg *AgentConfig) error {
		cfg.Provider = providerRouting
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

func WithPrompt(config PromptConfig) Option {
	return func(cfg *AgentConfig) error {
		cfg.Prompt.Prompt = config.Prompt
		cfg.Prompt.AgentSystemKey = config.AgentSystemKey
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
