package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	sq "github.com/Masterminds/squirrel"
	"github.com/YusovID/pr-reviewer-service/internal/apperrors"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/jmoiron/sqlx"
)

type UserRepository struct {
	db  *sqlx.DB
	log *slog.Logger
	sq  sq.StatementBuilderType
}

func NewUserRepository(db *sqlx.DB, log *slog.Logger) *UserRepository {
	return &UserRepository{
		db:  db,
		log: log,
		sq:  sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

type userWithTeamName struct {
	UserID   string `db:"user_id"`
	Username string `db:"username"`
	TeamName string `db:"team_name"`
	IsActive bool   `db:"is_active"`
}

func (ur *UserRepository) SetIsActive(ctx context.Context, userID string, isActive bool) (*api.User, error) {
	const op = "internal.repository.postgres.SetIsActive"

	ur.log.With(slog.String("op", op))
	ur.log.Info("setting", slog.String("userID", userID), slog.Bool("is active", isActive))

	query, args, err := ur.sq.Update("users").
		Set("is_active", isActive).
		Where(sq.Eq{"id": userID}).
		Suffix(`RETURNING 
            users.id as user_id, 
            users.username, 
            (SELECT name FROM teams WHERE id = users.team_id) as team_name, 
            users.is_active`).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("failed to build update user query: %w", err)
	}

	var dbUser userWithTeamName
	if err = ur.db.QueryRowxContext(ctx, query, args...).StructScan(&dbUser); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: user with id '%s'", apperrors.ErrNotFound, userID)
		}

		return nil, fmt.Errorf("failed to execute update user status: %w", err)
	}

	ur.log.Info("setting completed successfully")

	return &api.User{
		UserId:   dbUser.UserID,
		Username: dbUser.Username,
		TeamName: dbUser.TeamName,
		IsActive: dbUser.IsActive,
	}, nil
}

func (ur *UserRepository) DeactivateUsersByTeamID(ctx context.Context, tx *sqlx.Tx, teamID int) ([]string, error) {
	const op = "internal.repository.postgres.DeactivateUsersByTeamID"

	query, args, err := ur.sq.Update("users").
		Set("is_active", false).
		Where(sq.Eq{"team_id": teamID, "is_active": true}).
		Suffix("RETURNING id").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to build update query: %w", op, err)
	}

	var deactivatedUserIDs []string
	if err := tx.SelectContext(ctx, &deactivatedUserIDs, query, args...); err != nil {
		return nil, fmt.Errorf("%s: failed to execute update: %w", op, err)
	}

	return deactivatedUserIDs, nil
}
