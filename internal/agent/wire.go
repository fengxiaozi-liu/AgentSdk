//go:build wireinject

package agent

import (
	"ferryman-agent/internal/capability/workspace"
	"ferryman-agent/internal/config"
	"ferryman-agent/internal/data/db"
	"ferryman-agent/internal/data/repo"
	"ferryman-agent/internal/memory/history"
	"ferryman-agent/internal/memory/message"
	"ferryman-agent/internal/memory/session"
	"ferryman-agent/internal/prompt"
	"ferryman-agent/internal/security/permission"

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
