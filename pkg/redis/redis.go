package redis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ahrtolia/goboot/pkg/config"
	"github.com/google/wire"
	redislib "github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var ErrRedisDisabled = errors.New("redis client is disabled")

type Option struct {
	Enabled         bool          `mapstructure:"enabled"`
	Addr            string        `mapstructure:"addr"`
	Username        string        `mapstructure:"username"`
	Password        string        `mapstructure:"password"`
	DB              int           `mapstructure:"db"`
	MaxRetries      int           `mapstructure:"max_retries"`
	DialTimeout     time.Duration `mapstructure:"dial_timeout"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	PoolSize        int           `mapstructure:"pool_size"`
	MinIdleConns    int           `mapstructure:"min_idle_conns"`
	PoolTimeout     time.Duration `mapstructure:"pool_timeout"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime"`
	PingTimeout     time.Duration `mapstructure:"ping_timeout"`
}

func NewOption(cfg *config.ConfigManager) (*Option, error) {
	opt := &Option{
		Enabled:         cfg.GetViper().InConfig("redis"),
		Addr:            "127.0.0.1:6379",
		DB:              0,
		MaxRetries:      3,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolSize:        10,
		MinIdleConns:    2,
		PoolTimeout:     4 * time.Second,
		ConnMaxIdleTime: 5 * time.Minute,
		ConnMaxLifetime: 0,
		PingTimeout:     2 * time.Second,
	}

	v := cfg.GetViper()
	if redisCfg := v.Sub("redis"); redisCfg != nil {
		if err := redisCfg.Unmarshal(opt); err != nil {
			return nil, fmt.Errorf("failed to unmarshal redis options: %w", err)
		}
	}

	return opt, nil
}

type Client struct {
	mu         sync.RWMutex
	logger     *zap.Logger
	client     *redislib.Client
	currentCfg *Option
}

func NewClient(logger *zap.Logger, cfg *config.ConfigManager, opt *Option) (*Client, error) {
	c := &Client{
		logger: logger,
	}

	if err := c.applyConfig(opt); err != nil {
		return nil, err
	}

	if err := cfg.RegisterReloader("redis", c); err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Client) applyConfig(opt *Option) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !opt.Enabled {
		c.closeLocked()
		c.currentCfg = opt
		c.logger.Info("redis client disabled")
		return nil
	}

	newClient := redislib.NewClient(&redislib.Options{
		Addr:            opt.Addr,
		Username:        opt.Username,
		Password:        opt.Password,
		DB:              opt.DB,
		MaxRetries:      opt.MaxRetries,
		DialTimeout:     opt.DialTimeout,
		ReadTimeout:     opt.ReadTimeout,
		WriteTimeout:    opt.WriteTimeout,
		PoolSize:        opt.PoolSize,
		MinIdleConns:    opt.MinIdleConns,
		PoolTimeout:     opt.PoolTimeout,
		ConnMaxIdleTime: opt.ConnMaxIdleTime,
		ConnMaxLifetime: opt.ConnMaxLifetime,
	})

	ctx, cancel := context.WithTimeout(context.Background(), opt.PingTimeout)
	defer cancel()
	if err := newClient.Ping(ctx).Err(); err != nil {
		_ = newClient.Close()
		return fmt.Errorf("failed to connect redis: %w", err)
	}

	c.closeLocked()
	c.client = newClient
	c.currentCfg = opt

	c.logger.Info("redis client connected", zap.String("addr", opt.Addr), zap.Int("db", opt.DB))
	return nil
}

func (c *Client) ReloadConfig(v *viper.Viper) error {
	newOpt := &Option{
		Enabled:         v.InConfig("redis"),
		Addr:            "127.0.0.1:6379",
		DB:              0,
		MaxRetries:      3,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
		PoolSize:        10,
		MinIdleConns:    2,
		PoolTimeout:     4 * time.Second,
		ConnMaxIdleTime: 5 * time.Minute,
		ConnMaxLifetime: 0,
		PingTimeout:     2 * time.Second,
	}
	if redisCfg := v.Sub("redis"); redisCfg != nil {
		if err := redisCfg.Unmarshal(newOpt); err != nil {
			return fmt.Errorf("failed to unmarshal redis options: %w", err)
		}
	}

	if c.configEqual(newOpt) {
		c.logger.Debug("redis config unchanged, skip reload")
		return nil
	}

	return c.applyConfig(newOpt)
}

func (c *Client) configEqual(newOpt *Option) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.currentCfg == nil {
		return false
	}

	return c.currentCfg.Enabled == newOpt.Enabled &&
		c.currentCfg.Addr == newOpt.Addr &&
		c.currentCfg.Username == newOpt.Username &&
		c.currentCfg.Password == newOpt.Password &&
		c.currentCfg.DB == newOpt.DB &&
		c.currentCfg.MaxRetries == newOpt.MaxRetries &&
		c.currentCfg.DialTimeout == newOpt.DialTimeout &&
		c.currentCfg.ReadTimeout == newOpt.ReadTimeout &&
		c.currentCfg.WriteTimeout == newOpt.WriteTimeout &&
		c.currentCfg.PoolSize == newOpt.PoolSize &&
		c.currentCfg.MinIdleConns == newOpt.MinIdleConns &&
		c.currentCfg.PoolTimeout == newOpt.PoolTimeout &&
		c.currentCfg.ConnMaxIdleTime == newOpt.ConnMaxIdleTime &&
		c.currentCfg.ConnMaxLifetime == newOpt.ConnMaxLifetime &&
		c.currentCfg.PingTimeout == newOpt.PingTimeout
}

func (c *Client) Get() (*redislib.Client, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.client == nil {
		return nil, ErrRedisDisabled
	}
	return c.client, nil
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closeLocked()
}

func (c *Client) closeLocked() {
	if c.client != nil {
		_ = c.client.Close()
		c.client = nil
	}
}

var ProviderSet = wire.NewSet(NewOption, NewClient)
