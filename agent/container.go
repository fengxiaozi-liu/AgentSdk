package agent

import (
	datadb "ferryman-agent/data/db"
	"ferryman-agent/data/repo"
	"ferryman-agent/history"
	"ferryman-agent/message"
	"ferryman-agent/permission"
	"ferryman-agent/session"
	toolcore "ferryman-agent/tools/core"
	workspace "ferryman-agent/tools/workspace"
)

type Container struct {
	DB          *datadb.DbClient
	Sessions    session.Service
	Messages    message.Service
	History     history.Service
	Permissions permission.Service
	Workspace   workspace.Workspace
	Tools       []toolcore.BaseTool
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
	workspace workspace.Workspace,
	tools []toolcore.BaseTool,
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
		Workspace:   workspace,
		Tools:       tools,
		SessionRepo: sessionRepo,
		MessageRepo: messageRepo,
		HistoryRepo: historyRepo,
	}, nil
}
