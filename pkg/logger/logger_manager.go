package logger

import (
	"github.com/fsnotify/fsnotify"
	"github.com/google/wire"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"log"

	"sync"
)

// Manager 用于统一管理 Logger 实例，并在配置更新时热更新

type Manager struct {
	v      *viper.Viper
	mu     sync.Mutex
	option *Option
	logger *zap.Logger
}

// NewLoggerManager 根据 viper 配置初始化一个 Manager
func NewLoggerManager(v *viper.Viper) *Manager {

	// 初始创建一次 logger
	opt := loadOptionFromViper(v)

	l := create(opt)
	manager := &Manager{
		v:      v,
		option: opt,
		logger: l,
	}

	// 监听配置变化（包括本地文件和可能来自 Nacos 的更新）
	manager.watchConfig()

	// 启动时先输出一次
	manager.logger.Info("Manager 初始化完成",
		zap.String("level", manager.option.Level),
		zap.Bool("development", manager.option.Development),
	)

	return manager
}

// Logger 返回当前最新的 Logger 实例
func (m *Manager) Logger() *zap.Logger {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.logger
}

// 修改watchConfig方法，增加防抖机制和变更检测
func (m *Manager) watchConfig() {

	m.v.WatchConfig()
	m.v.OnConfigChange(func(e fsnotify.Event) {

		log.Printf("配置文件 [%s] 发生变化，正在更新 Logger...", e.Name)
		newOpt := loadOptionFromViper(m.v)

		m.mu.Lock()
		defer m.mu.Unlock()

		newLogger := create(newOpt)
		oldLogger := m.logger

		// 原子切换
		m.option = newOpt
		m.logger = newLogger

		// 异步关闭旧logger（避免阻塞）
		go func() {
			if err := oldLogger.Sync(); err != nil {
				return
			}
		}()

		newLogger.Info("Logger配置已热更新",
			zap.String("level", newOpt.Level),
		)
	})
}

// loadOptionFromViper 用于从 Viper 中提取或反序列化出 Option
func loadOptionFromViper(v *viper.Viper) *Option {

	logConfig := v.Sub("logger")

	if logConfig == nil {

		log.Fatal("logger configuration not found in config")

	}
	opt := &Option{}
	if err := logConfig.Unmarshal(opt); err != nil {
		log.Fatal("failed to unmarshal logger options", err)
	}
	return opt
}

var ProviderSet = wire.NewSet(NewLoggerManager)
