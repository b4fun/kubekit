package kubekit

import (
	"github.com/b4fun/kubekit/internal/logger"
)

type (
	Logger  = logger.Logger
	LogFunc = logger.LogFunc
)

var (
	NewStdLogger = logger.NewStdLogger
)
