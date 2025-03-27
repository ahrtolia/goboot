// cmd/wire.go
//go:build wireinject
// +build wireinject

package main

import (
	app "goboot/pkg"
	"goboot/pkg/config"
	"goboot/pkg/gin"
	"goboot/pkg/gorm"
	"goboot/pkg/logger"

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
		wire.Struct(new(app.App), "*"),
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
