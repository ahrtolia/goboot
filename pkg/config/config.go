// pkg/config/config.go
package config

import (
	"errors"
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/google/wire"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	ErrConfigCenterNotFound = errors.New("config center not found")
	ErrConfigReloaderExists = errors.New("config reloader already registered")
)

type Logger interface {
	Error(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
}

type ConfigReloader interface {
	ReloadConfig(newViper *viper.Viper) error
}

type ConfigCenter interface {
	Name() string
	Init(v *viper.Viper) error
	Watch(v *viper.Viper, onChange func()) error
	Close()
}

type Options struct {
	ConfigFile   string
	ConfigCenter string
}

type ConfigManager struct {
	options      Options
	v            *viper.Viper
	mu           sync.RWMutex
	reloaders    map[string]ConfigReloader
	configCenter ConfigCenter
	adapters     map[string]ConfigCenter
	logger       Logger
}

func NewConfigManager(logger Logger, opts Options) (*ConfigManager, error) {
	cm := &ConfigManager{
		options:   opts,
		v:         viper.New(),
		reloaders: make(map[string]ConfigReloader),
		adapters:  make(map[string]ConfigCenter),
		logger:    logger,
	}

	// Initialize local config
	if opts.ConfigFile != "" {
		if err := cm.initLocal(opts.ConfigFile); err != nil {
			return nil, fmt.Errorf("failed to init local config: %w", err)
		}
	}

	// Initialize config center
	if opts.ConfigCenter != "" {
		if err := cm.initConfigCenter(); err != nil {
			return nil, fmt.Errorf("failed to init config center: %w", err)
		}
	}

	return cm, nil
}

func (cm *ConfigManager) initLocal(configFile string) error {
	cm.v.SetConfigFile(configFile)
	if err := cm.v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read local config: %w", err)
	}

	cm.v.WatchConfig()
	cm.v.OnConfigChange(func(e fsnotify.Event) {
		cm.logger.Info("Config file changed, triggering reload",
			zap.String("file", e.Name))
		cm.fireReload()
	})

	return nil
}

func (cm *ConfigManager) initConfigCenter() error {
	centerConfig := cm.v.Sub("config_center." + cm.options.ConfigCenter)
	if centerConfig == nil {
		return fmt.Errorf("config center [%s] not configured", cm.options.ConfigCenter)
	}

	return cm.ActivateConfigCenter(cm.options.ConfigCenter)
}

func (cm *ConfigManager) RegisterAdapter(adapter ConfigCenter) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.adapters[adapter.Name()] = adapter
}

func (cm *ConfigManager) ActivateConfigCenter(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	adapter, exists := cm.adapters[name]
	if !exists {
		return ErrConfigCenterNotFound
	}

	if cm.configCenter != nil {
		cm.configCenter.Close()
	}

	if err := adapter.Init(cm.v); err != nil {
		return fmt.Errorf("failed to init config center: %w", err)
	}

	if err := adapter.Watch(cm.v, cm.fireReload); err != nil {
		return fmt.Errorf("failed to watch config center: %w", err)
	}

	cm.configCenter = adapter
	return nil
}

func (cm *ConfigManager) RegisterReloader(name string, reloader ConfigReloader) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.reloaders[name]; exists {
		return ErrConfigReloaderExists
	}

	cm.reloaders[name] = reloader
	return nil
}

func (cm *ConfigManager) fireReload() {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	currentConfig := cm.v.AllSettings()
	newViper := viper.New()
	for k, v := range currentConfig {
		newViper.Set(k, v)
	}

	for name, reloader := range cm.reloaders {
		go func(n string, r ConfigReloader) {
			if err := r.ReloadConfig(newViper); err != nil {
				cm.logger.Error("Failed to reload component",
					zap.String("component", n),
					zap.Error(err))
			}
		}(name, reloader)
	}
}

func (cm *ConfigManager) ReloadConfig(newViper *viper.Viper) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if err := cm.v.MergeConfigMap(newViper.AllSettings()); err != nil {
		return fmt.Errorf("failed to merge config: %w", err)
	}
	return nil
}

func (cm *ConfigManager) GetViper() *viper.Viper {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.v
}

var ProviderSet = wire.NewSet(
	wire.Struct(new(Options), "*"),
	NewConfigManager,
	wire.Bind(new(ConfigReloader), new(*ConfigManager)),
)
