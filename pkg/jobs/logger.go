package jobs

import (
	"go.uber.org/zap"
)

var (
	log  *zap.SugaredLogger
	atom zap.AtomicLevel
)

// InitLogger sets the logger and log level used by this package.
func InitLogger(logger *zap.SugaredLogger, level zap.AtomicLevel) {
	log = logger
	atom = level
}

func debug() bool {
	return atom.Level() == zap.DebugLevel
}
