package cron

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ahrtolia/goboot/pkg/config"
	"github.com/google/wire"
	"github.com/robfig/cron/v3"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var ErrCronDisabled = errors.New("cron scheduler is disabled")

type Option struct {
	Enabled     bool          `mapstructure:"enabled"`
	Location    string        `mapstructure:"location"`
	WithSeconds bool          `mapstructure:"with_seconds"`
	StopTimeout time.Duration `mapstructure:"stop_timeout"`
}

func NewOption(cfg *config.ConfigManager) (*Option, error) {
	opt := &Option{
		Enabled:     cfg.GetViper().InConfig("cron"),
		Location:    "Local",
		WithSeconds: false,
		StopTimeout: 5 * time.Second,
	}

	v := cfg.GetViper()
	if cronCfg := v.Sub("cron"); cronCfg != nil {
		if err := cronCfg.Unmarshal(opt); err != nil {
			return nil, fmt.Errorf("failed to unmarshal cron options: %w", err)
		}
	}
	return opt, nil
}

type Scheduler struct {
	mu         sync.RWMutex
	logger     *zap.Logger
	cron       *cron.Cron
	currentCfg *Option
}

func NewScheduler(logger *zap.Logger, cfg *config.ConfigManager, opt *Option) (*Scheduler, error) {
	s := &Scheduler{
		logger: logger,
	}

	if err := s.applyConfig(opt); err != nil {
		return nil, err
	}

	if err := cfg.RegisterReloader("cron", s); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Scheduler) applyConfig(opt *Option) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !opt.Enabled {
		s.stopLocked(opt.StopTimeout)
		s.currentCfg = opt
		return nil
	}

	loc, err := time.LoadLocation(opt.Location)
	if err != nil {
		return fmt.Errorf("invalid cron location: %w", err)
	}

	options := []cron.Option{cron.WithLocation(loc)}
	if opt.WithSeconds {
		options = append(options, cron.WithSeconds())
	}

	newCron := cron.New(options...)
	newCron.Start()

	s.stopLocked(opt.StopTimeout)
	s.cron = newCron
	s.currentCfg = opt

	s.logger.Info("cron scheduler started", zap.Bool("with_seconds", opt.WithSeconds), zap.String("location", opt.Location))
	return nil
}

func (s *Scheduler) stopLocked(timeout time.Duration) {
	if s.cron == nil {
		return
	}

	ctx := context.Background()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	stopCtx := s.cron.Stop()
	select {
	case <-stopCtx.Done():
	case <-ctx.Done():
	}

	s.cron = nil
}

func (s *Scheduler) ReloadConfig(v *viper.Viper) error {
	newOpt := &Option{
		Enabled:     v.InConfig("cron"),
		Location:    "Local",
		WithSeconds: false,
		StopTimeout: 5 * time.Second,
	}
	if cronCfg := v.Sub("cron"); cronCfg != nil {
		if err := cronCfg.Unmarshal(newOpt); err != nil {
			return fmt.Errorf("failed to unmarshal cron options: %w", err)
		}
	}

	if s.configEqual(newOpt) {
		s.logger.Debug("cron config unchanged, skip reload")
		return nil
	}

	return s.applyConfig(newOpt)
}

func (s *Scheduler) configEqual(newOpt *Option) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.currentCfg == nil {
		return false
	}

	return s.currentCfg.Enabled == newOpt.Enabled &&
		s.currentCfg.Location == newOpt.Location &&
		s.currentCfg.WithSeconds == newOpt.WithSeconds &&
		s.currentCfg.StopTimeout == newOpt.StopTimeout
}

func (s *Scheduler) AddFunc(spec string, cmd func()) (cron.EntryID, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.cron == nil {
		return 0, ErrCronDisabled
	}
	return s.cron.AddFunc(spec, cmd)
}

func (s *Scheduler) AddJob(spec string, job cron.Job) (cron.EntryID, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.cron == nil {
		return 0, ErrCronDisabled
	}
	return s.cron.AddJob(spec, job)
}

func (s *Scheduler) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stopLocked(5 * time.Second)
}

var ProviderSet = wire.NewSet(NewOption, NewScheduler)
