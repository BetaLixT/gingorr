package gingorr

import "go.uber.org/zap"

type IServiceProvider interface {
  GetLogger() *zap.Logger
}
