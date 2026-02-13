package app

import (
	"context"
	"github.com/ahrtolia/goboot/pkg/config"
	"github.com/ahrtolia/goboot/pkg/gin"
)

type HTTPStarter struct {
	cfg    *config.ConfigManager
	server *gin.Server
}

func NewHTTPStarter(cfg *config.ConfigManager, server *gin.Server) *HTTPStarter {
	return &HTTPStarter{
		cfg:    cfg,
		server: server,
	}
}

func (s *HTTPStarter) Name() string {
	return "http"
}

func (s *HTTPStarter) Enabled(ctx *Context) bool {
	return enabledByConfig(ctx, "http.enabled", "http", false)
}

func (s *HTTPStarter) Init(ctx *Context) error {
	return nil
}

func (s *HTTPStarter) Start(ctx *Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Start()
}

func (s *HTTPStarter) Stop(_ context.Context, _ *Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Close()
}
