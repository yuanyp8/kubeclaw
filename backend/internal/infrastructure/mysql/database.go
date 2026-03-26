package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	mysqlsql "github.com/go-sql-driver/mysql"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"kubeclaw/backend/internal/config"
)

// Database 封装 GORM 和底层 sql.DB，便于同时使用 ORM 与原生能力。
type Database struct {
	Gorm *gorm.DB
	SQL  *sql.DB
}

// Open 创建数据库连接，并按配置完成建库与连接池初始化。
func Open(ctx context.Context, cfg config.Config) (*Database, error) {
	serverCfg := mysqlsql.Config{
		User:   cfg.MySQLUser,
		Passwd: cfg.MySQLPassword,
		Net:    "tcp",
		Addr:   fmt.Sprintf("%s:%d", cfg.MySQLHost, cfg.MySQLPort),
		Params: map[string]string{
			"charset": cfg.MySQLCharset,
		},
		AllowNativePasswords: true,
		ParseTime:            cfg.MySQLParseTime,
		Loc:                  time.Local,
	}
	serverDSN := serverCfg.FormatDSN()

	sqlDB, err := sql.Open("mysql", serverDSN)
	if err != nil {
		return nil, fmt.Errorf("open mysql server connection: %w", err)
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping mysql server: %w", err)
	}

	if _, err := sqlDB.ExecContext(ctx, fmt.Sprintf(
		"CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET %s COLLATE %s_unicode_ci",
		cfg.MySQLDatabase,
		cfg.MySQLCharset,
		cfg.MySQLCharset,
	)); err != nil {
		return nil, fmt.Errorf("create database if not exists: %w", err)
	}

	appCfg := mysqlsql.Config{
		User:                 cfg.MySQLUser,
		Passwd:               cfg.MySQLPassword,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%d", cfg.MySQLHost, cfg.MySQLPort),
		DBName:               cfg.MySQLDatabase,
		Params:               map[string]string{"charset": cfg.MySQLCharset},
		AllowNativePasswords: true,
		ParseTime:            cfg.MySQLParseTime,
		Loc:                  time.Local,
	}
	appDSN := appCfg.FormatDSN()

	gormDB, err := gorm.Open(mysql.Open(appDSN), &gorm.Config{
		Logger: newZapGormLogger(),
	})
	if err != nil {
		return nil, fmt.Errorf("open gorm mysql connection: %w", err)
	}

	rawDB, err := gormDB.DB()
	if err != nil {
		return nil, fmt.Errorf("get raw sql db from gorm: %w", err)
	}

	rawDB.SetMaxOpenConns(cfg.MySQLMaxOpenConns)
	rawDB.SetMaxIdleConns(cfg.MySQLMaxIdleConns)
	rawDB.SetConnMaxLifetime(cfg.MySQLConnMaxLifetime)

	if err := rawDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping mysql app database: %w", err)
	}

	return &Database{
		Gorm: gormDB,
		SQL:  rawDB,
	}, nil
}
