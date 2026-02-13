package app

import (
	"context"
	"github.com/ahrtolia/goboot/pkg/config"
	"github.com/ahrtolia/goboot/pkg/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Context struct {
	Config *config.ConfigManager
	Logger *zap.Logger
	HTTP   *gin.Server
	DB     *gorm.DB
}

func NewContext(cfg *config.ConfigManager, logger *zap.Logger, httpSrv *gin.Server, db *gorm.DB) *Context {
	return &Context{
		Config: cfg,
		Logger: logger,
		HTTP:   httpSrv,
		DB:     db,
	}
}

type Starter interface {
	Name() string
	Enabled(ctx *Context) bool
	Init(ctx *Context) error
	Start(ctx *Context) error
	Stop(ctx context.Context, appCtx *Context) error
}

func NewStarters(
	loggerStarter *LoggerStarter,
	httpStarter *HTTPStarter,
	gormStarter *GormStarter,
) []Starter {
	return []Starter{
		loggerStarter,
		httpStarter,
		gormStarter,
	}
}
