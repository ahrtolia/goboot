package gin

import (
	"fmt"
	"github.com/Depado/ginprom"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"goboot/pkg/logger"
	"net/http"
	"time"
)

type InitControllers func(r *gin.Engine)

type Option struct {
	Port         int           `mapstructure:"port" default:"8080"`                        // 端口号，默认 8080
	Addr         string        `mapstructure:"addr" default:"0.0.0.0"`                     // 地址，默认 "0.0.0.0"
	LogFormat    string        `mapstructure:"log_format" default:"json"`                  // 日志格式，默认 "json"
	Debug        bool          `mapstructure:"debug" default:"false"`                      // 是否启用调试模式，默认 false
	GinMode      string        `mapstructure:"gin_mode" json:"gin_mode" default:"release"` // Gin 模式，默认 "release"
	ReadTimeout  time.Duration `mapstructure:"read_timeout" default:"10s"`                 // 请求超时时间，默认 10s
	WriteTimeout time.Duration `mapstructure:"write_timeout" default:"10s"`                // 响应超时时间，默认 10s
	IdleTimeout  time.Duration `mapstructure:"idle_timeout" default:"60s"`                 // 空闲超时时间，默认 60s
	MaxHeader    int           `mapstructure:"max_header" default:"1048576"`               // Header 最大字节数，默认 1048576 (1MB)
}

// NewOptions 从 viper 配置中加载 gin 服务配置
func NewOptions(logger *logger.Manager, viper *viper.Viper) *Option {
	option := &Option{
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

	httpConfig := viper.Sub("http")
	if httpConfig == nil {
		logger.Logger().Fatal("http configuration not found in yaml")
	}

	// 自动映射 viper 配置到 Options
	if err := httpConfig.Unmarshal(option); err != nil {
		logger.Logger().Fatal("failed to unmarshal http options", zap.Error(err))
	}

	return option
}

// New 创建一个 HTTP Server 并将配置应用到 gin 启动对象
func New(logger *logger.Manager, options *Option) *http.Server {
	// 设置 gin 模式
	gin.SetMode(options.GinMode)

	// 初始化 gin Router
	r := gin.Default()

	r.Use(gin.Recovery())
	r.Use(ginzap.RecoveryWithZap(logger.Logger(), true))
	p := ginprom.New(
		ginprom.Engine(r),
		ginprom.Subsystem("gin"),
		ginprom.Path("/metrics"),
	)
	r.Use(p.Instrument())

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"*"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		MaxAge: 12 * time.Hour,
	}))

	// 返回配置好的 HTTP Server
	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", options.Addr, options.Port),
		Handler:        r,
		ReadTimeout:    options.ReadTimeout,
		WriteTimeout:   options.WriteTimeout,
		IdleTimeout:    options.IdleTimeout,
		MaxHeaderBytes: options.MaxHeader,
	}

	pprof.Register(r)

	return s
}

// ProviderSet 用于 DI 的依赖提供者集合
var ProviderSet = wire.NewSet(New, NewOptions)
