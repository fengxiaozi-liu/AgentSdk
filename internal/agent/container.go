package agent

import (
	workspace "ferryman-agent/internal/capability/workspace"
	datadb "ferryman-agent/internal/data/db"
	"ferryman-agent/internal/data/repo"
	"ferryman-agent/internal/memory/history"
	"ferryman-agent/internal/memory/message"
	"ferryman-agent/internal/memory/session"
	"ferryman-agent/internal/prompt"
	providersvc "ferryman-agent/internal/provider"
	"ferryman-agent/internal/security/permission"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(NewContainer)

type Container struct {
	DB              *datadb.DbClient
	Sessions        session.Service
	Messages        message.Service
	History         history.Service
	Permissions     permission.Service
	Prompt          prompt.Service
	Workspace       workspace.Workspace
	ProviderService providersvc.Service
	SessionRepo     repo.SessionRepo
	MessageRepo     repo.MessageRepo
	HistoryRepo     repo.HistoryRepo
}

func NewContainer(
	db *datadb.DbClient,
	sessions session.Service,
	messages message.Service,
	history history.Service,
	permissions permission.Service,
	promptSvc prompt.Service,
	workspace workspace.Workspace,
	providerService providersvc.Service,
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
		DB:              db,
		Sessions:        sessions,
		Messages:        messages,
		History:         history,
		Permissions:     permissions,
		Prompt:          promptSvc,
		Workspace:       workspace,
		ProviderService: providerService,
		SessionRepo:     sessionRepo,
		MessageRepo:     messageRepo,
		HistoryRepo:     historyRepo,
	}, nil
}
