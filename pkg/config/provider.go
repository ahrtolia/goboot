package config

import "github.com/google/wire"

var ProviderSet = wire.NewSet(
	InitConfigManager,
	NewOptions,
)

var NacosProvider = wire.NewSet(
	NewNacosAdapter,
)
