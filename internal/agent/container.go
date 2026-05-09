package agent

import (
	workspace "ferryman-agent/internal/capability/workspace"
	sdkconfig "ferryman-agent/internal/config"
	datadb "ferryman-agent/internal/data/db"
	"ferryman-agent/internal/data/repo"
	"ferryman-agent/internal/memory/history"
	"ferryman-agent/internal/memory/message"
	"ferryman-agent/internal/memory/session"
	"ferryman-agent/internal/prompt"
	"ferryman-agent/internal/provider"
	"ferryman-agent/internal/security/permission"
)

type Container struct {
	DB             *datadb.DbClient
	Sessions       session.Service
	Messages       message.Service
	History        history.Service
	Permissions    permission.Service
	Prompt         prompt.Service
	Workspace      workspace.Workspace
	ProviderRouter provider.Router
	SessionRepo    repo.SessionRepo
	MessageRepo    repo.MessageRepo
	HistoryRepo    repo.HistoryRepo
}

func buildContainer(cfg *sdkconfig.Config) (*Container, error) {
	dbClient, err := datadb.NewDbClient(sdkconfig.ProvideDatabaseConfig(cfg))
	if err != nil {
		return nil, err
	}
	sessionRepo := repo.NewSessionRepo(dbClient)
	messageRepo := repo.NewMessageRepo(dbClient)
	historyRepo := repo.NewHistoryRepo(dbClient)

	sessions := session.NewService(sessionRepo)
	messages := message.NewService(messageRepo)
	historySvc := history.NewService(historyRepo)
	workingDir := sdkconfig.WorkingDir(cfg)
	permissions := permission.NewServiceWithWorkingDir(workingDir)
	promptSvc, err := prompt.NewService(sdkconfig.Prompt(cfg))
	if err != nil {
		return nil, err
	}
	ws := workspace.NewWorkspace(workingDir)
	providerRouter, err := provider.NewDefaultRouter(sdkconfig.ProviderConfigs(*cfg)...)
	if err != nil {
		return nil, err
	}

	return NewContainer(
		dbClient,
		sessions,
		messages,
		historySvc,
		permissions,
		promptSvc,
		ws,
		providerRouter,
		sessionRepo,
		messageRepo,
		historyRepo,
	)
}

func NewContainer(
	db *datadb.DbClient,
	sessions session.Service,
	messages message.Service,
	history history.Service,
	permissions permission.Service,
	promptSvc prompt.Service,
	workspace workspace.Workspace,
	providerRouter provider.Router,
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
		DB:             db,
		Sessions:       sessions,
		Messages:       messages,
		History:        history,
		Permissions:    permissions,
		Prompt:         promptSvc,
		Workspace:      workspace,
		ProviderRouter: providerRouter,
		SessionRepo:    sessionRepo,
		MessageRepo:    messageRepo,
		HistoryRepo:    historyRepo,
	}, nil
}
