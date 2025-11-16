//go:build integration

package postgres

import (
	"context"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var (
	testDB *sqlx.DB
	logger *slog.Logger
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	pgContainer, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:17"),
		postgres.WithDatabase("test-db"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("could not start postgres container: %s", err)
	}
	defer func() {
		if err := pgContainer.Terminate(ctx); err != nil {
			log.Fatalf("could not stop postgres container: %s", err)
		}
	}()

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatalf("failed to get connection string: %s", err)
	}

	testDB, err = sqlx.Connect("postgres", connStr)
	if err != nil {
		log.Fatalf("failed to connect to test postgres: %s", err)
	}

	_, b, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(b), "../../../")
	migrationsPath := filepath.Join(projectRoot, "migrations")

	slashedPath := filepath.ToSlash(migrationsPath)

	sourceURL := "file://" + slashedPath

	migrator, err := migrate.New(sourceURL, connStr)
	if err != nil {
		log.Fatalf("failed to create migrator with url '%s': %s", sourceURL, err)
	}

	if err = migrator.Up(); err != nil {
		log.Fatalf("failed to run migrations: %s", err)
	}

	code := m.Run()

	os.Exit(code)
}

func truncateTables(t *testing.T, db *sqlx.DB) {
	t.Helper()
	_, err := db.Exec("TRUNCATE TABLE teams, users, pull_requests, reviewers RESTART IDENTITY CASCADE")
	if err != nil {
		t.Fatalf("failed to truncate tables: %v", err)
	}
}
