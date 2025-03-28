package app

import (
	"context"
	"github.com/google/wire"
	"go.uber.org/zap"
	"goboot/pkg/config"
	"goboot/pkg/gin"
	"goboot/pkg/logger"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type App struct {
	Config *config.ConfigManager
	Http   *gin.Server
}

func New(
	cfg *config.ConfigManager,
	httpSrv *gin.Server,
) (*App, error) {
	return &App{
		Config: cfg,
		Http:   httpSrv,
	}, nil
}

func (a *App) Start() error {

	logger.Info("HTTP server listening on", zap.String("addr", a.Http.GetHttpServer().Addr))

	go func() {
		for {
			logger.Debug("debug: server is running...")
			logger.Info("debug: server is running...")
			logger.Warn("debug: server is running...")
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
		logger.Info("starting graceful shutdown...", zap.String("signal", s.String()))
		if a.Http.GetHttpServer() != nil {
			if err := a.Http.GetHttpServer().Shutdown(context.Background()); err != nil {
				logger.Warn("stop http server error", zap.Error(err))
			}
		}

		os.Exit(0)
	}
}

var ProviderSet = wire.NewSet(
	New,
)
