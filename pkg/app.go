package app

import (
	"context"
	"github.com/google/wire"
	"go.uber.org/zap"
	"goboot/pkg/config"
	"goboot/pkg/gin"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type App struct {
	Config *config.ConfigManager
	logger *zap.Logger
	Http   *gin.Server
}

func New(
	cfg *config.ConfigManager,
	logger *zap.Logger,
	httpSrv *gin.Server,
) (*App, error) {
	return &App{
		Config: cfg,
		logger: logger,
		Http:   httpSrv,
	}, nil
}

func (a *App) Start() error {

	a.logger.Info("HTTP server listening on", zap.String("addr", a.Http.GetHttpServer().Addr))

	go func() {
		for {
			a.logger.Info("server is running...")
			time.Sleep(2 * time.Second)
		}
	}()

	return nil
}

func (a *App) AwaitSignal() {
	c := make(chan os.Signal, 1)
	signal.Reset(syscall.SIGTERM, syscall.SIGINT)
	signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
	select {
	case s := <-c:
		a.logger.Info("starting graceful shutdown...", zap.String("signal", s.String()))
		if a.Http.GetHttpServer() != nil {
			if err := a.Http.GetHttpServer().Shutdown(context.Background()); err != nil {
				a.logger.Warn("stop http server error", zap.Error(err))
			}
		}

		os.Exit(0)
	}
}

var ProviderSet = wire.NewSet(
	New,
)
