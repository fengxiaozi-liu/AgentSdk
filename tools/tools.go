package tools

import (
	"context"
	agentlsp "ferryman-agent/extensions/lsp"
	"ferryman-agent/history"
	"ferryman-agent/message"
	"ferryman-agent/permission"
	"ferryman-agent/session"
	agenttool "ferryman-agent/tools/agent"
	toolcore "ferryman-agent/tools/core"
	mcptools "ferryman-agent/tools/mcp"
)

type toolsetConfig struct {
	permissions permission.Service
	history     history.Service
	lspClients  map[string]*agentlsp.Client
	ctx         context.Context
	sessions    session.Service
	messages    message.Service
	tools       []toolcore.BaseTool
	builders    []func(toolsetConfig) []toolcore.BaseTool
}

type Option func(*toolsetConfig)

func WithPermissions(permissions permission.Service) Option {
	return func(cfg *toolsetConfig) {
		cfg.permissions = permissions
	}
}

func WithHistory(historySvc history.Service) Option {
	return func(cfg *toolsetConfig) {
		cfg.history = historySvc
	}
}

func WithLSPClients(lspClients map[string]*agentlsp.Client) Option {
	return func(cfg *toolsetConfig) {
		cfg.lspClients = lspClients
	}
}

func WithContext(ctx context.Context) Option {
	return func(cfg *toolsetConfig) {
		cfg.ctx = ctx
	}
}

func WithSessions(sessions session.Service) Option {
	return func(cfg *toolsetConfig) {
		cfg.sessions = sessions
	}
}

func WithMessages(messages message.Service) Option {
	return func(cfg *toolsetConfig) {
		cfg.messages = messages
	}
}

func WithBaseTools(baseTools ...toolcore.BaseTool) Option {
	return func(cfg *toolsetConfig) {
		cfg.tools = append(cfg.tools, baseTools...)
	}
}

func WithMCPTools() Option {
	return func(cfg *toolsetConfig) {
		cfg.builders = append(cfg.builders, func(cfg toolsetConfig) []toolcore.BaseTool {
			ctx := cfg.ctx
			if ctx == nil {
				ctx = context.Background()
			}
			return mcptools.GetMcpTools(ctx, cfg.permissions)
		})
	}
}

func WithAgentTools() Option {
	return func(cfg *toolsetConfig) {
		cfg.builders = append(cfg.builders, func(cfg toolsetConfig) []toolcore.BaseTool {
			if cfg.sessions == nil || cfg.messages == nil {
				return nil
			}
			return []toolcore.BaseTool{
				agenttool.New(cfg.sessions, cfg.messages, cfg.lspClients),
			}
		})
	}
}

func applyOptions(opts ...Option) toolsetConfig {
	cfg := toolsetConfig{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	return cfg
}

func NewTaskToolset(opts ...Option) []toolcore.BaseTool {
	cfg := applyOptions(opts...)
	toolset := append([]toolcore.BaseTool{}, cfg.tools...)
	for _, builder := range cfg.builders {
		if builder != nil {
			toolset = append(toolset, builder(cfg)...)
		}
	}
	return toolset
}
