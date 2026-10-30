package pg

import (
	"context"

	"go.uber.org/zap"
)

var (
	log *zap.SugaredLogger
	ctx context.Context
	// ValidRoles lists the roles a database can be verified against.
	ValidRoles = map[string]bool{
		"primary": true,
		"standby": true,
	}
)

// InitLogger sets the logger used by this package.
func InitLogger(logger *zap.SugaredLogger) {
	log = logger
}

// InitContext sets the context used for database operations.
func InitContext(c context.Context) {
	ctx = c
}
