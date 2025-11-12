package postgres

import (
	"fmt"
	"log/slog"

	"github.com/Masterminds/squirrel"
	"github.com/YusovID/pr-reviewer-service/internal/config"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type Postgres struct {
	db  *sqlx.DB
	log *slog.Logger
	sq  squirrel.StatementBuilderType
}

func NewDB(cfg config.Postgres, log *slog.Logger) (*Postgres, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Database,
	)

	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("can't connect to database: %v", err)
	}

	return &Postgres{
		db:  db,
		log: log,
		sq:  squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}, nil
}

func (p *Postgres) DB() *sqlx.DB {
	return p.db
}
