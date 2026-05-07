package agent

import (
	"context"
	mcptools "ferryman-agent/internal/capability/mcp"
	workspace "ferryman-agent/internal/capability/workspace"
	datadb "ferryman-agent/internal/data/db"
	"ferryman-agent/internal/data/repo"
	"ferryman-agent/internal/memory/history"
	"ferryman-agent/internal/memory/message"
	"ferryman-agent/internal/memory/session"
	"ferryman-agent/internal/prompt"
	"ferryman-agent/internal/security/permission"
	toolcore "ferryman-agent/internal/tools"
)

type Container struct {
	DB          *datadb.DbClient
	Sessions    session.Service
	Messages    message.Service
	History     history.Service
	Permissions permission.Service
	Prompt      prompt.Service
	Workspace   workspace.Workspace
	SessionRepo repo.SessionRepo
	MessageRepo repo.MessageRepo
	HistoryRepo repo.HistoryRepo
}

func NewContainer(
	db *datadb.DbClient,
	sessions session.Service,
	messages message.Service,
	history history.Service,
	permissions permission.Service,
	promptSvc prompt.Service,
	workspace workspace.Workspace,
	sessionRepo repo.SessionRepo,
	messageRepo repo.MessageRepo,
	historyRepo repo.HistoryRepo,
) (*Container, error) {
	if db.Config.AutoMigrate {
		if err := db.AutoMigrate(&repo.SessionRecord{}, &repo.MessageRecord{}, &repo.HistoryRecord{}); err != nil {
			return nil, err
		}
	}
	return &Container{
		DB:          db,
		Sessions:    sessions,
		Messages:    messages,
		History:     history,
		Permissions: permissions,
		Prompt:      promptSvc,
		Workspace:   workspace,
		SessionRepo: sessionRepo,
		MessageRepo: messageRepo,
		HistoryRepo: historyRepo,
	}, nil
}

func (c *Container) DefaultTools() []toolcore.BaseTool {
	baseTools := []toolcore.BaseTool{
		workspace.NewGlobTool(c.Workspace),
		workspace.NewGrepTool(c.Workspace),
		workspace.NewLsTool(c.Workspace),
		workspace.NewSourcegraphTool(),
		workspace.NewViewTool(c.Workspace),
		workspace.NewEditTool(c.Workspace, c.Permissions, c.History),
		workspace.NewWriteTool(c.Workspace, c.Permissions, c.History),
		workspace.NewPatchTool(c.Workspace, c.Permissions, c.History),
		workspace.NewBashTool(c.Workspace, c.Permissions),
		workspace.NewFetchTool(c.Workspace, c.Permissions),
	}
	return append(
		baseTools,
		mcptools.GetMcpTools(context.Background(), c.Permissions)...,
	)
}
