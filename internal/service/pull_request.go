package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/YusovID/pr-reviewer-service/internal/apperrors"
	"github.com/YusovID/pr-reviewer-service/internal/domain"
	"github.com/YusovID/pr-reviewer-service/internal/repository"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/YusovID/pr-reviewer-service/pkg/logger/sl"
	"github.com/jmoiron/sqlx"
)

type Transactor interface {
	BeginTxx(context.Context, *sql.TxOptions) (*sqlx.Tx, error)
}

type PullRequestService interface {
	CreatePR(ctx context.Context, prID string, prName string, authorID string) (*api.PullRequest, error)
	MergePR(ctx context.Context, prID string) (*api.PullRequest, error)
	ReassignReviewer(ctx context.Context, prID string, oldReviewerID string) (*api.ReassignResponse, error)
	GetReviewAssignments(ctx context.Context, userID string) (*api.GetReviewResponse, error)
	GetStats(ctx context.Context) (*api.StatsResponse, error)
}

type PullRequestServiceImpl struct {
	db      Transactor
	log     *slog.Logger
	prCmd   repository.PRCommandRepository
	prQuery repository.PRQueryRepository
	userPR  repository.UserPRRepository
}

func NewPullRequestService(
	db Transactor,
	log *slog.Logger,
	prCmd repository.PRCommandRepository,
	prQuery repository.PRQueryRepository,
	userPR repository.UserPRRepository,
) *PullRequestServiceImpl {
	return &PullRequestServiceImpl{
		db:      db,
		log:     log,
		prCmd:   prCmd,
		prQuery: prQuery,
		userPR:  userPR,
	}
}

func (s *PullRequestServiceImpl) CreatePR(ctx context.Context, prID string, prName string, authorID string) (*api.PullRequest, error) {
	const op = "internal.service.pullrequest.CreatePR"
	log := s.log.With(slog.String("op", op), slog.String("pr_id", prID), slog.String("author_id", authorID))

	teamID, err := s.userPR.GetAuthorTeamID(ctx, authorID)
	if err != nil {
		if errors.Is(err, apperrors.ErrNotFound) {
			return nil, fmt.Errorf("%w: author not found or has no team", apperrors.ErrNotFound)
		}

		return nil, fmt.Errorf("%s: failed to get author team id: %w", op, err)
	}

	reviewerIDs, err := s.userPR.GetRandomActiveReviewers(ctx, teamID, []string{authorID}, 2)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get random reviewers: %w", op, err)
	}

	log.Info("found reviewers", slog.Any("reviewers", reviewerIDs))

	pr := &domain.PullRequest{
		ID:                prID,
		Name:              prName,
		AuthorID:          authorID,
		Status:            api.PullRequestStatusOPEN,
		NeedMoreReviewers: len(reviewerIDs) < 2,
		CreatedAt:         time.Now().UTC(),
	}

	err = s.transaction(ctx, op, func(tx *sqlx.Tx) error {
		if err := s.prCmd.CreatePR(ctx, tx, pr); err != nil {
			return err
		}

		if len(reviewerIDs) > 0 {
			if err := s.prCmd.AssignReviewers(ctx, tx, prID, reviewerIDs); err != nil {
				return fmt.Errorf("%s: failed to assign reviewers: %w", op, err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	log.Info("pr created successfully")

	pr.ReviewerIDs = reviewerIDs

	return toAPIPullRequest(pr), nil
}

func (s *PullRequestServiceImpl) MergePR(ctx context.Context, prID string) (*api.PullRequest, error) {
	const op = "internal.service.pullrequest.MergePR"
	log := s.log.With(slog.String("op", op), slog.String("pr_id", prID))

	var (
		pr          *domain.PullRequest
		reviewerIDs []string
	)

	mergedAt := time.Now().UTC()

	err := s.transaction(ctx, op, func(tx *sqlx.Tx) error {
		var err error

		pr, err = s.prCmd.GetPRByIDWithLock(ctx, tx, prID)
		if err != nil {
			return fmt.Errorf("%s: failed to get pr with lock: %w", op, err)
		}

		if pr.Status != api.PullRequestStatusMERGED {
			if err := s.prCmd.UpdatePRStatus(ctx, tx, prID, api.PullRequestStatusMERGED, mergedAt); err != nil {
				return fmt.Errorf("%s: failed to update PR status: %w", op, err)
			}
		}

		reviewerIDs, err = s.prQuery.GetReviewerIDs(ctx, tx, prID)
		if err != nil {
			return fmt.Errorf("%s: failed to get reviewers: %w", op, err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if pr.Status == api.PullRequestStatusMERGED {
		log.Info("PR already merged, returning current state")
	} else {
		log.Info("PR merged successfully")

		pr.Status = api.PullRequestStatusMERGED
		pr.MergedAt = &mergedAt
	}

	pr.ReviewerIDs = reviewerIDs

	return toAPIPullRequest(pr), nil
}

func (s *PullRequestServiceImpl) ReassignReviewer(ctx context.Context, prID string, oldReviewerID string) (*api.ReassignResponse, error) {
	const op = "internal.service.pullrequest.ReassignReviewer"
	log := s.log.With(slog.String("op", op), slog.String("pr_id", prID), slog.String("old_reviewer_id", oldReviewerID))

	var (
		pr                 *domain.PullRequest
		newReviewerID      string
		updatedReviewerIDs []string
	)

	err := s.transaction(ctx, op, func(tx *sqlx.Tx) error {
		var err error

		newReviewerID, pr, err = s.validateAndFindReplacement(ctx, tx, prID, oldReviewerID)
		if err != nil {
			return err
		}

		if err := s.prCmd.ReplaceReviewer(ctx, tx, prID, oldReviewerID, newReviewerID); err != nil {
			return fmt.Errorf("%s: failed to replace reviewer: %w", op, err)
		}

		updatedReviewerIDs, err = s.prQuery.GetReviewerIDs(ctx, tx, prID)
		if err != nil {
			return fmt.Errorf("%s: failed to get updated reviewers: %w", op, err)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	log.Info("reviewer reassigned successfully", slog.String("new_reviewer_id", newReviewerID))

	pr.ReviewerIDs = updatedReviewerIDs

	return &api.ReassignResponse{
		Pr:         *toAPIPullRequest(pr),
		ReplacedBy: newReviewerID,
	}, nil
}

func (s *PullRequestServiceImpl) GetReviewAssignments(ctx context.Context, userID string) (*api.GetReviewResponse, error) {
	const op = "internal.service.pullrequest.GetReviewAssignments"

	prs, err := s.prQuery.GetReviewAssignments(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get review assignments: %w", op, err)
	}

	apiPRs := make([]api.PullRequestShort, len(prs))
	for i, pr := range prs {
		apiPRs[i] = api.PullRequestShort{
			PullRequestId:   pr.ID,
			PullRequestName: pr.Name,
			AuthorId:        pr.AuthorID,
			Status:          api.PullRequestShortStatus(pr.Status),
		}
	}

	return &api.GetReviewResponse{
		UserId:       userID,
		PullRequests: apiPRs,
	}, nil
}

func (s *PullRequestServiceImpl) GetStats(ctx context.Context) (*api.StatsResponse, error) {
	const op = "internal.service.pullrequest.GetStats"

	stats, err := s.prQuery.GetUserStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s: failed to get user stats: %w", op, err)
	}

	userStats := make([]api.UserStats, len(stats))
	for i, stat := range stats {
		userStats[i] = api.UserStats{
			UserId:        stat.UserID,
			Username:      stat.Username,
			OpenReviews:   stat.OpenReviews,
			MergedReviews: stat.MergedReviews,
		}
	}

	return &api.StatsResponse{UserStats: userStats}, nil
}

func (s *PullRequestServiceImpl) transaction(ctx context.Context, op string, fn func(tx *sqlx.Tx) error) error {
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

func (s *PullRequestServiceImpl) validateAndFindReplacement(ctx context.Context, tx *sqlx.Tx, prID, oldReviewerID string) (string, *domain.PullRequest, error) {
	const op = "internal.service.pullrequest.validateAndFindReplacement"

	pr, err := s.prCmd.GetPRByIDWithLock(ctx, tx, prID)
	if err != nil {
		return "", nil, fmt.Errorf("%s: failed to get pr with lock: %w", op, err)
	}

	if pr.Status == api.PullRequestStatusMERGED {
		return "", nil, apperrors.ErrPRMerged
	}

	currentReviewerIDs, err := s.prQuery.GetReviewerIDs(ctx, tx, prID)
	if err != nil {
		return "", nil, fmt.Errorf("%s: failed to get current reviewers: %w", op, err)
	}

	var isAssigned bool

	for _, id := range currentReviewerIDs {
		if id == oldReviewerID {
			isAssigned = true
			break
		}
	}

	if !isAssigned {
		return "", nil, apperrors.ErrReviewerNotAssigned
	}

	teamID, err := s.userPR.GetReviewerTeamID(ctx, oldReviewerID)
	if err != nil {
		return "", nil, fmt.Errorf("%s: failed to get reviewer team: %w", op, err)
	}

	excludedIDs := excludeIDs(pr, currentReviewerIDs)

	newReviewerCandidates, err := s.userPR.GetRandomActiveReviewers(ctx, teamID, excludedIDs, 1)
	if err != nil {
		return "", nil, fmt.Errorf("%s: failed to get random reviewers: %w", op, err)
	}

	if len(newReviewerCandidates) == 0 {
		return "", nil, apperrors.ErrNoCandidate
	}

	return newReviewerCandidates[0], pr, nil
}

func excludeIDs(pr *domain.PullRequest, currentReviewerIDs []string) []string {
	excludeMap := make(map[string]struct{})
	for _, id := range currentReviewerIDs {
		excludeMap[id] = struct{}{}
	}

	excludeMap[pr.AuthorID] = struct{}{}

	excludedIDs := make([]string, 0, len(excludeMap))
	for id := range excludeMap {
		excludedIDs = append(excludedIDs, id)
	}

	return excludedIDs
}

func toAPIPullRequest(pr *domain.PullRequest) *api.PullRequest {
	return &api.PullRequest{
		PullRequestId:     pr.ID,
		PullRequestName:   pr.Name,
		AuthorId:          pr.AuthorID,
		Status:            pr.Status,
		AssignedReviewers: pr.ReviewerIDs,
		CreatedAt:         &pr.CreatedAt,
		MergedAt:          pr.MergedAt,
	}
}
