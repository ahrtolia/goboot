package app

import (
	"context"
	"fmt"
	"github.com/ahrtolia/goboot/pkg/config"
	"github.com/google/wire"
	"go.uber.org/zap"
	"os"
	"os/signal"
	"syscall"
)

type App struct {
	Config   *config.ConfigManager
	ctx      *Context
	starters []Starter
}

func New(
	cfg *config.ConfigManager,
	ctx *Context,
	starters []Starter,
) (*App, error) {
	return &App{
		Config:   cfg,
		ctx:      ctx,
		starters: starters,
	}, nil
}

func (a *App) Start() error {
	for _, s := range a.starters {
		if !s.Enabled(a.ctx) {
			continue
		}
		if err := s.Init(a.ctx); err != nil {
			return fmt.Errorf("starter %s init failed: %w", s.Name(), err)
		}
	}

	for _, s := range a.starters {
		if !s.Enabled(a.ctx) {
			continue
		}
		if err := s.Start(a.ctx); err != nil {
			return fmt.Errorf("starter %s start failed: %w", s.Name(), err)
		}
	}
	return nil
}

func (a *App) AwaitSignal() {
	c := make(chan os.Signal, 1)
	signal.Reset(syscall.SIGTERM, syscall.SIGINT)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	select {
	case s := <-c:
		zap.L().Info("starting graceful shutdown...", zap.String("signal", s.String()))
		_ = a.Stop(context.Background())
		os.Exit(0)
	}
}

func (a *App) Stop(ctx context.Context) error {
	for i := len(a.starters) - 1; i >= 0; i-- {
		s := a.starters[i]
		if !s.Enabled(a.ctx) {
			continue
		}
		if err := s.Stop(ctx, a.ctx); err != nil {
			zap.L().Warn("starter stop failed", zap.String("starter", s.Name()), zap.Error(err))
		}
	}
	return nil
}

var ProviderSet = wire.NewSet(
	New,
	NewContext,
	NewLoggerStarter,
	NewHTTPStarter,
	NewGormStarter,
	NewCronStarter,
	NewRedisStarter,
	NewStarters,
)
