package app

import (
	"context"

	"github.com/ahrtolia/goboot/pkg/config"
	"github.com/ahrtolia/goboot/pkg/cron_starter"
)

type CronStarter struct {
	cfg       *config.ConfigManager
	scheduler *cron_starter.Scheduler
}

func NewCronStarter(cfg *config.ConfigManager, scheduler *cron_starter.Scheduler) *CronStarter {
	return &CronStarter{
		cfg:       cfg,
		scheduler: scheduler,
	}
}

func (s *CronStarter) Name() string {
	return "cron_starter"
}

func (s *CronStarter) Enabled(ctx *Context) bool {
	return enabledByConfig(ctx, "", "cron_starter", false)
}

func (s *CronStarter) Init(ctx *Context) error {
	return nil
}

func (s *CronStarter) Start(ctx *Context) error {
	return nil
}

func (s *CronStarter) Stop(_ context.Context, _ *Context) error {
	if s.scheduler == nil {
		return nil
	}
	s.scheduler.Close()
	return nil
}
