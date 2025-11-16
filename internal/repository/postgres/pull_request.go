package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/YusovID/pr-reviewer-service/internal/apperrors"
	"github.com/YusovID/pr-reviewer-service/internal/domain"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/YusovID/pr-reviewer-service/pkg/logger/sl"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type PullRequestRepository struct {
	db  *sqlx.DB
	log *slog.Logger
	sq  sq.StatementBuilderType
}

func NewPullRequestRepository(db *sqlx.DB, log *slog.Logger) *PullRequestRepository {
	return &PullRequestRepository{
		db:  db,
		log: log,
		sq:  sq.StatementBuilder.PlaceholderFormat(sq.Dollar),
	}
}

func (r *PullRequestRepository) GetAuthorTeamID(ctx context.Context, authorID string) (int, error) {
	const op = "internal.repository.postgres.GetAuthorTeamID"

	query, args, err := r.sq.Select("team_id").
		From("users").
		Where(sq.Eq{"id": authorID}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to build query: %w", op, err)
	}

	var teamID int
	if err := r.db.GetContext(ctx, &teamID, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("%s: %w: user with id '%s'", op, apperrors.ErrNotFound, authorID)
		}

		return 0, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}

	return teamID, nil
}

func (r *PullRequestRepository) GetRandomActiveReviewers(ctx context.Context, teamID int, excludeUserIDs []string, count int) ([]string, error) {
	const op = "internal.repository.postgres.GetRandomActiveReviewers"

	queryBuilder := r.sq.Select("id").
		From("users").
		Where(sq.Eq{"team_id": teamID, "is_active": true})

	if len(excludeUserIDs) > 0 {
		queryBuilder = queryBuilder.Where(sq.NotEq{"id": excludeUserIDs})
	}

	query, args, err := queryBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to build query: %w", op, err)
	}

	var allCandidateIDs []string
	if err := r.db.SelectContext(ctx, &allCandidateIDs, query, args...); err != nil {
		return nil, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}

	numCandidates := len(allCandidateIDs)
	if numCandidates == 0 {
		return []string{}, nil
	}

	if numCandidates <= count {
		return allCandidateIDs, nil
	}

	result := make([]string, count)

	for i := 0; i < count; i++ {
		idx := rand.Intn(numCandidates-i) + i
		allCandidateIDs[i], allCandidateIDs[idx] = allCandidateIDs[idx], allCandidateIDs[i]
		result[i] = allCandidateIDs[i]
	}

	return result, nil
}

func (r *PullRequestRepository) CreatePR(ctx context.Context, tx *sqlx.Tx, pr *domain.PullRequest) error {
	const op = "internal.repository.postgres.CreatePR"

	query, args, err := r.sq.Insert("pull_requests").
		Columns("id", "name", "author_id", "status", "need_more_reviewers").
		Values(pr.ID, pr.Name, pr.AuthorID, pr.Status, pr.NeedMoreReviewers).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: failed to build insert query: %w", op, err)
	}

	_, err = tx.ExecContext(ctx, query, args...)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return &apperrors.PRAlreadyExistsError{PRID: pr.ID}
			}

			if pqErr.Code == "23503" {
				return fmt.Errorf("%s: %w: author with id '%s' not found", op, apperrors.ErrNotFound, pr.AuthorID)
			}
		}

		return fmt.Errorf("%s: failed to execute insert: %w", op, err)
	}

	return nil
}

func (r *PullRequestRepository) AssignReviewers(ctx context.Context, tx *sqlx.Tx, prID string, reviewerIDs []string) error {
	const op = "internal.repository.postgres.AssignReviewers"

	insertBuilder := r.sq.Insert("reviewers").
		Columns("pull_request_id", "user_id")

	for _, userID := range reviewerIDs {
		insertBuilder = insertBuilder.Values(prID, userID)
	}

	query, args, err := insertBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("%s: failed to build insert query: %w", op, err)
	}

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("%s: failed to execute insert: %w", op, err)
	}

	return nil
}

func (r *PullRequestRepository) GetReviewerIDs(ctx context.Context, ext sqlx.ExtContext, prID string) ([]string, error) {
	const op = "internal.repository.postgres.GetReviewerIDs"

	query, args, err := r.sq.Select("user_id").
		From("reviewers").
		Where(sq.Eq{"pull_request_id": prID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to build query: %w", op, err)
	}

	var reviewerIDs []string
	if err := sqlx.SelectContext(ctx, ext, &reviewerIDs, query, args...); err != nil {
		return nil, fmt.Errorf("%s: failed to select reviewers: %w", op, err)
	}

	return reviewerIDs, nil
}

func (r *PullRequestRepository) GetPRByIDWithReviewers(ctx context.Context, prID string) (*domain.PullRequest, error) {
	const op = "internal.repository.postgres.GetPRByIDWithReviewers"

	pr, err := r.GetPRByID(ctx, prID)
	if err != nil {
		return nil, err
	}

	reviewerIDs, err := r.GetReviewerIDs(ctx, r.db, prID)
	if err != nil {
		r.log.Error("failed to get reviewers for PR", sl.Err(err), slog.String("pr_id", prID))
		return nil, fmt.Errorf("%s: failed to get reviewers: %w", op, err)
	}

	pr.ReviewerIDs = reviewerIDs

	return pr, nil
}

func (r *PullRequestRepository) GetPRByIDWithLock(ctx context.Context, tx *sqlx.Tx, prID string) (*domain.PullRequest, error) {
	const op = "internal.repository.postgres.GetPRByIDWithLock"

	query, args, err := r.sq.Select("id", "name", "author_id", "status", "need_more_reviewers", "created_at", "merged_at").
		From("pull_requests").
		Where(sq.Eq{"id": prID}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to build query: %w", op, err)
	}

	var pr domain.PullRequest
	if err := tx.GetContext(ctx, &pr, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w: PR with id '%s'", op, apperrors.ErrNotFound, prID)
		}

		return nil, fmt.Errorf("%s: failed to get PR with lock: %w", op, err)
	}

	return &pr, nil
}

func (r *PullRequestRepository) GetPRByID(ctx context.Context, prID string) (*domain.PullRequest, error) {
	const op = "internal.repository.postgres.GetPRByID"

	query, args, err := r.sq.Select("id", "name", "author_id", "status", "need_more_reviewers", "created_at", "merged_at").
		From("pull_requests").
		Where(sq.Eq{"id": prID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to build query: %w", op, err)
	}

	var pr domain.PullRequest
	if err := r.db.GetContext(ctx, &pr, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("%s: %w: PR with id '%s'", op, apperrors.ErrNotFound, prID)
		}

		return nil, fmt.Errorf("%s: failed to get PR: %w", op, err)
	}

	return &pr, nil
}

func (r *PullRequestRepository) UpdatePRStatus(ctx context.Context, tx *sqlx.Tx, prID string, status api.PullRequestStatus, mergedAt time.Time) error {
	const op = "internal.repository.postgres.UpdatePRStatus"

	updateBuilder := r.sq.Update("pull_requests").
		Set("status", status).
		Where(sq.Eq{"id": prID})

	if status == api.PullRequestStatusMERGED {
		updateBuilder = updateBuilder.Set("merged_at", mergedAt).Set("need_more_reviewers", false)
	}

	query, args, err := updateBuilder.ToSql()
	if err != nil {
		return fmt.Errorf("%s: failed to build update query: %w", op, err)
	}

	res, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%s: failed to execute update: %w", op, err)
	}

	if rowsAffected, err := res.RowsAffected(); err == nil && rowsAffected == 0 {
		return fmt.Errorf("%s: %w: PR with id '%s'", op, apperrors.ErrNotFound, prID)
	}

	return nil
}

func (r *PullRequestRepository) GetReviewerTeamID(ctx context.Context, reviewerID string) (int, error) {
	const op = "internal.repository.postgres.GetReviewerTeamID"

	query, args, err := r.sq.Select("team_id").
		From("users").
		Where(sq.Eq{"id": reviewerID}).
		ToSql()
	if err != nil {
		return 0, fmt.Errorf("%s: failed to build query: %w", op, err)
	}

	var teamID int
	if err := r.db.GetContext(ctx, &teamID, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("%s: %w: reviewer user with id '%s'", op, apperrors.ErrNotFound, reviewerID)
		}

		return 0, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}

	return teamID, nil
}

func (r *PullRequestRepository) ReplaceReviewer(ctx context.Context, tx *sqlx.Tx, prID string, oldReviewerID string, newReviewerID string) error {
	const op = "internal.repository.postgres.ReplaceReviewer"

	deleteQuery, deleteArgs, err := r.sq.Delete("reviewers").
		Where(sq.Eq{"pull_request_id": prID, "user_id": oldReviewerID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: failed to build delete query: %w", op, err)
	}

	if _, err := tx.ExecContext(ctx, deleteQuery, deleteArgs...); err != nil {
		return fmt.Errorf("%s: failed to execute delete: %w", op, err)
	}

	insertQuery, insertArgs, err := r.sq.Insert("reviewers").
		Columns("pull_request_id", "user_id").
		Values(prID, newReviewerID).
		ToSql()
	if err != nil {
		return fmt.Errorf("%s: failed to build insert query: %w", op, err)
	}

	if _, err := tx.ExecContext(ctx, insertQuery, insertArgs...); err != nil {
		return fmt.Errorf("%s: failed to execute insert: %w", op, err)
	}

	return nil
}

func (r *PullRequestRepository) GetReviewAssignments(ctx context.Context, userID string) ([]domain.PullRequest, error) {
	const op = "internal.repository.postgres.GetReviewAssignments"
	log := r.log.With(slog.String("op", op), slog.String("user_id", userID))

	query, args, err := r.sq.Select(
		"pr.id", "pr.name", "pr.author_id", "pr.status",
	).From("pull_requests pr").
		Join("reviewers r ON pr.id = r.pull_request_id").
		Where(sq.Eq{"r.user_id": userID}).
		OrderBy("pr.created_at DESC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to build query: %w", op, err)
	}

	var prs []domain.PullRequest
	if err := r.db.SelectContext(ctx, &prs, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Info("no review assignments found")
			return []domain.PullRequest{}, nil
		}

		return nil, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}

	return prs, nil
}

func (r *PullRequestRepository) GetUserStats(ctx context.Context) ([]domain.Stats, error) {
	const op = "internal.repository.postgres.GetUserStats"

	query, args, err := r.sq.Select(
		"u.id as user_id",
		"u.username",
		"COUNT(CASE WHEN pr.status = 'OPEN' THEN 1 END) as open_reviews",
		"COUNT(CASE WHEN pr.status = 'MERGED' THEN 1 END) as merged_reviews",
	).
		From("users u").
		LeftJoin("reviewers r ON u.id = r.user_id").
		LeftJoin("pull_requests pr ON r.pull_request_id = pr.id").
		GroupBy("u.id", "u.username").
		OrderBy("u.username").
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("%s: failed to build query: %w", op, err)
	}

	var stats []domain.Stats
	if err := r.db.SelectContext(ctx, &stats, query, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return []domain.Stats{}, nil
		}

		return nil, fmt.Errorf("%s: failed to execute query: %w", op, err)
	}

	return stats, nil
}

func mapReviewersToPRs(prs []domain.PullRequest, reviewers []domain.Reviewer) []domain.PullRequest {
	prMap := make(map[string]*domain.PullRequest, len(prs))
	for i := range prs {
		prMap[prs[i].ID] = &prs[i]
	}

	for _, reviewer := range reviewers {
		if pr, ok := prMap[reviewer.PullRequestID]; ok {
			pr.ReviewerIDs = append(pr.ReviewerIDs, reviewer.UserID)
		}
	}

	result := make([]domain.PullRequest, 0, len(prMap))
	for _, pr := range prMap {
		result = append(result, *pr)
	}

	return result
}

func (r *PullRequestRepository) GetOpenPRsByReviewers(ctx context.Context, tx *sqlx.Tx, userIDs []string) ([]domain.PullRequest, error) {
	const op = "internal.repository.postgres.GetOpenPRsByReviewers"

	prIDsQuery, args, err := r.sq.Select("DISTINCT pull_request_id").
		From("reviewers").
		Join("pull_requests pr ON pr.id = reviewers.pull_request_id").
		Where(sq.Eq{"reviewers.user_id": userIDs, "pr.status": "OPEN"}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to build pr_ids query: %w", op, err)
	}

	var prIDs []string
	if err := tx.SelectContext(ctx, &prIDs, prIDsQuery, args...); err != nil {
		return nil, fmt.Errorf("%s: failed to select pr_ids: %w", op, err)
	}

	if len(prIDs) == 0 {
		return []domain.PullRequest{}, nil
	}

	prsQuery, args, err := r.sq.Select("id", "name", "author_id", "status").
		From("pull_requests").
		Where(sq.Eq{"id": prIDs}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to build prs query: %w", op, err)
	}

	var prs []domain.PullRequest
	if err := tx.SelectContext(ctx, &prs, prsQuery, args...); err != nil {
		return nil, fmt.Errorf("%s: failed to select prs: %w", op, err)
	}

	reviewersQuery, args, err := r.sq.Select("pull_request_id", "user_id").
		From("reviewers").
		Where(sq.Eq{"pull_request_id": prIDs}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("%s: failed to build reviewers query: %w", op, err)
	}

	var reviewers []domain.Reviewer
	if err := tx.SelectContext(ctx, &reviewers, reviewersQuery, args...); err != nil {
		return nil, fmt.Errorf("%s: failed to select reviewers: %w", op, err)
	}

	resultPRs := mapReviewersToPRs(prs, reviewers)

	return resultPRs, nil
}
