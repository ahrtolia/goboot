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
	Logger *logger.Manager
	Http   *gin.Server
}

func New(
	cfg *config.ConfigManager,
	log *logger.Manager,
	httpSrv *gin.Server,
) (*App, error) {
	return &App{
		Config: cfg,
		Logger: log,
		Http:   httpSrv,
	}, nil
}

func (a *App) Start() error {

	if a.Http.GetHttpServer() != nil {
		go func() {
			err := a.Http.GetHttpServer().ListenAndServe()
			if err != nil {
				a.Logger.Logger().Fatal("http server start error", zap.Error(err))
			}
		}()
		a.Logger.Logger().Info("http server start", zap.String("address:", a.Http.GetHttpServer().Addr))
	}

	go func() {
		for {
			a.Logger.Logger().Info("info")
			a.Logger.Logger().Debug("debug")
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

		a.Logger.Logger().Info("receive a signal", zap.String("signal", s.String()))
		if a.Http.GetHttpServer() != nil {
			if err := a.Http.GetHttpServer().Shutdown(context.Background()); err != nil {
				a.Logger.Logger().Warn("stop http server error", zap.Error(err))
			}
		}

		os.Exit(0)
	}
}

var ProviderSet = wire.NewSet(
	wire.Struct(new(App), "*"), // 自动装配结构体字段
)
