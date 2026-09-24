package etcd

import (
	"go.uber.org/zap"
)

var (
	log *zap.SugaredLogger
)

// InitLogger sets the logger used by this package.
func InitLogger(logger *zap.SugaredLogger) {
	log = logger
}
