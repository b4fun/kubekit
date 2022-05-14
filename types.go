package podkit

import (
	"github.com/b4fun/podkit/internal/logger"
)

type (
	Logger  = logger.Logger
	LogFunc = logger.LogFunc
)

var (
	NewStdLogger = logger.NewStdLogger
)
