//go:build wireinject

package agent

import (
	"ferryman-agent/internal/config"
	"ferryman-agent/internal/data/db"
	"ferryman-agent/internal/data/repo"
	"ferryman-agent/internal/service/history"
	"ferryman-agent/internal/service/message"
	"ferryman-agent/internal/service/permission"
	"ferryman-agent/internal/service/prompt"
	"ferryman-agent/internal/service/session"
	"ferryman-agent/internal/tools/workspace"

	"github.com/google/wire"
)

func wireContainer(cfg *config.Config) (*Container, error) {
	panic(wire.Build(
		config.ProviderSet,
		db.ProviderSet,
		repo.ProviderSet,
		session.ProviderSet,
		message.ProviderSet,
		history.ProviderSet,
		prompt.NewService,
		workspace.ProviderSet,
		permission.ProviderSet,
		NewContainer,
	))
}
