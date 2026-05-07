package security

import (
	"ferryman-agent/internal/security/permission"

	"github.com/google/wire"
)

var ProviderSet = wire.NewSet(permission.ProviderSet)
