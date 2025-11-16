package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/YusovID/pr-reviewer-service/internal/config"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/ilyakaznacheev/cleanenv"
)

type MigrationCfg struct {
	ConnStr         string
	MigrationsPath  string
	MigrationsTable string
}

func main() {
	migration, err := Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	m, err := migrate.New(
		"file://"+migration.MigrationsPath,
		fmt.Sprintf("%s?sslmode=disable&x-migrations-table=%s", migration.ConnStr, migration.MigrationsTable),
	)
	if err != nil {
		log.Fatalf("can't create new migration: %v", err)
	}

	var cmd string
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "down":
		if err := m.Down(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				fmt.Println("no migrations to roll back")
				return
			}
			log.Fatalf("can't down migrations: %v", err)
		}
		fmt.Println("migrations rolled back successfully")
	case "up":
		fallthrough
	default:
		if err := m.Up(); err != nil {
			if errors.Is(err, migrate.ErrNoChange) {
				fmt.Println("no new migrations to apply")
				return
			}

			log.Fatalf("can't do migrations: %v", err)
		}
		fmt.Println("migrations applied successfully")
	}
}

func Load() (*MigrationCfg, error) {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		return nil, fmt.Errorf("CONFIG_PATH is not set")
	}

	if _, err := os.Stat(configPath); err != nil {
		return nil, fmt.Errorf("file '%s' doesn't exist: %v", configPath, err)
	}

	migrationsPath := os.Getenv("MIGRATIONS_PATH")
	if migrationsPath == "" {
		return nil, fmt.Errorf("MIGRATIONS_PATH is not set")
	}

	migrationsTable := os.Getenv("MIGRATIONS_TABLE")
	if migrationsTable == "" {
		return nil, fmt.Errorf("MIGRATIONS_TABLE is not set")
	}

	var cfg config.Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		return nil, fmt.Errorf("can't read config: %v", err)
	}

	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.Postgres.Username,
		cfg.Postgres.Password,
		cfg.Postgres.Host,
		cfg.Postgres.Port,
		cfg.Postgres.Database,
	)

	return &MigrationCfg{
		ConnStr:         connStr,
		MigrationsPath:  migrationsPath,
		MigrationsTable: migrationsTable,
	}, nil
}
