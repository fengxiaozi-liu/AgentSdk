package db

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"ferryman-agent/config"
	"github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Session struct {
	ID               string `gorm:"primaryKey"`
	ParentSessionID  string `gorm:"index"`
	Title            string
	MessageCount     int64
	PromptTokens     int64
	CompletionTokens int64
	SummaryMessageID string
	Cost             float64
	CreatedAt        int64 `gorm:"autoCreateTime"`
	UpdatedAt        int64 `gorm:"autoUpdateTime"`
}

type Message struct {
	ID         string `gorm:"primaryKey"`
	SessionID  string `gorm:"index"`
	Role       string
	Parts      string
	Model      string
	FinishedAt int64
	CreatedAt  int64 `gorm:"autoCreateTime"`
	UpdatedAt  int64 `gorm:"autoUpdateTime"`
}

type File struct {
	ID        string `gorm:"primaryKey"`
	SessionID string `gorm:"index"`
	Path      string `gorm:"index"`
	Content   string
	Version   string
	CreatedAt int64 `gorm:"autoCreateTime"`
	UpdatedAt int64 `gorm:"autoUpdateTime"`
}

func Open(cfg config.DatabaseConfig) (*gorm.DB, error) {
	if cfg.Type == "" {
		cfg.Type = config.DatabaseSQLite
	}

	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(parseLogLevel(cfg.LogLevel))}
	var (
		database *gorm.DB
		err      error
	)
	switch cfg.Type {
	case config.DatabaseSQLite:
		dsn := cfg.DSN
		if dsn == "" {
			dsn = cfg.Path
		}
		if dsn == "" {
			dsn = filepath.Join(config.DefaultDataDirectory, "agent.db")
		}
		if dir := filepath.Dir(dsn); dir != "." && dir != "" {
			if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
				return nil, mkErr
			}
		}
		database, err = gorm.Open(sqlite.Open(dsn), gormCfg)
	case config.DatabaseMySQL:
		dsn := cfg.DSN
		if dsn == "" {
			dsn = mysqlDSN(cfg)
		}
		database, err = gorm.Open(mysql.Open(dsn), gormCfg)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
	if err != nil {
		return nil, err
	}

	if sqlDB, err := database.DB(); err == nil {
		if cfg.MaxOpenConns > 0 {
			sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
		}
		if cfg.MaxIdleConns > 0 {
			sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
		}
		if cfg.ConnMaxLifetimeSecs > 0 {
			sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnMaxLifetimeSecs) * time.Second)
		}
	}

	if cfg.AutoMigrate {
		if err := AutoMigrate(database); err != nil {
			return nil, err
		}
	}
	return database, nil
}

func AutoMigrate(database *gorm.DB) error {
	return database.AutoMigrate(&Session{}, &Message{}, &File{})
}

func mysqlDSN(cfg config.DatabaseConfig) string {
	host := cfg.Host
	if host == "" {
		host = "127.0.0.1"
	}
	port := cfg.Port
	if port == 0 {
		port = 3306
	}
	charset := cfg.Charset
	if charset == "" {
		charset = "utf8mb4"
	}
	loc := cfg.Loc
	if loc == "" {
		loc = "Local"
	}
	values := url.Values{}
	values.Set("charset", charset)
	values.Set("parseTime", fmt.Sprintf("%t", cfg.ParseTime))
	values.Set("loc", loc)
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?%s",
		cfg.Username,
		cfg.Password,
		host,
		port,
		cfg.Database,
		values.Encode(),
	)
}

func parseLogLevel(level string) logger.LogLevel {
	switch level {
	case "silent":
		return logger.Silent
	case "error":
		return logger.Error
	case "warn":
		return logger.Warn
	case "info":
		return logger.Info
	default:
		return logger.Warn
	}
}
