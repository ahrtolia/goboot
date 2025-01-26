package app

import (
	"context"
	"github.com/google/wire"
	"go.uber.org/zap"
	"goboot/pkg/logger"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type App struct {
	name       string
	httpServer *http.Server
	logger     *logger.Manager
}

type Option struct {
}

func NewOption() *Option {
	return &Option{}
}

func New(
	name string,
	loggerManager *logger.Manager,
	httpServer *http.Server,
) (*App, error) {
	return &App{
		name:       name,
		httpServer: httpServer,
		logger:     loggerManager,
	}, nil
}

func (a *App) Start() error {

	if a.httpServer != nil {
		go func() {
			err := a.httpServer.ListenAndServe()
			if err != nil {
				a.logger.Logger().Fatal("http server start error", zap.Error(err))
			}
		}()
		a.logger.Logger().Info("http server start", zap.String("address:", a.httpServer.Addr))
	}

	go func() {
		for {
			a.logger.Logger().Info("info")
			a.logger.Logger().Debug("debug")
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

		a.logger.Logger().Info("receive a signal", zap.String("signal", s.String()))
		if a.httpServer != nil {
			if err := a.httpServer.Shutdown(context.Background()); err != nil {
				a.logger.Logger().Warn("stop http server error", zap.Error(err))
			}
		}

		os.Exit(0)
	}
}

var ProviderSet = wire.NewSet(New, NewOption)
