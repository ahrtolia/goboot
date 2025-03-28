package gin

import (
	"context"
	"fmt"
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
	"goboot/pkg/config"
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
}

func NewOption(cfg *config.ConfigManager) (*Option, error) {
	v := cfg.GetViper()
	opt := &Option{
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
	s.mu.Lock()
	defer s.mu.Unlock()

	gin.SetMode(opt.GinMode)

	// 创建新router实例
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(ginzap.RecoveryWithZap(s.logger, true))

	// 配置中间件
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
		promInstance.Use(router) // 将 Prometheus 挂载到新的 router
	}
	// 注册pprof
	pprof.Register(router)

	// 创建新server实例
	newServer := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", opt.Addr, opt.Port),
		Handler:        router,
		ReadTimeout:    opt.ReadTimeout,
		WriteTimeout:   opt.WriteTimeout,
		IdleTimeout:    opt.IdleTimeout,
		MaxHeaderBytes: opt.MaxHeader,
	}

	// 启动新服务器
	go func() {
		s.logger.Info("Starting HTTP server", zap.String("addr", newServer.Addr))
		if err := newServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("HTTP server failed to start", zap.Error(err))
		}
	}()

	// 关闭旧服务器
	if s.server != nil {
		go s.gracefulShutdown(s.server)
	}

	s.server = newServer
	s.router = router
	s.currentCfg = opt

	return nil
}

func (s *Server) ReloadConfig(v *viper.Viper) error {
	newOpt := &Option{}
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

	return s.applyConfig(newOpt)
}

func (s *Server) configEqual(newOpt *Option) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.currentCfg.Port == newOpt.Port &&
		s.currentCfg.Addr == newOpt.Addr &&
		s.currentCfg.ReadTimeout == newOpt.ReadTimeout &&
		s.currentCfg.WriteTimeout == newOpt.WriteTimeout &&
		s.currentCfg.IdleTimeout == newOpt.IdleTimeout &&
		s.currentCfg.MaxHeader == newOpt.MaxHeader
}

func (s *Server) gracefulShutdown(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		s.logger.Error("HTTP server shutdown error", zap.Error(err))
	} else {
		s.logger.Info("HTTP server stopped gracefully")
	}
}

func (s *Server) GetRouter() *gin.Engine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.router
}

func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.server != nil {
		go s.gracefulShutdown(s.server)
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

// Wire Provider Set
var ProviderSet = wire.NewSet(
	NewServer,
	NewOption,
)
