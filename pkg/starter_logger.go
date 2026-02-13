package app

import (
	"context"
	"github.com/ahrtolia/goboot/pkg/config"
	"github.com/ahrtolia/goboot/pkg/logger"
	"go.uber.org/zap"
)

type LoggerStarter struct {
	cfg    *config.ConfigManager
	logger *zap.Logger
}

func NewLoggerStarter(cfg *config.ConfigManager, loggerInstance *zap.Logger) *LoggerStarter {
	return &LoggerStarter{
		cfg:    cfg,
		logger: loggerInstance,
	}
}

func (s *LoggerStarter) Name() string {
	return "logger"
}

func (s *LoggerStarter) Enabled(ctx *Context) bool {
	return enabledByConfig(ctx, "logger.enabled", "logger", false)
}

func (s *LoggerStarter) Init(ctx *Context) error {
	return nil
}

func (s *LoggerStarter) Start(ctx *Context) error {
	return nil
}

func (s *LoggerStarter) Stop(_ context.Context, _ *Context) error {
	logger.Close()
	return nil
}
