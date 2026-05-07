//go:build wireinject

package agent

import (
	"ferryman-agent/config"
	"ferryman-agent/data/db"
	"ferryman-agent/data/repo"
	"ferryman-agent/history"
	"ferryman-agent/message"
	"ferryman-agent/permission"
	"ferryman-agent/session"
	"ferryman-agent/tools/workspace"

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
		workspace.ProviderSet,
		permission.ProviderSet,
		NewContainer,
	))
}
