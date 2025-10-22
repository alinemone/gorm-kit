package gormkit

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	sqlite "github.com/glebarez/sqlite"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	Driver   string
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string

	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration

	LogLevel       string
	AutoMigrate    bool
	RetryAttempts  int
	ConnectTimeout time.Duration
}

type Manager struct {
	db     *gorm.DB
	sqlDB  *sql.DB
	config *Config
}

func New(cfg *Config) (*Manager, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	if cfg.MaxOpenConns == 0 {
		cfg.MaxOpenConns = 25
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = 5
	}
	if cfg.ConnMaxLifetime == 0 {
		cfg.ConnMaxLifetime = 5 * time.Minute
	}
	if cfg.ConnMaxIdleTime == 0 {
		cfg.ConnMaxIdleTime = 5 * time.Minute
	}
	if cfg.RetryAttempts == 0 {
		cfg.RetryAttempts = 3
	}
	if cfg.ConnectTimeout == 0 {
		cfg.ConnectTimeout = 10 * time.Second
	}

	m := &Manager{config: cfg}

	if err := m.connect(); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *Manager) connect() error {
	var dialector gorm.Dialector

	switch m.config.Driver {
	case "postgres":
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			m.config.Host, m.config.Port, m.config.User, m.config.Password,
			m.config.Database, m.config.SSLMode)
		dialector = postgres.Open(dsn)

	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			m.config.User, m.config.Password, m.config.Host, m.config.Port, m.config.Database)
		dialector = mysql.Open(dsn)

	case "sqlite", "test":
		if m.config.Database == "" {
			m.config.Database = ":memory:"
		}
		dialector = sqlite.Open(m.config.Database)

	default:
		return fmt.Errorf("unsupported driver: %s", m.config.Driver)
	}

	logLevel := logger.Info
	if m.config.LogLevel == "silent" {
		logLevel = logger.Silent
	} else if m.config.LogLevel == "error" {
		logLevel = logger.Error
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	var err error
	for i := 0; i < m.config.RetryAttempts; i++ {
		m.db, err = gorm.Open(dialector, gormConfig)
		if err == nil {
			break
		}
		if i < m.config.RetryAttempts-1 {
			time.Sleep(100 * time.Millisecond)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	m.sqlDB, err = m.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	m.sqlDB.SetMaxOpenConns(m.config.MaxOpenConns)
	m.sqlDB.SetMaxIdleConns(m.config.MaxIdleConns)
	m.sqlDB.SetConnMaxLifetime(m.config.ConnMaxLifetime)
	m.sqlDB.SetConnMaxIdleTime(m.config.ConnMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), m.config.ConnectTimeout)
	defer cancel()

	if err := m.sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	log.Printf("Connected to %s database: %s", m.config.Driver, m.config.Database)
	return nil
}

func (m *Manager) DB() *gorm.DB {
	return m.db
}

func (m *Manager) WithContext(ctx context.Context) *gorm.DB {
	return m.db.WithContext(ctx)
}

func (m *Manager) Migrate(models ...interface{}) error {
	if !m.config.AutoMigrate {
		return nil
	}
	return m.db.AutoMigrate(models...)
}

func (m *Manager) Transaction(ctx context.Context, fn func(*gorm.DB) error) error {
	return m.db.WithContext(ctx).Transaction(fn)
}

func (m *Manager) Ping(ctx context.Context) error {
	return m.sqlDB.PingContext(ctx)
}

func (m *Manager) Stats() sql.DBStats {
	return m.sqlDB.Stats()
}

func (m *Manager) Close() error {
	if m.sqlDB != nil {
		return m.sqlDB.Close()
	}
	return nil
}

func Paginate(page, limit int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page < 1 {
			page = 1
		}
		if limit < 1 {
			limit = 10
		}
		offset := (page - 1) * limit
		return db.Offset(offset).Limit(limit)
	}
}
