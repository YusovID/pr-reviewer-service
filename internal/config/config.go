package config

import (
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env      string `yml:"env" default:"local"`
	Postgres Postgres
	Server   Server `yml:"server" env-required:"true"`
}

// Postgres содержит параметры для подключения к базе данных PostgreSQL.
type Postgres struct {
	Username string `env:"POSTGRES_USER" env-required:"true"`
	Password string `env:"POSTGRES_PASSWORD" env-required:"true"`
	Host     string `env:"POSTGRES_HOST" env-required:"true"`
	Port     string `env:"POSTGRES_PORT" env-required:"true"`
	Database string `env:"POSTGRES_DB" env-required:"true"`
}

type Server struct {
	Host    string `yml:"host" default:"localhost"`
	Port    string `yml:"port" default:"8080"`
	Timeout string `yml:"timeout" default:"5s"`
}

func MustLoad() *Config {
	// Получаем путь к файлу конфигурации из переменной окружения.
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("CONFIG_PATH is not set")
	}

	// Проверяем, существует ли файл по указанному пути.
	if _, err := os.Stat(configPath); err != nil {
		log.Fatalf("config file does not exist: %s", configPath)
	}

	var cfg Config

	// Читаем YAML-файл и переменные окружения в структуру Config.
	// cleanenv автоматически сопоставляет поля структуры с данными из источников.
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("cannot read config: %s", err)
	}

	return &cfg
}
