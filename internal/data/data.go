package data

import (
	"ferryman-agent/internal/data/db"
	"ferryman-agent/internal/data/repo"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	db.ProviderSet,
	repo.ProviderSet,
)
