package database

import (
	"database/sql"
	"fmt"
	"strings"

	"login/internal/config"

	_ "github.com/go-sql-driver/mysql"
)

func NewMySQL(cfg config.MySQLConfig) (*sql.DB, error) {
	db, err := sql.Open("mysql", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	return db, nil
}

func EnsureDatabase(adminDSN, dbName string) error {
	db, err := sql.Open("mysql", adminDSN)
	if err != nil {
		return fmt.Errorf("open mysql admin dsn: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("ping mysql admin dsn: %w", err)
	}

	safeDBName := strings.ReplaceAll(dbName, "`", "")
	if _, err := db.Exec("CREATE DATABASE IF NOT EXISTS `" + safeDBName + "` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"); err != nil {
		return fmt.Errorf("create database %s: %w", safeDBName, err)
	}

	return nil
}
