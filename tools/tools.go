package tools

import (
	"context"
	"ferryman-agent/history"
	"ferryman-agent/message"
	"ferryman-agent/permission"
	"ferryman-agent/prompt"
	"ferryman-agent/session"
	agenttool "ferryman-agent/tools/agent"
	toolcore "ferryman-agent/tools/core"
	mcptools "ferryman-agent/tools/mcp"
	basetools "ferryman-agent/tools/workspace"
)

type toolsetConfig struct {
	permissions permission.Service
	history     history.Service
	ctx         context.Context
	sessions    session.Service
	messages    message.Service
	prompts     prompt.Service
	workspace   basetools.Workspace
	tools       []toolcore.BaseTool
	fileHooks   []toolcore.FileHook
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

func WithPrompts(prompts prompt.Service) Option {
	return func(cfg *toolsetConfig) {
		cfg.prompts = prompts
	}
}

func WithWorkspace(workspace basetools.Workspace) Option {
	return func(cfg *toolsetConfig) {
		cfg.workspace = workspace
	}
}

func WithBaseTools(baseTools ...toolcore.BaseTool) Option {
	return func(cfg *toolsetConfig) {
		cfg.tools = append(cfg.tools, baseTools...)
	}
}

func WithFileHooks(hooks ...toolcore.FileHook) Option {
	return func(cfg *toolsetConfig) {
		cfg.fileHooks = append(cfg.fileHooks, hooks...)
	}
}

func WithBaseFileTools() Option {
	return func(cfg *toolsetConfig) {
		cfg.builders = append(cfg.builders, func(cfg toolsetConfig) []toolcore.BaseTool {
			baseTools := []toolcore.BaseTool{
				basetools.NewViewTool(cfg.workspace, cfg.fileHooks...),
			}
			if cfg.permissions != nil && cfg.history != nil {
				baseTools = append(baseTools,
					basetools.NewEditTool(cfg.workspace, cfg.permissions, cfg.history, cfg.fileHooks...),
					basetools.NewWriteTool(cfg.workspace, cfg.permissions, cfg.history, cfg.fileHooks...),
					basetools.NewPatchTool(cfg.workspace, cfg.permissions, cfg.history, cfg.fileHooks...),
				)
			}
			return baseTools
		})
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
				agenttool.New(cfg.sessions, cfg.messages, cfg.prompts),
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
