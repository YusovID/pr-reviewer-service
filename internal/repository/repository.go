// package repository defines the interfaces for the data persistence layer.
// These interfaces abstract the underlying database implementation from the service layer.
package repository

import (
	"context"
	"time"

	"github.com/YusovID/pr-reviewer-service/internal/domain"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/jmoiron/sqlx"
)

// TeamRepository defines the contract for interacting with team and user data.
type TeamRepository interface {
	// CreateTeamWithUsers creates a new team and upserts its members.
	// This operation is expected to be transactional.
	// It returns apperrors.ErrAlreadyExists if a team with the same name already exists.
	CreateTeamWithUsers(ctx context.Context, team api.Team) (*domain.TeamWithMembers, error)

	// GetTeamByName retrieves a team by its unique name, along with its list of members.
	// The ext argument allows this method to be executed within a transaction (*sqlx.Tx)
	// or directly on a DB connection (*sqlx.DB).
	// It returns apperrors.ErrNotFound if the team is not found.
	GetTeamByName(ctx context.Context, ext sqlx.ExtContext, name string) (*domain.TeamWithMembers, error)
}

// UserRepository defines the contract for user-specific data operations.
type UserRepository interface {
	// SetIsActive updates the active status of a user.
	// It returns apperrors.ErrNotFound if the user does not exist.
	SetIsActive(ctx context.Context, userID string, isActive bool) (*api.User, error)

	// DeactivateUsersByTeamID deactivates all active users belonging to a specific team ID.
	// This method is intended to be run within a transaction and returns the IDs of the deactivated users.
	DeactivateUsersByTeamID(ctx context.Context, tx *sqlx.Tx, teamID int) ([]string, error)
}

// PRQueryRepository defines the contract for read-only pull request operations, following the CQRS pattern.
type PRQueryRepository interface {
	// GetPRByID retrieves a single pull request by its ID without its reviewers.
	// Returns apperrors.ErrNotFound if the PR is not found.
	GetPRByID(ctx context.Context, prID string) (*domain.PullRequest, error)

	// GetPRByIDWithReviewers retrieves a pull request and its assigned reviewers.
	// Returns apperrors.ErrNotFound if the PR is not found.
	GetPRByIDWithReviewers(ctx context.Context, prID string) (*domain.PullRequest, error)

	// GetReviewerIDs retrieves the IDs of all reviewers for a given pull request.
	// The ext argument allows this method to be executed within a transaction or on a direct DB connection.
	GetReviewerIDs(ctx context.Context, ext sqlx.ExtContext, prID string) ([]string, error)

	// GetReviewAssignments retrieves all pull requests assigned to a specific user for review.
	GetReviewAssignments(ctx context.Context, userID string) ([]domain.PullRequest, error)

	// GetUserStats retrieves review statistics for all users.
	GetUserStats(ctx context.Context) ([]domain.Stats, error)

	// GetOpenPRsByReviewers finds all open pull requests where any of the specified user IDs are reviewers.
	// This method is intended for transactional use to ensure data consistency during reassignments.
	GetOpenPRsByReviewers(ctx context.Context, tx *sqlx.Tx, userIDs []string) ([]domain.PullRequest, error)
}

// PRCommandRepository defines the contract for write and locking operations on pull requests, following the CQRS pattern.
// All methods are expected to be executed within a transaction.
type PRCommandRepository interface {
	// CreatePR inserts a new pull request record.
	// It returns apperrors.ErrAlreadyExists if a PR with the same ID already exists.
	CreatePR(ctx context.Context, tx *sqlx.Tx, pr *domain.PullRequest) error

	// AssignReviewers associates a list of reviewers with a pull request.
	AssignReviewers(ctx context.Context, tx *sqlx.Tx, prID string, reviewerIDs []string) error

	// GetPRByIDWithLock retrieves a pull request by its ID and acquires a row-level lock ("FOR UPDATE").
	// This prevents concurrent modifications to the PR record within the transaction.
	// It returns apperrors.ErrNotFound if the PR is not found.
	GetPRByIDWithLock(ctx context.Context, tx *sqlx.Tx, prID string) (*domain.PullRequest, error)

	// UpdatePRStatus updates the status and potentially the merged_at timestamp of a pull request.
	UpdatePRStatus(ctx context.Context, tx *sqlx.Tx, prID string, status api.PullRequestStatus, mergedAt time.Time) error

	// ReplaceReviewer atomically replaces an old reviewer with a new one for a specific pull request.
	ReplaceReviewer(ctx context.Context, tx *sqlx.Tx, prID string, oldReviewerID string, newReviewerID string) error
}

// UserPRRepository defines a contract for operations that cross the User and PullRequest domains,
// typically used by services to gather information for business logic decisions.
type UserPRRepository interface {
	// GetAuthorTeamID returns the team ID for a given user ID (author).
	// It returns apperrors.ErrNotFound if the user is not found.
	GetAuthorTeamID(ctx context.Context, authorID string) (int, error)

	// GetReviewerTeamID returns the team ID for a given user ID (reviewer).
	// It returns apperrors.ErrNotFound if the user is not found.
	GetReviewerTeamID(ctx context.Context, reviewerID string) (int, error)

	// GetRandomActiveReviewers selects a specified number of random, active reviewers from a team,
	// excluding a list of provided user IDs.
	GetRandomActiveReviewers(ctx context.Context, teamID int, excludeUserIDs []string, count int) ([]string, error)
}
