package service

import (
	"context"
	"database/sql"
	"time"

	"github.com/YusovID/pr-reviewer-service/internal/domain"
	"github.com/YusovID/pr-reviewer-service/internal/repository"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/mock"
)

type TeamRepositoryMock struct {
	mock.Mock
}

var _ repository.TeamRepository = (*TeamRepositoryMock)(nil)

func (m *TeamRepositoryMock) GetTeamByName(ctx context.Context, ext sqlx.ExtContext, name string) (*domain.TeamWithMembers, error) {
	args := m.Called(ctx, ext, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*domain.TeamWithMembers), args.Error(1)
}

func (m *TeamRepositoryMock) CreateTeamWithUsers(ctx context.Context, team api.Team) (*domain.TeamWithMembers, error) {
	args := m.Called(ctx, team)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TeamWithMembers), args.Error(1)
}

type UserRepositoryMock struct {
	mock.Mock
}

var _ repository.UserRepository = (*UserRepositoryMock)(nil)

func (m *UserRepositoryMock) SetIsActive(ctx context.Context, userID string, isActive bool) (*api.User, error) {
	args := m.Called(ctx, userID, isActive)

	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*api.User), args.Error(1)
}

type TxMock struct {
	mock.Mock
	sqlx.ExtContext
}

func (m *TxMock) Commit() error {
	args := m.Called()
	return args.Error(0)
}
func (m *TxMock) Rollback() error {
	args := m.Called()
	return args.Error(0)
}

type PRCommandRepositoryMock struct {
	mock.Mock
}

func (m *PRCommandRepositoryMock) CreatePR(ctx context.Context, tx *sqlx.Tx, pr *domain.PullRequest) error {
	args := m.Called(ctx, tx, pr)
	return args.Error(0)
}
func (m *PRCommandRepositoryMock) AssignReviewers(ctx context.Context, tx *sqlx.Tx, prID string, reviewerIDs []string) error {
	args := m.Called(ctx, tx, prID, reviewerIDs)
	return args.Error(0)
}
func (m *PRCommandRepositoryMock) GetPRByIDWithLock(ctx context.Context, tx *sqlx.Tx, prID string) (*domain.PullRequest, error) {
	args := m.Called(ctx, tx, prID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*domain.PullRequest), args.Error(1)
}
func (m *PRCommandRepositoryMock) UpdatePRStatus(ctx context.Context, tx *sqlx.Tx, prID string, status api.PullRequestStatus, mergedAt time.Time) error {
	args := m.Called(ctx, tx, prID, status, mergedAt)
	return args.Error(0)
}
func (m *PRCommandRepositoryMock) ReplaceReviewer(ctx context.Context, tx *sqlx.Tx, prID string, oldReviewerID string, newReviewerID string) error {
	args := m.Called(ctx, tx, prID, oldReviewerID, newReviewerID)
	return args.Error(0)
}

type PRQueryRepositoryMock struct {
	mock.Mock
}

func (m *PRQueryRepositoryMock) GetPRByID(ctx context.Context, prID string) (*domain.PullRequest, error) {
	args := m.Called(ctx, prID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*domain.PullRequest), args.Error(1)
}
func (m *PRQueryRepositoryMock) GetPRByIDWithReviewers(ctx context.Context, prID string) (*domain.PullRequest, error) {
	args := m.Called(ctx, prID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*domain.PullRequest), args.Error(1)
}
func (m *PRQueryRepositoryMock) GetReviewerIDs(ctx context.Context, ext sqlx.ExtContext, prID string) ([]string, error) {
	args := m.Called(ctx, ext, prID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]string), args.Error(1)
}
func (m *PRQueryRepositoryMock) GetReviewAssignments(ctx context.Context, userID string) ([]domain.PullRequest, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]domain.PullRequest), args.Error(1)
}

func (m *PRQueryRepositoryMock) GetOpenPRsByReviewers(ctx context.Context, tx *sqlx.Tx, userIDs []string) ([]domain.PullRequest, error) {
	args := m.Called(ctx, tx, userIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PullRequest), args.Error(1)
}

type UserPRRepositoryMock struct {
	mock.Mock
}

func (m *UserPRRepositoryMock) GetAuthorTeamID(ctx context.Context, authorID string) (int, error) {
	args := m.Called(ctx, authorID)
	return args.Int(0), args.Error(1)
}
func (m *UserPRRepositoryMock) GetReviewerTeamID(ctx context.Context, reviewerID string) (int, error) {
	args := m.Called(ctx, reviewerID)
	return args.Int(0), args.Error(1)
}
func (m *UserPRRepositoryMock) GetRandomActiveReviewers(ctx context.Context, teamID int, excludeUserIDs []string, count int) ([]string, error) {
	args := m.Called(ctx, teamID, excludeUserIDs, count)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]string), args.Error(1)
}

type TransactorMock struct {
	mock.Mock
}

func (m *TransactorMock) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	var tx *sqlx.Tx

	args := m.Called(ctx, opts)
	if args.Get(0) != nil {
		tx = args.Get(0).(*sqlx.Tx)
	}

	return tx, args.Error(1)
}

func (m *PRQueryRepositoryMock) GetUserStats(ctx context.Context) ([]domain.Stats, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).([]domain.Stats), args.Error(1)
}

func (m *UserRepositoryMock) DeactivateUsersByTeamID(ctx context.Context, tx *sqlx.Tx, teamID int) ([]string, error) {
	args := m.Called(ctx, tx, teamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}
