// cmd/wire.go
//go:build wireinject
// +build wireinject

package main

import (
	app "github.com/ahrtolia/goboot/pkg"
	"github.com/ahrtolia/goboot/pkg/config"
	"github.com/ahrtolia/goboot/pkg/cron_starter"
	"github.com/ahrtolia/goboot/pkg/gin_starter"
	"github.com/ahrtolia/goboot/pkg/gorm_starter"
	"github.com/ahrtolia/goboot/pkg/logger"
	redispkg "github.com/ahrtolia/goboot/pkg/redis"

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
		gin_starter.ProviderSet,
	)

	dbSet = wire.NewSet(
		gorm_starter.ProviderSet,
	)

	cronSet = wire.NewSet(
		cron_starter.ProviderSet,
	)

	redisSet = wire.NewSet(
		redispkg.ProviderSet,
	)

	appSet = wire.NewSet(
		app.ProviderSet,
	)

	globalSet = wire.NewSet(
		configSet,
		loggerSet,
		httpSet,
		dbSet,
		cronSet,
		redisSet,
		appSet,
	)
)

func CreateApp(configFile string) (*app.App, error) {
	panic(wire.Build(
		globalSet,
	))
}
