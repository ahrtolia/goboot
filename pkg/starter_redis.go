package app

import (
	"context"

	"github.com/ahrtolia/goboot/pkg/config"
	redispkg "github.com/ahrtolia/goboot/pkg/redis"
)

type RedisStarter struct {
	cfg    *config.ConfigManager
	client *redispkg.Client
}

func NewRedisStarter(cfg *config.ConfigManager, client *redispkg.Client) *RedisStarter {
	return &RedisStarter{
		cfg:    cfg,
		client: client,
	}
}

func (s *RedisStarter) Name() string {
	return "redis"
}

func (s *RedisStarter) Enabled(ctx *Context) bool {
	return enabledByConfig(ctx, "", "redis", false)
}

func (s *RedisStarter) Init(ctx *Context) error {
	return nil
}

func (s *RedisStarter) Start(ctx *Context) error {
	return nil
}

func (s *RedisStarter) Stop(_ context.Context, _ *Context) error {
	if s.client == nil {
		return nil
	}
	s.client.Close()
	return nil
}
