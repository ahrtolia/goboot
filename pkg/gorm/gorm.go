package gorm

import (
	"fmt"
	"github.com/ahrtolia/goboot/pkg/config"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"log"
	"time"
)

type Option struct {
	DbHost              string        `mapstructure:"db_host" default:"localhost"`            // 数据库主机地址，默认 "localhost"
	DbPort              int           `mapstructure:"db_port" default:"3306"`                 // 数据库端口，默认 3306
	DbUser              string        `mapstructure:"db_user"`                                // 数据库用户名
	DbPassword          string        `mapstructure:"db_password"`                            // 数据库密码
	DbName              string        `mapstructure:"db_name"`                                // 数据库名称
	DbCharset           string        `mapstructure:"db_charset" default:"utf8mb4"`           // 数据库字符集，默认 "utf8mb4"
	DbMaxIdleConns      int           `mapstructure:"db_max_idle_conns" default:"10"`         // 最大空闲连接数，默认 10
	DbMaxOpenConns      int           `mapstructure:"db_max_open_conns" default:"100"`        // 最大打开连接数，默认 100
	DbConnMaxLifetime   time.Duration `mapstructure:"db_conn_max_lifetime" default:"1h"`      // 连接最大存活时间，默认 1小时
	DbParseTime         bool          `mapstructure:"db_parse_time" default:"true"`           // 是否解析时间，默认 true
	DbLoc               string        `mapstructure:"db_loc" default:"Local"`                 // 数据库时区，默认 "Local"
	DbLogLevel          string        `mapstructure:"db_log_level" default:"warn"`            // GORM 日志级别，可选 "silent", "error", "warn", "info"，默认 "warn"
	DbEnableAutoMigrate bool          `mapstructure:"db_enable_auto_migrate" default:"false"` // 是否启用自动迁移，默认 false
	DbSslMode           string        `mapstructure:"db_ssl_mode" default:"disable"`          // SSL 模式（PostgreSQL 可用），默认 "disable"
	DbSocket            string        `mapstructure:"db_socket"`                              // 数据库 Unix 套接字连接（适用于 Google Cloud 或特殊环境）
	DbDriver            string        `mapstructure:"db_driver" default:"mysql"`              // 数据库驱动类型，默认 "mysql"
}

func NewOption(cfg *config.ConfigManager) (*Option, error) {
	opt := defaultOption()
	if cfg == nil || cfg.GetViper() == nil {
		return opt, nil
	}
	if dbConfig := cfg.GetViper().Sub("db"); dbConfig != nil {
		if err := dbConfig.Unmarshal(opt); err != nil {
			return nil, fmt.Errorf("failed to unmarshal db options: %w", err)
		}
	}
	return opt, nil
}

func New(option *Option) *gorm.DB {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		option.DbUser, option.DbPassword, option.DbHost, option.DbPort,
		option.DbName, option.DbCharset, option.DbParseTime, option.DbLoc)

	// 初始化数据库连接
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("无法连接到数据库: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("无法获取数据库连接池: %v", err)
	}
	if option.DbMaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(option.DbMaxIdleConns)
	}
	if option.DbMaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(option.DbMaxOpenConns)
	}
	if option.DbConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(option.DbConnMaxLifetime)
	}

	return db
}

func defaultOption() *Option {
	return &Option{
		DbHost:            "localhost",
		DbPort:            3306,
		DbCharset:         "utf8mb4",
		DbMaxIdleConns:    10,
		DbMaxOpenConns:    100,
		DbConnMaxLifetime: time.Hour,
		DbParseTime:       true,
		DbLoc:             "Local",
		DbLogLevel:        "warn",
		DbSslMode:         "disable",
		DbDriver:          "mysql",
	}
}

var ProviderSet = wire.NewSet(New, NewOption)
