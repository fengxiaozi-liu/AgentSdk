package agent

import (
	"github.com/opencode-ai/opencode/agent/config"
	"github.com/opencode-ai/opencode/agent/history"
	"github.com/opencode-ai/opencode/agent/memory"
	"github.com/opencode-ai/opencode/agent/message"
	"github.com/opencode-ai/opencode/agent/permission"
	"github.com/opencode-ai/opencode/agent/session"
)

type Runtime struct {
	Config      config.AgentRuntimeConfig
	Sessions    session.Service
	Messages    message.Service
	History     history.Service
	Memory      memory.Service
	Permissions permission.Service
}

func NewRuntime(
	cfg config.AgentRuntimeConfig,
	sessions session.Service,
	messages message.Service,
	files history.Service,
	permissions permission.Service,
) *Runtime {
	return &Runtime{
		Config:      cfg,
		Sessions:    sessions,
		Messages:    messages,
		History:     files,
		Memory:      memory.NewService(sessions, messages, files),
		Permissions: permissions,
	}
}
