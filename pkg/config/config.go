// pkg/config/config.go
package config

import (
	"errors"
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var (
	ErrConfigCenterNotFound = errors.New("config center not found")
	ErrConfigReloaderExists = errors.New("config reloader already registered")
)

type ConfigReloader interface {
	ReloadConfig(newViper *viper.Viper) error
}

type ConfigCenter interface {
	Name() string
	Init(v *viper.Viper) error
	Watch(v *viper.Viper, onChange func()) error
	Close()
}

type ConfigFile string
type ConfigCenterType string

type Options struct {
	ConfigFile   ConfigFile
	ConfigCenter ConfigCenterType
}

func NewOptions() Options {
	return Options{
		ConfigFile:   ConfigFile("config.yaml"), // 你可以改成读取 ENV 或默认值
		ConfigCenter: ConfigCenterType("nacos"),
	}
}

type ConfigManager struct {
	options      Options
	v            *viper.Viper
	mu           sync.RWMutex
	reloaders    map[string]ConfigReloader
	configCenter ConfigCenter
	adapters     map[string]ConfigCenter
}

func NewConfigManager(opt Options) *ConfigManager {
	return &ConfigManager{
		options:   opt,
		v:         viper.New(),
		reloaders: make(map[string]ConfigReloader),
		adapters:  make(map[string]ConfigCenter),
	}
}

func (cm *ConfigManager) initLocal(configFile string) error {
	cm.v.SetConfigFile(configFile)
	if err := cm.v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read local config: %w", err)
	}

	cm.v.WatchConfig()
	cm.v.OnConfigChange(func(e fsnotify.Event) {
		cm.fireReload()
	})

	return nil
}

func (cm *ConfigManager) initConfigCenter() error {
	centerConfig := cm.v.Sub(string("config_center." + cm.options.ConfigCenter))
	if centerConfig == nil {
		return fmt.Errorf("config center [%s] not configured", cm.options.ConfigCenter)
	}

	return cm.ActivateConfigCenter(string(cm.options.ConfigCenter))
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
