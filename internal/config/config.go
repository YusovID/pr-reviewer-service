package config

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env      string   `yml:"env" default:"local"`
	Postgres Postgres `yml:"postgres"`
	Server   Server   `yml:"server" env-required:"true"`
}

type Postgres struct {
	Username        string        `env:"POSTGRES_USER" env-required:"true"`
	Password        string        `env:"POSTGRES_PASSWORD" env-required:"true"`
	Host            string        `yml:"host" env-required:"true"`
	Port            string        `env:"POSTGRES_PORT" env-required:"true"`
	Database        string        `env:"POSTGRES_DB" env-required:"true"`
	MaxOpenConns    int           `yml:"max_open_conns" default:"50"`
	MaxIdleConns    int           `yml:"max_idle_conns" default:"10"`
	ConnMaxLifetime time.Duration `yml:"conn_max_lifetime" default:"5m"`
	ConnMaxIdleTime time.Duration `yml:"conn_max_idle_time" default:"1m"`
}

type Server struct {
	Host    string        `yml:"host" default:"localhost"`
	Port    string        `yml:"port" default:"8080"`
	Timeout time.Duration `yml:"timeout" default:"5s"`
}

func Load() (*Config, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		return nil, errors.New("CONFIG_PATH is not set")
	}

	if _, err := os.Stat(configPath); err != nil {
		return nil, fmt.Errorf("config file does not exist: %w", err)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("cannot read config: %w", err)
	}

	return &cfg, nil
}
