package logger

import (
	"fmt"
	"github.com/ahrtolia/goboot/pkg/config"
	"github.com/natefinch/lumberjack"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"io"
	"os"
	"sync"
)

var (
	globalLogger  *zap.Logger
	globalMu      sync.RWMutex
	globalCleanup func()
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

	SetGlobalLogger(logger, cleanup)
	zap.ReplaceGlobals(logger)

	// 注册动态配置监听
	_ = cfg.RegisterReloader("logger", config.ConfigReloaderFunc(func(v *viper.Viper) error {
		newOpt := loadOptions(v)
		newLogger, newCleanup, err := createLogger(newOpt)
		if err != nil {
			fmt.Printf("failed to create new logger: %v\n", err)
		}

		SetGlobalLogger(newLogger, newCleanup)
		zap.ReplaceGlobals(newLogger)
		return nil
	}))

	return logger, nil
}

func loadOptions(v *viper.Viper) *Option {
	opt := &Option{
		Level:       "info",
		Development: false,
	}
	_ = v.UnmarshalKey("logger", opt)
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

func SetGlobalLogger(l *zap.Logger, cleanup func()) {
	globalMu.Lock()
	defer globalMu.Unlock()

	// 先关闭旧的
	if globalCleanup != nil {
		globalCleanup()
	}

	globalLogger = l
	globalCleanup = cleanup
}

func Debug(msg string, fields ...zap.Field) {
	L().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	L().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	L().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	L().Error(msg, fields...)
}
