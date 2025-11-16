package service

import (
	"context"
	"database/sql"
	"log/slog"
	"os"
	"testing"

	"github.com/YusovID/pr-reviewer-service/internal/apperrors"
	"github.com/YusovID/pr-reviewer-service/internal/domain"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Helper struct to hold all mocks, defined at the package level
type mocks struct {
	userRepo    *UserRepositoryMock
	teamRepo    *TeamRepositoryMock
	prQueryRepo *PRQueryRepositoryMock
	prCmdRepo   *PRCommandRepositoryMock
	userPRRepo  *UserPRRepositoryMock
	transactor  *TransactorMock
}

func TestUserServiceImpl_SetIsActive(t *testing.T) {
	ctx := context.Background()
	testUserID := "u1"
	expectedUser := &api.User{
		UserId:   testUserID,
		Username: "Test User",
		TeamName: "backend",
		IsActive: false,
	}

	testCases := []struct {
		name          string
		setupMock     func(repoMock *UserRepositoryMock)
		userID        string
		isActive      bool
		expectedUser  *api.User
		expectedError bool
	}{
		{
			name: "Success: User status is updated",
			setupMock: func(repoMock *UserRepositoryMock) {
				repoMock.On(
					"SetIsActive",
					mock.Anything,
					testUserID,
					false,
				).Return(expectedUser, nil)
			},
			userID:        testUserID,
			isActive:      false,
			expectedUser:  expectedUser,
			expectedError: false,
		},
		{
			name: "Failure: User not found in repository",
			setupMock: func(repoMock *UserRepositoryMock) {
				repoMock.On(
					"SetIsActive",
					mock.Anything,
					testUserID,
					false,
				).Return(nil, apperrors.ErrNotFound)
			},
			userID:        testUserID,
			isActive:      false,
			expectedUser:  nil,
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := new(UserRepositoryMock)
			tc.setupMock(repoMock)

			service := NewUserService(repoMock, nil, nil, nil, nil, nil, slog.Default())

			resultUser, err := service.SetIsActive(ctx, tc.userID, tc.isActive)

			assert.Equal(t, tc.expectedUser, resultUser)

			if tc.expectedError {
				assert.Error(t, err)
				assert.ErrorIs(t, err, apperrors.ErrNotFound)
			} else {
				assert.NoError(t, err)
			}

			repoMock.AssertExpectations(t)
		})
	}
}

func TestUserServiceImpl_DeactivateTeam(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	teamInDB := &domain.TeamWithMembers{
		ID:   1,
		Name: "test-team",
	}
	deactivatedUserIDs := []string{"u1", "u2"}

	testCases := []struct {
		name                     string
		teamName                 string
		setupMocks               func(m *mocks)
		expectedDeactivatedCount int
		expectedReassignedCount  int
		expectedError            error
	}{
		{
			name:     "Success: Deactivate users and reassign PRs",
			teamName: "test-team",
			setupMocks: func(m *mocks) {
				prsToReassign := []domain.PullRequest{
					{ID: "pr-1", AuthorID: "author-1", ReviewerIDs: []string{"u1", "u3"}},
				}

				_, tx, smock := newMockDBAndTx(t)
				smock.ExpectCommit()
				m.transactor.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(tx, nil)
				m.teamRepo.On("GetTeamByName", ctx, mock.Anything, "test-team").Return(teamInDB, nil)
				m.userRepo.On("DeactivateUsersByTeamID", ctx, mock.Anything, 1).Return(deactivatedUserIDs, nil)
				m.prQueryRepo.On("GetOpenPRsByReviewers", ctx, mock.Anything, mock.Anything).Return(prsToReassign, nil)
				m.userPRRepo.On("GetRandomActiveReviewers", ctx, 1, mock.Anything, 1).Return([]string{"new-rev"}, nil)
				m.prCmdRepo.On("ReplaceReviewer", ctx, mock.Anything, "pr-1", "u1", "new-rev").Return(nil)
			},
			expectedDeactivatedCount: 2,
			expectedReassignedCount:  1,
			expectedError:            nil,
		},
		{
			name:     "Failure: Team not found",
			teamName: "unknown-team",
			setupMocks: func(m *mocks) {
				_, tx, smock := newMockDBAndTx(t)
				smock.ExpectRollback()
				m.transactor.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(tx, nil)
				m.teamRepo.On("GetTeamByName", ctx, mock.Anything, "unknown-team").Return(nil, apperrors.ErrNotFound)
			},
			expectedError: apperrors.ErrNotFound,
		},
		{
			name:     "Success: No active users to deactivate",
			teamName: "test-team",
			setupMocks: func(m *mocks) {
				_, tx, smock := newMockDBAndTx(t)
				smock.ExpectCommit()
				m.transactor.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(tx, nil)
				m.teamRepo.On("GetTeamByName", ctx, mock.Anything, "test-team").Return(teamInDB, nil)
				m.userRepo.On("DeactivateUsersByTeamID", ctx, mock.Anything, 1).Return([]string{}, nil)
			},
			expectedDeactivatedCount: 0,
			expectedReassignedCount:  0,
		},
		{
			name:     "Success: No open PRs to reassign",
			teamName: "test-team",
			setupMocks: func(m *mocks) {
				_, tx, smock := newMockDBAndTx(t)
				smock.ExpectCommit()
				m.transactor.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(tx, nil)
				m.teamRepo.On("GetTeamByName", ctx, mock.Anything, "test-team").Return(teamInDB, nil)
				m.userRepo.On("DeactivateUsersByTeamID", ctx, mock.Anything, 1).Return(deactivatedUserIDs, nil)
				m.prQueryRepo.On("GetOpenPRsByReviewers", ctx, mock.Anything, mock.Anything).Return([]domain.PullRequest{}, nil)
			},
			expectedDeactivatedCount: 2,
			expectedReassignedCount:  0,
		},
		{
			name:     "Success: No replacement candidate found",
			teamName: "test-team",
			setupMocks: func(m *mocks) {
				prsToReassign := []domain.PullRequest{
					{ID: "pr-1", AuthorID: "author-1", ReviewerIDs: []string{"u1", "u3"}},
				}

				_, tx, smock := newMockDBAndTx(t)
				smock.ExpectCommit()
				m.transactor.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(tx, nil)
				m.teamRepo.On("GetTeamByName", ctx, mock.Anything, "test-team").Return(teamInDB, nil)
				m.userRepo.On("DeactivateUsersByTeamID", ctx, mock.Anything, 1).Return(deactivatedUserIDs, nil)
				m.prQueryRepo.On("GetOpenPRsByReviewers", ctx, mock.Anything, mock.Anything).Return(prsToReassign, nil)
				m.userPRRepo.On("GetRandomActiveReviewers", ctx, 1, mock.Anything, 1).Return([]string{}, nil)
			},
			expectedDeactivatedCount: 2,
			expectedReassignedCount:  1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := &mocks{
				userRepo:    new(UserRepositoryMock),
				teamRepo:    new(TeamRepositoryMock),
				prQueryRepo: new(PRQueryRepositoryMock),
				prCmdRepo:   new(PRCommandRepositoryMock),
				userPRRepo:  new(UserPRRepositoryMock),
				transactor:  new(TransactorMock),
			}
			tc.setupMocks(m)

			service := NewUserService(m.userRepo, m.teamRepo, m.prQueryRepo, m.prCmdRepo, m.userPRRepo, m.transactor, logger)

			deactivated, reassigned, err := service.DeactivateTeam(ctx, tc.teamName)

			assert.Equal(t, tc.expectedDeactivatedCount, deactivated)
			assert.Equal(t, tc.expectedReassignedCount, reassigned)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tc.expectedError)
			} else {
				assert.NoError(t, err)
			}

			m.userRepo.AssertExpectations(t)
			m.teamRepo.AssertExpectations(t)
			m.prQueryRepo.AssertExpectations(t)
			m.prCmdRepo.AssertExpectations(t)
			m.userPRRepo.AssertExpectations(t)
			m.transactor.AssertExpectations(t)
		})
	}
}
