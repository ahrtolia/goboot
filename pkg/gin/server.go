package gin

import (
	"context"
	"fmt"
	"github.com/ahrtolia/goboot/pkg/config"
	"net/http"
	"sync"
	"time"

	"github.com/Depado/ginprom"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	promInitOnce sync.Once
	promInstance *ginprom.Prometheus
)

type Option struct {
	Port         int           `mapstructure:"port"`
	Addr         string        `mapstructure:"addr"`
	LogFormat    string        `mapstructure:"log_format"`
	Debug        bool          `mapstructure:"debug"`
	GinMode      string        `mapstructure:"gin_mode"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout"`
	MaxHeader    int           `mapstructure:"max_header"`
}

type Server struct {
	mu         sync.RWMutex
	server     *http.Server
	router     *gin.Engine
	currentCfg *Option
	logger     *zap.Logger
	cleanup    func() // 旧服务器清理函数
	started    bool
}

func NewOption(cfg *config.ConfigManager) (*Option, error) {
	v := cfg.GetViper()
	opt := defaultOption()

	httpConfig := v.Sub("http")
	if httpConfig != nil {
		if err := httpConfig.Unmarshal(opt); err != nil {
			return nil, fmt.Errorf("failed to unmarshal http options: %w", err)
		}
	}
	return opt, nil
}

func NewServer(
	logger *zap.Logger,
	cfg *config.ConfigManager,
	opt *Option,
) (*Server, error) {
	s := &Server{
		logger: logger,
	}

	// 初始创建服务器
	if err := s.applyConfig(opt); err != nil {
		return nil, err
	}

	// 注册配置监听
	if err := cfg.RegisterReloader("http", s); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Server) applyConfig(opt *Option) error {
	gin.SetMode(opt.GinMode)

	router := s.buildRouter()
	newServer := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", opt.Addr, opt.Port),
		Handler:        router,
		ReadTimeout:    opt.ReadTimeout,
		WriteTimeout:   opt.WriteTimeout,
		IdleTimeout:    opt.IdleTimeout,
		MaxHeaderBytes: opt.MaxHeader,
	}

	s.mu.Lock()
	s.server = newServer
	s.router = router
	s.currentCfg = opt
	s.mu.Unlock()

	return nil
}

func (s *Server) buildRouter() *gin.Engine {
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(ginzap.RecoveryWithZap(s.logger, true))

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"*"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	promInitOnce.Do(func() {
		promInstance = ginprom.New(
			ginprom.Subsystem("gin"),
			ginprom.Path("/metrics"),
		)
	})

	if promInstance != nil {
		promInstance.Use(router)
	}
	pprof.Register(router)

	return router
}

func (s *Server) ReloadConfig(v *viper.Viper) error {
	newOpt := defaultOption()
	if httpConfig := v.Sub("http"); httpConfig != nil {
		if err := httpConfig.Unmarshal(newOpt); err != nil {
			return fmt.Errorf("failed to unmarshal http options: %w", err)
		}
	}

	// 比较配置差异
	if s.configEqual(newOpt) {
		s.logger.Debug("HTTP config unchanged, skip reload")
		return nil
	}

	s.mu.RLock()
	wasStarted := s.started
	oldServer := s.server
	s.mu.RUnlock()

	if err := s.applyConfig(newOpt); err != nil {
		return err
	}

	if wasStarted {
		if oldServer != nil {
			_ = s.gracefulShutdown(oldServer)
		}
		s.mu.RLock()
		newServer := s.server
		s.mu.RUnlock()
		s.startServer(newServer)
	}

	return nil
}

func (s *Server) configEqual(newOpt *Option) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.currentCfg == nil {
		return false
	}

	return s.currentCfg.Port == newOpt.Port &&
		s.currentCfg.Addr == newOpt.Addr &&
		s.currentCfg.LogFormat == newOpt.LogFormat &&
		s.currentCfg.Debug == newOpt.Debug &&
		s.currentCfg.GinMode == newOpt.GinMode &&
		s.currentCfg.ReadTimeout == newOpt.ReadTimeout &&
		s.currentCfg.WriteTimeout == newOpt.WriteTimeout &&
		s.currentCfg.IdleTimeout == newOpt.IdleTimeout &&
		s.currentCfg.MaxHeader == newOpt.MaxHeader
}

func (s *Server) gracefulShutdown(server *http.Server) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP server shutdown error", zap.Error(err))
		return err
	}
	s.logger.Info("HTTP server stopped gracefully")
	return nil
}

func (s *Server) GetRouter() *gin.Engine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.router
}

func (s *Server) Start() error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}
	if s.server == nil {
		s.mu.Unlock()
		return fmt.Errorf("http server not initialized")
	}
	server := s.server
	s.started = true
	s.mu.Unlock()

	s.startServer(server)
	return nil
}

func (s *Server) Close() error {
	s.mu.Lock()
	server := s.server
	s.started = false
	s.mu.Unlock()

	if server != nil {
		go func() {
			_ = s.gracefulShutdown(server)
		}()
	}

	if s.cleanup != nil {
		s.cleanup()
	}

	return nil
}

func (s *Server) GetHttpServer() *http.Server {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.server
}

func (s *Server) startServer(server *http.Server) {
	if server == nil {
		return
	}
	go func() {
		s.logger.Info("Starting HTTP server", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("HTTP server failed to start", zap.Error(err))
		}
	}()
}

func defaultOption() *Option {
	return &Option{
		Port:         8080,
		Addr:         "0.0.0.0",
		LogFormat:    "json",
		Debug:        false,
		GinMode:      "release",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
		MaxHeader:    1048576,
	}
}

// Wire Provider Set
var ProviderSet = wire.NewSet(
	NewServer,
	NewOption,
)
