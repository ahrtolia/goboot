package app

import (
	"context"
	"database/sql"
	"github.com/ahrtolia/goboot/pkg/config"
	"gorm.io/gorm"
)

type GormStarter struct {
	cfg *config.ConfigManager
	db  *gorm.DB
}

func NewGormStarter(cfg *config.ConfigManager, db *gorm.DB) *GormStarter {
	return &GormStarter{
		cfg: cfg,
		db:  db,
	}
}

func (s *GormStarter) Name() string {
	return "db"
}

func (s *GormStarter) Enabled(ctx *Context) bool {
	return enabledByConfig(ctx, "db.enabled", "db", false)
}

func (s *GormStarter) Init(ctx *Context) error {
	return nil
}

func (s *GormStarter) Start(ctx *Context) error {
	return nil
}

func (s *GormStarter) Stop(_ context.Context, _ *Context) error {
	if s.db == nil {
		return nil
	}
	sqlDB, err := s.db.DB()
	if err != nil || sqlDB == nil {
		return nil
	}
	return closeSQL(sqlDB)
}

func closeSQL(db *sql.DB) error {
	return db.Close()
}
