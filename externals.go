package gingorr

import (
	"context"

	"go.uber.org/zap"
)

type ILoggerFactory interface {
  NewLogger(context.Context) *zap.Logger
}
