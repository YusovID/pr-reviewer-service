package http

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/YusovID/pr-reviewer-service/internal/apperrors"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestServer_PostTeamAdd(t *testing.T) {
	inputTeam := api.Team{
		TeamName: "backend",
		Members:  []api.TeamMember{{UserId: "u1", Username: "Alice", IsActive: true}},
	}

	testCases := []struct {
		name                 string
		requestBody          string
		setupMocks           func(*TeamServiceMock)
		expectedStatusCode   int
		expectedResponseBody string
	}{
		{
			name:        "Success",
			requestBody: `{"team_name": "backend", "members": [{"user_id": "u1", "username": "Alice", "is_active": true}]}`,
			setupMocks: func(tsm *TeamServiceMock) {
				tsm.On("CreateTeam", mock.Anything, mock.MatchedBy(func(team api.Team) bool {
					return team.TeamName == inputTeam.TeamName
				})).Return(&inputTeam, nil).Once()
			},
			expectedStatusCode:   http.StatusCreated,
			expectedResponseBody: `{"team":{"team_name":"backend","members":[{"is_active":true,"user_id":"u1","username":"Alice"}]}}`,
		},
		{
			name:        "Service Error - Already Exists",
			requestBody: `{"team_name": "backend", "members": []}`,
			setupMocks: func(tsm *TeamServiceMock) {
				tsm.On("CreateTeam", mock.Anything, mock.Anything).Return(nil, &apperrors.TeamAlreadyExistsError{TeamName: "backend"}).Once()
			},
			expectedStatusCode:   http.StatusConflict,
			expectedResponseBody: `{"error":{"code":"TEAM_EXISTS","message":"team with this name already exists"}}`,
		},
		{
			name:                 "Invalid JSON Body",
			requestBody:          `{invalid json}`,
			setupMocks:           func(tsm *TeamServiceMock) {},
			expectedStatusCode:   http.StatusBadRequest,
			expectedResponseBody: `{"error": "invalid request body"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			teamServiceMock := new(TeamServiceMock)
			tc.setupMocks(teamServiceMock)

			server := NewServer(slog.New(slog.NewJSONHandler(os.Stdout, nil)), teamServiceMock, nil, nil)
			req := httptest.NewRequest(http.MethodPost, "/team/add", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			server.PostTeamAdd(rr, req)

			assert.Equal(t, tc.expectedStatusCode, rr.Code)
			assert.JSONEq(t, tc.expectedResponseBody, rr.Body.String())
			teamServiceMock.AssertExpectations(t)
		})
	}
}

func TestServer_PostPullRequestCreate(t *testing.T) {
	now := time.Now()
	createdPR := &api.PullRequest{
		PullRequestId:     "pr-1",
		PullRequestName:   "New Feature",
		AuthorId:          "author-1",
		Status:            api.PullRequestStatusOPEN,
		AssignedReviewers: []string{"reviewer-1"},
		CreatedAt:         &now,
	}

	testCases := []struct {
		name                 string
		requestBody          string
		setupMocks           func(*PullRequestServiceMock)
		expectedStatusCode   int
		expectedResponseBody string
	}{
		{
			name:        "Success",
			requestBody: `{"pull_request_id": "pr-1", "pull_request_name": "New Feature", "author_id": "author-1"}`,
			setupMocks: func(prsm *PullRequestServiceMock) {
				prsm.On("CreatePR", mock.Anything, "pr-1", "New Feature", "author-1").
					Return(createdPR, nil).Once()
			},
			expectedStatusCode: http.StatusCreated,
			expectedResponseBody: `{
				"pr": {
					"pull_request_id": "pr-1",
					"pull_request_name": "New Feature",
					"author_id": "author-1",
					"status": "OPEN",
					"assigned_reviewers": ["reviewer-1"],
					"createdAt": "` + now.Format(time.RFC3339Nano) + `",
					"mergedAt": null
				}
			}`,
		},
		{
			name:        "Service Error - Author Not Found",
			requestBody: `{"pull_request_id": "pr-1", "pull_request_name": "New Feature", "author_id": "author-not-found"}`,
			setupMocks: func(prsm *PullRequestServiceMock) {
				prsm.On("CreatePR", mock.Anything, "pr-1", "New Feature", "author-not-found").
					Return(nil, apperrors.ErrNotFound).Once()
			},
			expectedStatusCode:   http.StatusNotFound,
			expectedResponseBody: `{"error":{"code":"NOT_FOUND","message":"resource not found"}}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prServiceMock := new(PullRequestServiceMock)
			tc.setupMocks(prServiceMock)

			server := NewServer(slog.New(slog.NewJSONHandler(os.Stdout, nil)), nil, nil, prServiceMock)
			req := httptest.NewRequest(http.MethodPost, "/pullRequest/create", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			server.PostPullRequestCreate(rr, req)

			assert.Equal(t, tc.expectedStatusCode, rr.Code)
			assert.JSONEq(t, tc.expectedResponseBody, rr.Body.String())
			prServiceMock.AssertExpectations(t)
		})
	}
}

func TestServer_PostPullRequestReassign(t *testing.T) {
	reassignedResponse := &api.ReassignResponse{
		Pr: api.PullRequest{
			PullRequestId:     "pr-123",
			AssignedReviewers: []string{"new-reviewer"},
		},
		ReplacedBy: "new-reviewer",
	}

	testCases := []struct {
		name                 string
		requestBody          string
		setupMocks           func(*PullRequestServiceMock)
		expectedStatusCode   int
		expectedResponseBody string
	}{
		{
			name:        "Success",
			requestBody: `{"pull_request_id": "pr-123", "old_user_id": "old-reviewer"}`,
			setupMocks: func(prsm *PullRequestServiceMock) {
				prsm.On("ReassignReviewer", mock.Anything, "pr-123", "old-reviewer").
					Return(reassignedResponse, nil).Once()
			},
			expectedStatusCode:   http.StatusOK,
			expectedResponseBody: `{"pr":{"pull_request_id":"pr-123","pull_request_name":"","author_id":"","status":"","assigned_reviewers":["new-reviewer"],"createdAt":null,"mergedAt":null},"replaced_by":"new-reviewer"}`,
		},
		{
			name:        "Service Error - PR Merged",
			requestBody: `{"pull_request_id": "pr-123", "old_user_id": "old-reviewer"}`,
			setupMocks: func(prsm *PullRequestServiceMock) {
				prsm.On("ReassignReviewer", mock.Anything, "pr-123", "old-reviewer").
					Return(nil, apperrors.ErrPRMerged).Once()
			},
			expectedStatusCode:   http.StatusConflict,
			expectedResponseBody: `{"error":{"code":"PR_MERGED","message":"cannot modify merged pull request"}}`,
		},
		{
			name:        "Service Error - Not Assigned",
			requestBody: `{"pull_request_id": "pr-123", "old_user_id": "not-a-reviewer"}`,
			setupMocks: func(prsm *PullRequestServiceMock) {
				prsm.On("ReassignReviewer", mock.Anything, "pr-123", "not-a-reviewer").
					Return(nil, apperrors.ErrReviewerNotAssigned).Once()
			},
			expectedStatusCode:   http.StatusConflict,
			expectedResponseBody: `{"error":{"code":"NOT_ASSIGNED","message":"reviewer is not assigned to this PR"}}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prServiceMock := new(PullRequestServiceMock)
			tc.setupMocks(prServiceMock)

			server := NewServer(slog.New(slog.NewJSONHandler(os.Stdout, nil)), nil, nil, prServiceMock)
			req := httptest.NewRequest(http.MethodPost, "/pullRequest/reassign", strings.NewReader(tc.requestBody))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			server.PostPullRequestReassign(rr, req)

			assert.Equal(t, tc.expectedStatusCode, rr.Code)
			assert.JSONEq(t, tc.expectedResponseBody, rr.Body.String())
			prServiceMock.AssertExpectations(t)
		})
	}
}

func TestServer_GetUsersGetReview(t *testing.T) {
	reviewResponse := &api.GetReviewResponse{
		UserId: "user-1",
		PullRequests: []api.PullRequestShort{
			{PullRequestId: "pr-1", PullRequestName: "Feature A", AuthorId: "author-A", Status: "OPEN"},
		},
	}

	testCases := []struct {
		name                 string
		targetURL            string
		setupMocks           func(*PullRequestServiceMock)
		expectedStatusCode   int
		expectedResponseBody string
	}{
		{
			name:      "Success",
			targetURL: "/users/getReview?user_id=user-1",
			setupMocks: func(prsm *PullRequestServiceMock) {
				prsm.On("GetReviewAssignments", mock.Anything, "user-1").Return(reviewResponse, nil).Once()
			},
			expectedStatusCode:   http.StatusOK,
			expectedResponseBody: `{"user_id":"user-1","pull_requests":[{"pull_request_id":"pr-1","pull_request_name":"Feature A","author_id":"author-A","status":"OPEN"}]}`,
		},
		{
			name:      "User Not Found",
			targetURL: "/users/getReview?user_id=not-found",
			setupMocks: func(prsm *PullRequestServiceMock) {
				prsm.On("GetReviewAssignments", mock.Anything, "not-found").Return(nil, apperrors.ErrNotFound).Once()
			},
			expectedStatusCode:   http.StatusNotFound,
			expectedResponseBody: `{"error":{"code":"NOT_FOUND","message":"resource not found"}}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prServiceMock := new(PullRequestServiceMock)
			tc.setupMocks(prServiceMock)

			server := NewServer(slog.New(slog.NewJSONHandler(os.Stdout, nil)), nil, nil, prServiceMock)
			router := api.Handler(server)
			req := httptest.NewRequest(http.MethodGet, tc.targetURL, nil)
			rr := httptest.NewRecorder()

			router.ServeHTTP(rr, req)

			assert.Equal(t, tc.expectedStatusCode, rr.Code)
			assert.JSONEq(t, tc.expectedResponseBody, rr.Body.String())
			prServiceMock.AssertExpectations(t)
		})
	}
}

func (m *PullRequestServiceMock) GetStats(ctx context.Context) (*api.StatsResponse, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*api.StatsResponse), args.Error(1)
}

func TestServer_GetStats(t *testing.T) {
	expectedStats := &api.StatsResponse{
		UserStats: []api.UserStats{
			{UserId: "u1", Username: "Alice", OpenReviews: 1, MergedReviews: 5},
		},
	}

	prServiceMock := new(PullRequestServiceMock)
	prServiceMock.On("GetStats", mock.Anything).Return(expectedStats, nil).Once()

	server := NewServer(slog.New(slog.NewJSONHandler(os.Stdout, nil)), nil, nil, prServiceMock)
	router := api.Handler(server)
	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	expectedJSON := `{"user_stats":[{"user_id":"u1","username":"Alice","open_reviews":1,"merged_reviews":5}]}`
	assert.JSONEq(t, expectedJSON, rr.Body.String())
	prServiceMock.AssertExpectations(t)
}
