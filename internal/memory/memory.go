package memory

import (
	"ferryman-agent/internal/memory/history"
	"ferryman-agent/internal/memory/message"
	"ferryman-agent/internal/memory/session"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(
	history.ProviderSet,
	message.ProviderSet,
	session.ProviderSet,
)
