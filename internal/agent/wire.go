//go:build wireinject

package agent

import (
	"ferryman-agent/internal/capability/workspace"
	"ferryman-agent/internal/config"
	"ferryman-agent/internal/data"
	"ferryman-agent/internal/memory"
	"ferryman-agent/internal/prompt"
	providersvc "ferryman-agent/internal/provider"
	"ferryman-agent/internal/security"

	"github.com/google/wire"
)

func wireContainer(cfg *config.Config) (*Container, error) {
	panic(wire.Build(
		config.ProviderSet,
		data.ProviderSet,
		memory.ProviderSet,
		prompt.ProviderSet,
		providersvc.ProviderSet,
		workspace.ProviderSet,
		security.ProviderSet,
		ProviderSet,
	))
}
