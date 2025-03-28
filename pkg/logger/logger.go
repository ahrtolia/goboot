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

func NewLogger(cfg *config.ConfigManager) (*zap.Logger, error) {
	opt := loadOptions(cfg.GetViper())

	logger, cleanup, err := createLogger(opt)
	if err != nil {
		return nil, err
	}

	defer cleanup()

	SetGlobalLogger(logger)

	// 注册动态配置监听
	_ = cfg.RegisterReloader("logger", config.ConfigReloaderFunc(func(v *viper.Viper) error {
		newOpt := loadOptions(v)
		newLogger, _, err := createLogger(newOpt)
		if err != nil {
			return err
		}
		SetGlobalLogger(newLogger)
		return nil
	}))

	return logger, nil
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
	atomicLevel := zap.NewAtomicLevel()
	_ = atomicLevel.UnmarshalText([]byte(opt.Level))

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	cores := make([]zapcore.Core, 0)

	if opt.ConsoleEnabled {
		consoleCore := zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			zapcore.Lock(os.Stdout),
			atomicLevel,
		)
		cores = append(cores, consoleCore)
	}

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
