package http

import (
	"context"

	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/stretchr/testify/mock"
)

type TeamServiceMock struct {
	mock.Mock
}

func (m *TeamServiceMock) CreateTeam(ctx context.Context, team api.Team) (*api.Team, error) {
	args := m.Called(ctx, team)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*api.Team), args.Error(1)
}

func (m *TeamServiceMock) GetTeam(ctx context.Context, name string) (*api.Team, error) {
	args := m.Called(ctx, name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*api.Team), args.Error(1)
}

type UserServiceMock struct {
	mock.Mock
}

func (m *UserServiceMock) SetIsActive(ctx context.Context, userID string, isActive bool) (*api.User, error) {
	args := m.Called(ctx, userID, isActive)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*api.User), args.Error(1)
}

type PullRequestServiceMock struct {
	mock.Mock
}

func (m *PullRequestServiceMock) CreatePR(ctx context.Context, prID string, prName string, authorID string) (*api.PullRequest, error) {
	args := m.Called(ctx, prID, prName, authorID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*api.PullRequest), args.Error(1)
}

func (m *PullRequestServiceMock) MergePR(ctx context.Context, prID string) (*api.PullRequest, error) {
	args := m.Called(ctx, prID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*api.PullRequest), args.Error(1)
}

func (m *PullRequestServiceMock) ReassignReviewer(ctx context.Context, prID string, oldReviewerID string) (*api.ReassignResponse, error) {
	args := m.Called(ctx, prID, oldReviewerID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*api.ReassignResponse), args.Error(1)
}

func (m *PullRequestServiceMock) GetReviewAssignments(ctx context.Context, userID string) (*api.GetReviewResponse, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*api.GetReviewResponse), args.Error(1)
}
