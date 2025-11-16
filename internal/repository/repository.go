package repository

import (
	"context"
	"time"

	"github.com/YusovID/pr-reviewer-service/internal/domain"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/jmoiron/sqlx"
)

type TeamRepository interface {
	CreateTeamWithUsers(ctx context.Context, team api.Team) (*domain.TeamWithMembers, error)
	GetTeamByName(ctx context.Context, ext sqlx.ExtContext, name string) (*domain.TeamWithMembers, error)
}

type UserRepository interface {
	SetIsActive(ctx context.Context, userID string, isActive bool) (*api.User, error)
	DeactivateUsersByTeamID(ctx context.Context, tx *sqlx.Tx, teamID int) ([]string, error)
}

type PRQueryRepository interface {
	GetPRByID(ctx context.Context, prID string) (*domain.PullRequest, error)
	GetPRByIDWithReviewers(ctx context.Context, prID string) (*domain.PullRequest, error)
	GetReviewerIDs(ctx context.Context, ext sqlx.ExtContext, prID string) ([]string, error)
	GetReviewAssignments(ctx context.Context, userID string) ([]domain.PullRequest, error)
	GetUserStats(ctx context.Context) ([]domain.Stats, error)
	GetOpenPRsByReviewers(ctx context.Context, tx *sqlx.Tx, userIDs []string) ([]domain.PullRequest, error)
}

type PRCommandRepository interface {
	CreatePR(ctx context.Context, tx *sqlx.Tx, pr *domain.PullRequest) error
	AssignReviewers(ctx context.Context, tx *sqlx.Tx, prID string, reviewerIDs []string) error
	GetPRByIDWithLock(ctx context.Context, tx *sqlx.Tx, prID string) (*domain.PullRequest, error)
	UpdatePRStatus(ctx context.Context, tx *sqlx.Tx, prID string, status api.PullRequestStatus, mergedAt time.Time) error
	ReplaceReviewer(ctx context.Context, tx *sqlx.Tx, prID string, oldReviewerID string, newReviewerID string) error
}

type UserPRRepository interface {
	GetAuthorTeamID(ctx context.Context, authorID string) (int, error)
	GetReviewerTeamID(ctx context.Context, reviewerID string) (int, error)
	GetRandomActiveReviewers(ctx context.Context, teamID int, excludeUserIDs []string, count int) ([]string, error)
}
