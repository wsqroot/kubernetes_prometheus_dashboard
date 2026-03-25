package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server          ServerConfig     `yaml:"server"`
	MySQL           MySQLConfig      `yaml:"mysql"`
	KubeconfigMySQL KubeconfigDB     `yaml:"kubeconfig_mysql"`
	Auth            AuthConfig       `yaml:"auth"`
	CORS            CORSConfig       `yaml:"cors"`
	Kubeconfig      KubeconfigConfig `yaml:"kubeconfig"`
}

type ServerConfig struct {
	Addr         string        `yaml:"addr"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
	IdleTimeout  time.Duration `yaml:"idle_timeout"`
}

type MySQLConfig struct {
	DSN             string        `yaml:"dsn"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

type AuthConfig struct {
	UserLookupSQL string `yaml:"user_lookup_sql"`
	PasswordMode  string `yaml:"password_mode"`
}

type CORSConfig struct {
	AllowedOrigins []string `yaml:"allowed_origins"`
}

type KubeconfigConfig struct {
	MaxUploadSizeMB int64 `yaml:"max_upload_size_mb"`
}

type KubeconfigDB struct {
	AdminDSN        string        `yaml:"admin_dsn"`
	DSN             string        `yaml:"dsn"`
	Database        string        `yaml:"database"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

func Load(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":8080"
	}
	if cfg.Kubeconfig.MaxUploadSizeMB <= 0 {
		cfg.Kubeconfig.MaxUploadSizeMB = 10
	}
	if cfg.KubeconfigMySQL.Database == "" {
		cfg.KubeconfigMySQL.Database = "kubeconfig"
	}

	return &cfg, nil
}
