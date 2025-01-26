package logger

import (
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
)

// Option 存放常见的日志配置参数

type Option struct {
	Level          string `json:"level,omitempty" mapstructure:"level" yaml:"level"`
	Development    bool   `json:"development,omitempty" mapstructure:"development" yaml:"development"`
	FileName       string `json:"file_name,omitempty" mapstructure:"file_name" yaml:"file_name"`
	MaxSizeMB      int    `json:"max_size_mb,omitempty" mapstructure:"max_size_mb" yaml:"max_size_mb"`
	MaxAgeDays     int    `json:"max_age_days,omitempty" mapstructure:"max_age_days" yaml:"max_age_days"`
	Compress       bool   `json:"compress,omitempty" mapstructure:"compress" yaml:"compress"`
	ConsoleEnabled bool   `json:"console_enabled,omitempty" mapstructure:"console_enabled" yaml:"console_enabled"`
	FileEnabled    bool   `json:"file_enabled,omitempty" mapstructure:"file_enabled" yaml:"file_enabled"`
}

// create 根据配置生成具体的 zap.Logger

func create(option *Option) *zap.Logger {

	atomicLevel := zap.NewAtomicLevel()

	switch option.Level {
	case "debug":
		atomicLevel.SetLevel(zap.DebugLevel)
	case "info":
		atomicLevel.SetLevel(zap.InfoLevel)
	case "warn":
		atomicLevel.SetLevel(zap.WarnLevel)
	case "error":
		atomicLevel.SetLevel(zap.ErrorLevel)
	default:
		atomicLevel.SetLevel(zap.InfoLevel)
	}
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var cores []zapcore.Core

	if option.ConsoleEnabled {
		consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
		consoleCore := zapcore.NewCore(
			consoleEncoder,
			zapcore.AddSync(os.Stdout),
			atomicLevel,
		)
		cores = append(cores, consoleCore)
	}

	if option.FileEnabled {
		fileName := option.FileName
		if fileName == "" {
			fileName = "app.log"
		}
		fileSyncer := zapcore.AddSync(&lumberjack.Logger{
			Filename:   fileName,
			MaxSize:    option.MaxSizeMB,
			MaxAge:     option.MaxAgeDays,
			Compress:   option.Compress,
			MaxBackups: 0,
		})
		jsonEncoder := zapcore.NewJSONEncoder(encoderConfig)
		fileCore := zapcore.NewCore(
			jsonEncoder,
			fileSyncer,
			atomicLevel,
		)
		cores = append(cores, fileCore)
	}

	core := zapcore.NewTee(cores...)
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

	return logger
}
