package logger

import (
	"github.com/natefinch/lumberjack"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"goboot/pkg/config"
	"io"
	"os"
	"sync"
)

var (
	globalLogger *zap.Logger
	globalMu     sync.RWMutex
)

type Logger interface {
	Error(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Debug(msg string, fields ...zap.Field)
}

type Option struct {
	Level          string `mapstructure:"level"`
	Development    bool   `mapstructure:"development"`
	FileName       string `mapstructure:"file_name"`
	MaxSizeMB      int    `mapstructure:"max_size_mb"`
	MaxAgeDays     int    `mapstructure:"max_age_days"`
	Compress       bool   `mapstructure:"compress"`
	ConsoleEnabled bool   `mapstructure:"console_enabled"`
	FileEnabled    bool   `mapstructure:"file_enabled"`
}

type Manager struct {
	mu      sync.RWMutex
	option  *Option
	logger  *zap.Logger
	cleanup func() // 旧日志清理函数
}

func NewManager(cfg *config.ConfigManager) (*Manager, error) {
	m := &Manager{}

	// 初始加载配置
	if err := m.ReloadConfig(cfg.GetViper()); err != nil {
		return nil, err
	}

	// 注册配置监听
	if err := cfg.RegisterReloader("logger", m); err != nil {
		return nil, err
	}

	// 设置全局访问
	SetGlobalLogger(m.logger)

	return m, nil
}

func (m *Manager) ReloadConfig(v *viper.Viper) error {
	newOpt := loadOptions(v)

	// 创建新logger
	newLogger, cleanup, err := createLogger(newOpt)
	if err != nil {
		return err
	}

	// 替换旧实例
	m.mu.Lock()
	defer m.mu.Unlock()

	// 关闭旧logger
	if m.cleanup != nil {
		go m.cleanup()
	}

	m.option = newOpt
	m.logger = newLogger
	m.cleanup = cleanup

	// 更新全局logger
	SetGlobalLogger(newLogger)

	return nil
}

func (m *Manager) Logger() *zap.Logger {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.logger
}

func loadOptions(v *viper.Viper) *Option {
	opt := &Option{
		Level:       "info",
		Development: false,
	}

	sub := v.Sub("logger")
	if sub != nil {
		_ = sub.Unmarshal(opt)
	}
	return opt
}

func createLogger(opt *Option) (*zap.Logger, func(), error) {
	// 创建核心配置...
	// （保持原有create函数逻辑，返回logger和清理函数）

	// 示例实现：
	atomicLevel := zap.NewAtomicLevel()
	_ = atomicLevel.UnmarshalText([]byte(opt.Level))

	encoderConfig := zapcore.EncoderConfig{
		// 保持原有encoder配置
	}

	cores := make([]zapcore.Core, 0)

	// 控制台输出
	if opt.ConsoleEnabled {
		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.Lock(os.Stdout),
			atomicLevel,
		)
		cores = append(cores, consoleCore)
	}

	// 文件输出
	var fileSyncer zapcore.WriteSyncer
	if opt.FileEnabled {
		lj := &lumberjack.Logger{
			Filename: opt.FileName,
			MaxSize:  opt.MaxSizeMB,
			MaxAge:   opt.MaxAgeDays,
			Compress: opt.Compress,
		}
		fileSyncer = zapcore.AddSync(lj)
		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			fileSyncer,
			atomicLevel,
		)
		cores = append(cores, fileCore)
	}

	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	cleanup := func() {
		_ = logger.Sync()
		if fileSyncer != nil {
			if closer, ok := fileSyncer.(io.Closer); ok {
				_ = closer.Close()
			}
		}
	}

	return logger, cleanup, nil
}

// 全局访问方法
func L() *zap.Logger {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return globalLogger
}

func SetGlobalLogger(l *zap.Logger) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = l
}

func (m *Manager) Error(msg string, fields ...zap.Field) {
	m.logger.Error(msg, fields...)
}

func (m *Manager) Info(msg string, fields ...zap.Field) {
	m.logger.Info(msg, fields...)
}

func (m *Manager) Debug(msg string, fields ...zap.Field) {
	m.logger.Debug(msg, fields...)
}
