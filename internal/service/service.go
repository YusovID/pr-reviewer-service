package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	"github.com/YusovID/pr-reviewer-service/pkg/logger/sl"
	"github.com/jmoiron/sqlx"
)

type Transactor interface {
	BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error)
}

type BaseService struct {
	db  Transactor
	log *slog.Logger
}

func NewBaseService(db Transactor, log *slog.Logger) BaseService {
	return BaseService{
		db:  db,
		log: log,
	}
}

func (s *BaseService) transaction(ctx context.Context, op string, fn func(tx *sqlx.Tx) error) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%s: failed to begin transaction: %w", op, err)
	}

	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			s.log.Error("failed to rollback transaction", sl.Err(err))
		}
	}()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%s: failed to commit transaction: %w", op, err)
	}

	return nil
}
