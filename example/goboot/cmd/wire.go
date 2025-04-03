// cmd/wire.go
//go:build wireinject
// +build wireinject

package main

import (
	app "github.com/ahrtolia/goboot/pkg"
	"github.com/ahrtolia/goboot/pkg/config"
	"github.com/ahrtolia/goboot/pkg/gin"
	"github.com/ahrtolia/goboot/pkg/gorm"
	"github.com/ahrtolia/goboot/pkg/logger"

	"github.com/google/wire"
)

var (
	configSet = wire.NewSet(
		config.ProviderSet,
		config.NacosProvider,
	)

	loggerSet = wire.NewSet(
		logger.ProviderSet,
	)

	httpSet = wire.NewSet(
		gin.ProviderSet,
	)

	dbSet = wire.NewSet(
		gorm.ProviderSet,
	)

	appSet = wire.NewSet(
		app.ProviderSet,
	)

	globalSet = wire.NewSet(
		configSet,
		loggerSet,
		httpSet,
		dbSet,
		appSet,
	)
)

func CreateApp(configFile string) (*app.App, error) {
	panic(wire.Build(
		globalSet,
	))
}
