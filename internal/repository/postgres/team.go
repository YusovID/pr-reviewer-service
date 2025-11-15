package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"

	sq "github.com/Masterminds/squirrel"
	"github.com/YusovID/pr-reviewer-service/internal/apperrors"
	"github.com/YusovID/pr-reviewer-service/internal/domain"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/YusovID/pr-reviewer-service/pkg/logger/sl"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type TeamRepository struct {
	db  *sqlx.DB
	log *slog.Logger
	sq  sq.StatementBuilderType
}

func NewTeamRepository(db *sqlx.DB, log *slog.Logger) *TeamRepository {
	return &TeamRepository{
		db:  db,
		log: log,
		sq:  sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (tr *TeamRepository) CreateTeamWithUsers(ctx context.Context, team api.Team) (*domain.TeamWithMembers, error) {
	const op = "internal.repository.postgres.CreateTeamWithUsers"
	log := tr.log.With(slog.String("op", op), slog.String("team_name", team.TeamName))
	log.Info("creating team with users")

	tx, err := tr.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err := tx.Rollback(); err != nil && !errors.Is(err, sql.ErrTxDone) {
			log.Error("failed to rollback transaction", sl.Err(err))
		}
	}()

	createdTeam, err := tr.insertTeam(ctx, tx, team.TeamName)
	if err != nil {
		return nil, err
	}

	if len(team.Members) > 0 {
		err = tr.upsertTeamMembers(ctx, tx, createdTeam.ID, team.Members)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	domainMembers := make([]domain.User, len(team.Members))
	for i, member := range team.Members {
		domainMembers[i] = domain.User{
			ID:       member.UserId,
			Username: member.Username,
			TeamID:   createdTeam.ID,
			IsActive: member.IsActive,
		}
	}

	result := &domain.TeamWithMembers{
		ID:      createdTeam.ID,
		Name:    createdTeam.Name,
		Members: domainMembers,
	}

	log.Info("team created successfully", slog.Int("team_id", createdTeam.ID))

	return result, nil
}

func (tr *TeamRepository) insertTeam(ctx context.Context, tx *sqlx.Tx, teamName string) (*domain.Team, error) {
	query, args, err := tr.sq.Insert("teams").
		Columns("name").
		Values(teamName).
		Suffix("RETURNING id, name").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build team insert query: %w", err)
	}

	var createdTeam domain.Team

	err = tx.QueryRowxContext(ctx, query, args...).StructScan(&createdTeam)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return nil, &apperrors.TeamAlreadyExistsError{TeamName: teamName}
		}

		return nil, fmt.Errorf("failed to execute team insert: %w", err)
	}

	return &createdTeam, nil
}

func (tr *TeamRepository) upsertTeamMembers(ctx context.Context, tx *sqlx.Tx, teamID int, members []api.TeamMember) error {
	insertBuilder := tr.sq.Insert("users").
		Columns("id", "username", "team_id", "is_active")

	for _, member := range members {
		insertBuilder = insertBuilder.Values(
			member.UserId,
			member.Username,
			teamID,
			member.IsActive,
		)
	}

	query, args, err := insertBuilder.Suffix(`
        ON CONFLICT (id) DO UPDATE SET
            username = EXCLUDED.username,
            team_id = EXCLUDED.team_id,
            is_active = EXCLUDED.is_active`).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build bulk users upsert query: %w", err)
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to execute bulk users upsert: %w", err)
	}

	return nil
}

func (tr *TeamRepository) GetTeamByName(ctx context.Context, name string) (*domain.TeamWithMembers, error) {
	const op = "internal.repository.postgres.GetTeamByName"
	log := tr.log.With(slog.String("op", op), slog.String("team_name", name))
	log.Info("getting team by name")

	query, args, err := tr.sq.Select("id", "name").
		From("teams").
		Where(sq.Eq{"name": name}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select team query: %w", err)
	}

	var team domain.Team
	if err := tr.db.GetContext(ctx, &team, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%w: team with name '%s'", apperrors.ErrNotFound, name)
		}

		return nil, fmt.Errorf("failed to get team by name: %w", err)
	}

	queryMembers, args, err := tr.sq.Select("id", "username", "team_id", "is_active").
		From("users").
		Where(sq.Eq{"team_id": team.ID}).
		OrderBy("username").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("failed to build select members query: %w", err)
	}

	var members []domain.User
	if err := tr.db.SelectContext(ctx, &members, queryMembers, args...); err != nil {
		return nil, fmt.Errorf("failed to get team members: %w", err)
	}

	log.Info("team getting successfull")

	return &domain.TeamWithMembers{
		ID:      team.ID,
		Name:    team.Name,
		Members: members,
	}, nil
}
