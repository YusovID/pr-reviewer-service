package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/YusovID/pr-reviewer-service/internal/apperrors"
	"github.com/YusovID/pr-reviewer-service/internal/domain"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func newMockDBAndTx(t *testing.T) (*sqlx.DB, *sqlx.Tx, sqlmock.Sqlmock) {
	t.Helper()

	mockDB, smock, err := sqlmock.New()
	require.NoError(t, err)

	sqlxDB := sqlx.NewDb(mockDB, "sqlmock")

	smock.ExpectBegin()

	tx, err := sqlxDB.Beginx()
	require.NoError(t, err)

	return sqlxDB, tx, smock
}

func TestPullRequestServiceImpl_CreatePR(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	testCases := []struct {
		name          string
		setupMocks    func(transactor *TransactorMock, prCmd *PRCommandRepositoryMock, userPR *UserPRRepositoryMock)
		prID          string
		prName        string
		authorID      string
		expectedPR    *api.PullRequest
		expectedError bool
	}{
		{
			name:     "Success with 2 reviewers",
			prID:     "pr-1",
			prName:   "feat: new logic",
			authorID: "author-1",
			setupMocks: func(transactor *TransactorMock, prCmd *PRCommandRepositoryMock, userPR *UserPRRepositoryMock) {
				_, mockedTx, smock := newMockDBAndTx(t)
				smock.ExpectCommit()

				transactor.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(mockedTx, nil).Once()
				userPR.On("GetAuthorTeamID", ctx, "author-1").Return(1, nil).Once()
				userPR.On("GetRandomActiveReviewers", ctx, 1, []string{"author-1"}, 2).Return([]string{"rev-1", "rev-2"}, nil).Once()
				prCmd.On("CreatePR", ctx, mockedTx, mock.AnythingOfType("*domain.PullRequest")).Return(nil).Once()
				prCmd.On("AssignReviewers", ctx, mockedTx, "pr-1", []string{"rev-1", "rev-2"}).Return(nil).Once()
			},
			expectedPR: &api.PullRequest{
				PullRequestId:     "pr-1",
				PullRequestName:   "feat: new logic",
				AuthorId:          "author-1",
				Status:            "OPEN",
				AssignedReviewers: []string{"rev-1", "rev-2"},
			},
			expectedError: false,
		},
		{
			name:     "Success with 1 reviewer",
			prID:     "pr-2",
			prName:   "fix: bug",
			authorID: "author-2",
			setupMocks: func(transactor *TransactorMock, prCmd *PRCommandRepositoryMock, userPR *UserPRRepositoryMock) {
				_, mockedTx, smock := newMockDBAndTx(t)
				smock.ExpectCommit()

				transactor.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(mockedTx, nil).Once()
				userPR.On("GetAuthorTeamID", ctx, "author-2").Return(2, nil).Once()
				userPR.On("GetRandomActiveReviewers", ctx, 2, []string{"author-2"}, 2).Return([]string{"rev-3"}, nil).Once()
				prCmd.On("CreatePR", ctx, mockedTx, mock.MatchedBy(func(pr *domain.PullRequest) bool {
					return pr.NeedMoreReviewers
				})).Return(nil).Once()
				prCmd.On("AssignReviewers", ctx, mockedTx, "pr-2", []string{"rev-3"}).Return(nil).Once()
			},
			expectedPR: &api.PullRequest{
				PullRequestId:     "pr-2",
				PullRequestName:   "fix: bug",
				AuthorId:          "author-2",
				Status:            "OPEN",
				AssignedReviewers: []string{"rev-3"},
			},
			expectedError: false,
		},
		{
			name:     "Failure on GetAuthorTeamID",
			authorID: "author-3",
			setupMocks: func(transactor *TransactorMock, prCmd *PRCommandRepositoryMock, userPR *UserPRRepositoryMock) {
				userPR.On("GetAuthorTeamID", ctx, "author-3").Return(0, errors.New("db error")).Once()
			},
			expectedError: true,
		},
		{
			name:     "Failure on BeginTxx",
			authorID: "author-4",
			setupMocks: func(transactor *TransactorMock, prCmd *PRCommandRepositoryMock, userPR *UserPRRepositoryMock) {
				userPR.On("GetAuthorTeamID", ctx, "author-4").Return(1, nil).Once()
				userPR.On("GetRandomActiveReviewers", ctx, 1, []string{"author-4"}, 2).Return([]string{"rev-1"}, nil).Once()
				transactor.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(nil, errors.New("cannot begin tx")).Once()
			},
			expectedError: true,
		},
		{
			name:     "Failure on CreatePR in repo",
			prID:     "pr-5",
			authorID: "author-5",
			setupMocks: func(transactor *TransactorMock, prCmd *PRCommandRepositoryMock, userPR *UserPRRepositoryMock) {
				_, mockedTx, smock := newMockDBAndTx(t)
				smock.ExpectRollback()

				transactor.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(mockedTx, nil).Once()
				userPR.On("GetAuthorTeamID", ctx, "author-5").Return(1, nil).Once()
				userPR.On("GetRandomActiveReviewers", ctx, 1, []string{"author-5"}, 2).Return([]string{"rev-1"}, nil).Once()
				prCmd.On("CreatePR", ctx, mockedTx, mock.Anything).Return(errors.New("repo create failed")).Once()
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transactorMock := new(TransactorMock)
			prCmdMock := new(PRCommandRepositoryMock)
			userPRMock := new(UserPRRepositoryMock)
			tc.setupMocks(transactorMock, prCmdMock, userPRMock)

			service := NewPullRequestService(transactorMock, logger, prCmdMock, nil, userPRMock)
			pr, err := service.CreatePR(ctx, tc.prID, tc.prName, tc.authorID)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, pr)
				assert.Equal(t, tc.expectedPR.PullRequestId, pr.PullRequestId)
				assert.Equal(t, tc.expectedPR.PullRequestName, pr.PullRequestName)
				assert.Equal(t, tc.expectedPR.AuthorId, pr.AuthorId)
				assert.Equal(t, tc.expectedPR.Status, pr.Status)
				assert.ElementsMatch(t, tc.expectedPR.AssignedReviewers, pr.AssignedReviewers)
			}

			transactorMock.AssertExpectations(t)
			prCmdMock.AssertExpectations(t)
			userPRMock.AssertExpectations(t)
		})
	}
}

func TestPullRequestServiceImpl_MergePR(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	prID := "pr-to-merge"

	openPR := &domain.PullRequest{
		ID:     prID,
		Status: api.PullRequestStatusOPEN,
	}
	mergedPR := &domain.PullRequest{
		ID:       prID,
		Status:   api.PullRequestStatusMERGED,
		MergedAt: &time.Time{},
	}

	testCases := []struct {
		name          string
		setupMocks    func(transactor *TransactorMock, prCmd *PRCommandRepositoryMock, prQuery *PRQueryRepositoryMock)
		expectedError error
		assertResult  func(t *testing.T, pr *api.PullRequest)
	}{
		{
			name: "Success - Merge an OPEN PR",
			setupMocks: func(transactor *TransactorMock, prCmd *PRCommandRepositoryMock, prQuery *PRQueryRepositoryMock) {
				_, mockedTx, smock := newMockDBAndTx(t)
				smock.ExpectCommit()

				transactor.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(mockedTx, nil).Once()
				prCmd.On("GetPRByIDWithLock", mock.Anything, mockedTx, prID).Return(openPR, nil).Once()
				prCmd.On("UpdatePRStatus", mock.Anything, mockedTx, prID, api.PullRequestStatusMERGED, mock.AnythingOfType("time.Time")).Return(nil).Once()
				prQuery.On("GetReviewerIDs", mock.Anything, mockedTx, prID).Return([]string{"rev1"}, nil).Once()
			},
			assertResult: func(t *testing.T, pr *api.PullRequest) {
				assert.Equal(t, api.PullRequestStatusMERGED, pr.Status)
				assert.NotNil(t, pr.MergedAt)
				assert.Contains(t, pr.AssignedReviewers, "rev1")
			},
		},
		{
			name: "Success - Idempotent call on MERGED PR",
			setupMocks: func(transactor *TransactorMock, prCmd *PRCommandRepositoryMock, prQuery *PRQueryRepositoryMock) {
				_, mockedTx, smock := newMockDBAndTx(t)
				smock.ExpectCommit()

				transactor.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(mockedTx, nil).Once()
				prCmd.On("GetPRByIDWithLock", mock.Anything, mockedTx, prID).Return(mergedPR, nil).Once()
				prQuery.On("GetReviewerIDs", mock.Anything, mockedTx, prID).Return([]string{"rev1"}, nil).Once()
			},
			assertResult: func(t *testing.T, pr *api.PullRequest) {
				assert.Equal(t, api.PullRequestStatusMERGED, pr.Status)
			},
		},
		{
			name: "Failure - PR not found",
			setupMocks: func(transactor *TransactorMock, prCmd *PRCommandRepositoryMock, prQuery *PRQueryRepositoryMock) {
				_, mockedTx, smock := newMockDBAndTx(t)
				smock.ExpectRollback()

				transactor.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(mockedTx, nil).Once()
				prCmd.On("GetPRByIDWithLock", mock.Anything, mockedTx, prID).Return(nil, apperrors.ErrNotFound).Once()
			},
			expectedError: apperrors.ErrNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transactorMock := new(TransactorMock)
			prCmdMock := new(PRCommandRepositoryMock)
			prQueryMock := new(PRQueryRepositoryMock)
			tc.setupMocks(transactorMock, prCmdMock, prQueryMock)

			service := NewPullRequestService(transactorMock, logger, prCmdMock, prQueryMock, nil)
			pr, err := service.MergePR(ctx, prID)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tc.expectedError))
			} else {
				assert.NoError(t, err)
				tc.assertResult(t, pr)
			}

			transactorMock.AssertExpectations(t)
			prCmdMock.AssertExpectations(t)
			prQueryMock.AssertExpectations(t)
		})
	}
}

func TestPullRequestServiceImpl_ReassignReviewer(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	prInDB := &domain.PullRequest{
		ID:       "pr-1",
		AuthorID: "author-1",
		Status:   api.PullRequestStatusOPEN,
	}

	testCases := []struct {
		name             string
		setupMocks       func(transactor *TransactorMock, prCmd *PRCommandRepositoryMock, prQuery *PRQueryRepositoryMock, userPR *UserPRRepositoryMock)
		prID             string
		oldReviewerID    string
		expectedResponse *api.ReassignResponse
		expectedErrorIs  error
	}{
		{
			name:          "Success reassign",
			prID:          "pr-1",
			oldReviewerID: "old-rev",
			setupMocks: func(transactor *TransactorMock, prCmd *PRCommandRepositoryMock, prQuery *PRQueryRepositoryMock, userPR *UserPRRepositoryMock) {
				_, mockedTx, smock := newMockDBAndTx(t)
				smock.ExpectCommit()

				transactor.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(mockedTx, nil).Once()
				prCmd.On("GetPRByIDWithLock", mock.Anything, mockedTx, "pr-1").Return(prInDB, nil).Once()
				prQuery.On("GetReviewerIDs", mock.Anything, mockedTx, "pr-1").Return([]string{"old-rev", "other-rev"}, nil).Once()
				userPR.On("GetReviewerTeamID", ctx, "old-rev").Return(1, nil).Once()
				userPR.On("GetRandomActiveReviewers", ctx, 1, mock.Anything, 1).Return([]string{"new-rev"}, nil).Once()
				prCmd.On("ReplaceReviewer", mock.Anything, mockedTx, "pr-1", "old-rev", "new-rev").Return(nil).Once()
				prQuery.On("GetReviewerIDs", mock.Anything, mockedTx, "pr-1").Return([]string{"new-rev", "other-rev"}, nil).Once()
			},
			expectedResponse: &api.ReassignResponse{
				Pr: api.PullRequest{
					PullRequestId:     "pr-1",
					AuthorId:          "author-1",
					Status:            api.PullRequestStatusOPEN,
					AssignedReviewers: []string{"new-rev", "other-rev"},
				},
				ReplacedBy: "new-rev",
			},
		},
		{
			name:          "Failure - PR is merged",
			prID:          "pr-merged",
			oldReviewerID: "old-rev",
			setupMocks: func(transactor *TransactorMock, prCmd *PRCommandRepositoryMock, prQuery *PRQueryRepositoryMock, userPR *UserPRRepositoryMock) {
				_, mockedTx, smock := newMockDBAndTx(t)
				smock.ExpectRollback()

				transactor.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(mockedTx, nil).Once()
				mergedPR := *prInDB
				mergedPR.ID = "pr-merged"
				mergedPR.Status = api.PullRequestStatusMERGED
				prCmd.On("GetPRByIDWithLock", mock.Anything, mockedTx, "pr-merged").Return(&mergedPR, nil).Once()
			},
			expectedErrorIs: apperrors.ErrPRMerged,
		},
		{
			name:          "Failure - Reviewer not assigned",
			prID:          "pr-1",
			oldReviewerID: "not-assigned-rev",
			setupMocks: func(transactor *TransactorMock, prCmd *PRCommandRepositoryMock, prQuery *PRQueryRepositoryMock, userPR *UserPRRepositoryMock) {
				_, mockedTx, smock := newMockDBAndTx(t)
				smock.ExpectRollback()

				transactor.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(mockedTx, nil).Once()
				prCmd.On("GetPRByIDWithLock", mock.Anything, mockedTx, "pr-1").Return(prInDB, nil).Once()
				prQuery.On("GetReviewerIDs", mock.Anything, mockedTx, "pr-1").Return([]string{"old-rev", "other-rev"}, nil).Once()
			},
			expectedErrorIs: apperrors.ErrReviewerNotAssigned,
		},
		{
			name:          "Failure - No candidate for replacement",
			prID:          "pr-1",
			oldReviewerID: "old-rev",
			setupMocks: func(transactor *TransactorMock, prCmd *PRCommandRepositoryMock, prQuery *PRQueryRepositoryMock, userPR *UserPRRepositoryMock) {
				_, mockedTx, smock := newMockDBAndTx(t)
				smock.ExpectRollback()

				transactor.On("BeginTxx", mock.Anything, (*sql.TxOptions)(nil)).Return(mockedTx, nil).Once()
				prCmd.On("GetPRByIDWithLock", mock.Anything, mockedTx, "pr-1").Return(prInDB, nil).Once()
				prQuery.On("GetReviewerIDs", mock.Anything, mockedTx, "pr-1").Return([]string{"old-rev", "other-rev"}, nil).Once()
				userPR.On("GetReviewerTeamID", ctx, "old-rev").Return(1, nil).Once()
				userPR.On("GetRandomActiveReviewers", ctx, 1, mock.Anything, 1).Return([]string{}, nil).Once()
			},
			expectedErrorIs: apperrors.ErrNoCandidate,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			transactorMock := new(TransactorMock)
			prCmdMock := new(PRCommandRepositoryMock)
			prQueryMock := new(PRQueryRepositoryMock)
			userPRMock := new(UserPRRepositoryMock)
			tc.setupMocks(transactorMock, prCmdMock, prQueryMock, userPRMock)

			service := NewPullRequestService(transactorMock, logger, prCmdMock, prQueryMock, userPRMock)
			resp, err := service.ReassignReviewer(ctx, tc.prID, tc.oldReviewerID)

			if tc.expectedErrorIs != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tc.expectedErrorIs))
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, tc.expectedResponse.ReplacedBy, resp.ReplacedBy)
				assert.ElementsMatch(t, tc.expectedResponse.Pr.AssignedReviewers, resp.Pr.AssignedReviewers)
			}

			transactorMock.AssertExpectations(t)
			prCmdMock.AssertExpectations(t)
			prQueryMock.AssertExpectations(t)
			userPRMock.AssertExpectations(t)
		})
	}
}

func TestPullRequestServiceImpl_GetReviewAssignments(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	userID := "user-with-reviews"

	testCases := []struct {
		name          string
		setupMocks    func(prQuery *PRQueryRepositoryMock)
		expectedResp  *api.GetReviewResponse
		expectedError bool
	}{
		{
			name: "Success - User has assignments",
			setupMocks: func(prQuery *PRQueryRepositoryMock) {
				prs := []domain.PullRequest{
					{ID: "pr-1", Name: "Feature A", AuthorID: "author-A", Status: api.PullRequestStatusOPEN},
					{ID: "pr-2", Name: "Fix B", AuthorID: "author-B", Status: api.PullRequestStatusMERGED},
				}
				prQuery.On("GetReviewAssignments", ctx, userID).Return(prs, nil).Once()
			},
			expectedResp: &api.GetReviewResponse{
				UserId: userID,
				PullRequests: []api.PullRequestShort{
					{PullRequestId: "pr-1", PullRequestName: "Feature A", AuthorId: "author-A", Status: "OPEN"},
					{PullRequestId: "pr-2", PullRequestName: "Fix B", AuthorId: "author-B", Status: "MERGED"},
				},
			},
		},
		{
			name: "Success - User has no assignments",
			setupMocks: func(prQuery *PRQueryRepositoryMock) {
				prQuery.On("GetReviewAssignments", ctx, userID).Return([]domain.PullRequest{}, nil).Once()
			},
			expectedResp: &api.GetReviewResponse{
				UserId:       userID,
				PullRequests: []api.PullRequestShort{},
			},
		},
		{
			name: "Failure - Repository returns error",
			setupMocks: func(prQuery *PRQueryRepositoryMock) {
				prQuery.On("GetReviewAssignments", ctx, userID).Return(nil, errors.New("database connection failed")).Once()
			},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prQueryMock := new(PRQueryRepositoryMock)
			tc.setupMocks(prQueryMock)

			service := NewPullRequestService(nil, logger, nil, prQueryMock, nil)
			resp, err := service.GetReviewAssignments(ctx, userID)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expectedResp.UserId, resp.UserId)
				assert.ElementsMatch(t, tc.expectedResp.PullRequests, resp.PullRequests)
			}

			prQueryMock.AssertExpectations(t)
		})
	}
}

func TestPullRequestServiceImpl_GetStats(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	prQueryMock := new(PRQueryRepositoryMock)

	service := NewPullRequestService(nil, logger, nil, prQueryMock, nil)

	domainStats := []domain.Stats{
		{UserID: "u1", Username: "Alice", OpenReviews: 1, MergedReviews: 10},
	}
	prQueryMock.On("GetUserStats", ctx).Return(domainStats, nil).Once()

	statsResp, err := service.GetStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, statsResp)
	require.Len(t, statsResp.UserStats, 1)
	assert.Equal(t, "u1", statsResp.UserStats[0].UserId)
	assert.Equal(t, 10, statsResp.UserStats[0].MergedReviews)
	prQueryMock.AssertExpectations(t)

	prQueryMock.On("GetUserStats", ctx).Return(nil, errors.New("db error")).Once()

	_, err = service.GetStats(ctx)
	require.Error(t, err)
	prQueryMock.AssertExpectations(t)
}
