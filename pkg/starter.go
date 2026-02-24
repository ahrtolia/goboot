package app

import (
	"context"
	"github.com/ahrtolia/goboot/pkg/config"
	"github.com/ahrtolia/goboot/pkg/cron_starter"
	"github.com/ahrtolia/goboot/pkg/gin_starter"
	redispkg "github.com/ahrtolia/goboot/pkg/redis"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Context struct {
	Config *config.ConfigManager
	Logger *zap.Logger
	HTTP   *gin_starter.Server
	DB     *gorm.DB
	Cron   *cron_starter.Scheduler
	Redis  *redispkg.Client
}

func NewContext(cfg *config.ConfigManager, logger *zap.Logger, httpSrv *gin_starter.Server, db *gorm.DB, cronScheduler *cron_starter.Scheduler, redisClient *redispkg.Client) *Context {
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
	cronStarter *CronStarter,
	redisStarter *RedisStarter,
) []Starter {
	return []Starter{
		loggerStarter,
		httpStarter,
		gormStarter,
	}
}
