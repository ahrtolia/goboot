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

type ConfigReloaderFunc func(*viper.Viper) error

func (f ConfigReloaderFunc) ReloadConfig(v *viper.Viper) error {
	return f(v)
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

	cm := &ConfigManager{
		options:   opt,
		v:         viper.New(),
		reloaders: make(map[string]ConfigReloader),
		adapters:  make(map[string]ConfigCenter),
	}

	// 注册 Nacos 适配器
	cm.RegisterAdapter(NewNacosAdapter())

	localConfigErr := cm.initLocal(string(opt.ConfigFile)) // 从本地文件加载
	if localConfigErr != nil {
		fmt.Println(localConfigErr)
	}

	configCenterErr := cm.initConfigCenter() // 激活 Nacos 并 merge 配置
	if configCenterErr != nil {
		fmt.Println(configCenterErr)
	}

	return cm
}

func (cm *ConfigManager) initLocal(configFile string) error {
	cm.v.SetConfigFile(configFile)

	err := cm.v.ReadInConfig()
	if err != nil {
		// 改成 warn 模式，允许 fallback 到远程配置
		fmt.Println("[Config] Failed to load local config:", err)
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
	currentConfig := cm.v.AllSettings()
	cm.mu.RUnlock()

	newViper := viper.New()
	for k, v := range currentConfig {
		newViper.Set(k, v)
	}

	cm.mu.Lock()
	cm.v = newViper
	cm.mu.Unlock()

	for name, reloader := range cm.reloaders {
		go func(n string, r ConfigReloader) {
			if err := r.ReloadConfig(newViper); err != nil {
				fmt.Printf("[Config] Reloader [%s] failed: %v\n", n, err)
			} else {
				fmt.Printf("[Config] Reloader [%s] success\n", n)
			}
		}(name, reloader)
	}
}

func (cm *ConfigManager) ReloadConfig(newViper *viper.Viper) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.v = newViper
	return nil
}

func (cm *ConfigManager) GetViper() *viper.Viper {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.v
}
