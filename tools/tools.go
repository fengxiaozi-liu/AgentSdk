package tools

import (
	agentlsp "ferryman-agent/extensions/lsp"
	"ferryman-agent/history"
	"ferryman-agent/permission"
	toolcore "ferryman-agent/tools/core"
)

type toolsetConfig struct {
	permissions permission.Service
	history     history.Service
	lspClients  map[string]*agentlsp.Client
	tools       []toolcore.BaseTool
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

func WithBaseTools(baseTools ...toolcore.BaseTool) Option {
	return func(cfg *toolsetConfig) {
		cfg.tools = append(cfg.tools, baseTools...)
	}
}

func WithMCPTools(mcpTools ...toolcore.BaseTool) Option {
	return func(cfg *toolsetConfig) {
		cfg.tools = append(cfg.tools, mcpTools...)
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
	return append([]toolcore.BaseTool{}, cfg.tools...)
}
