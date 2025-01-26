//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	app "goboot/pkg"
	"goboot/pkg/config/nacos"
	"goboot/pkg/gin"
	"goboot/pkg/gorm"
	"goboot/pkg/logger"
)

var providerSet = wire.NewSet(
	//config.ProviderSet,
	nacos.ProviderSet,
	logger.ProviderSet,
	gin.ProviderSet,
	app.ProviderSet,
	gorm.ProviderSet,
)

func CreateApp(cf string) (*app.App, error) {
	panic(wire.Build(providerSet))
}
