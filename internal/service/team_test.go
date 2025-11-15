package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/YusovID/pr-reviewer-service/internal/apperrors"
	"github.com/YusovID/pr-reviewer-service/internal/domain"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTeamServiceImpl_CreateTeam(t *testing.T) {
	ctx := context.Background()

	inputTeam := api.Team{
		TeamName: "test-team",
		Members: []api.TeamMember{
			{UserId: "u1", Username: "Test User", IsActive: true},
		},
	}

	domainTeamWithMembers := &domain.TeamWithMembers{
		ID:   1,
		Name: "test-team",
		Members: []domain.User{
			{ID: "u1", Username: "Test User", TeamID: 1, IsActive: true},
		},
	}

	testCases := []struct {
		name          string
		setupMock     func(repoMock *TeamRepositoryMock)
		inputTeam     api.Team
		expectedTeam  *api.Team
		expectedError bool
	}{
		{
			name: "Success: Team and users are created",
			setupMock: func(repoMock *TeamRepositoryMock) {
				repoMock.On("CreateTeamWithUsers", mock.Anything, inputTeam).Return(domainTeamWithMembers, nil)
			},
			inputTeam: inputTeam,
			expectedTeam: &api.Team{
				TeamName: "test-team",
				Members: []api.TeamMember{
					{UserId: "u1", Username: "Test User", IsActive: true},
				},
			},
			expectedError: false,
		},
		{
			name: "Failure: Repository returns error on CreateTeamWithUsers",
			setupMock: func(repoMock *TeamRepositoryMock) {
				repoMock.On("CreateTeamWithUsers", mock.Anything, inputTeam).Return(nil, errors.New("database connection lost"))
			},
			inputTeam:     inputTeam,
			expectedTeam:  nil,
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := new(TeamRepositoryMock)
			tc.setupMock(repoMock)

			service := NewTeamService(repoMock)

			resultTeam, err := service.CreateTeam(ctx, tc.inputTeam)

			assert.Equal(t, tc.expectedTeam, resultTeam)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			repoMock.AssertExpectations(t)
		})
	}
}

func TestTeamServiceImpl_GetTeam(t *testing.T) {
	ctx := context.Background()
	teamName := "existing-team"

	domainTeamWithMembers := &domain.TeamWithMembers{
		ID:   1,
		Name: teamName,
		Members: []domain.User{
			{ID: "u1", Username: "Alice", TeamID: 1, IsActive: true},
			{ID: "u2", Username: "Bob", TeamID: 1, IsActive: true},
		},
	}

	expectedApiTeam := &api.Team{
		TeamName: teamName,
		Members: []api.TeamMember{
			{UserId: "u1", Username: "Alice", IsActive: true},
			{UserId: "u2", Username: "Bob", IsActive: true},
		},
	}

	testCases := []struct {
		name          string
		teamName      string
		setupMock     func(repoMock *TeamRepositoryMock)
		expectedTeam  *api.Team
		expectedError error
	}{
		{
			name:     "Success: Team is found",
			teamName: teamName,
			setupMock: func(repoMock *TeamRepositoryMock) {
				repoMock.On("GetTeamByName", ctx, teamName).Return(domainTeamWithMembers, nil).Once()
			},
			expectedTeam:  expectedApiTeam,
			expectedError: nil,
		},
		{
			name:     "Failure: Team not found",
			teamName: "non-existent-team",
			setupMock: func(repoMock *TeamRepositoryMock) {
				repoMock.On("GetTeamByName", ctx, "non-existent-team").Return(nil, apperrors.ErrNotFound).Once()
			},
			expectedTeam:  nil,
			expectedError: apperrors.ErrNotFound,
		},
		{
			name:     "Failure: Repository returns a generic error",
			teamName: "any-team",
			setupMock: func(repoMock *TeamRepositoryMock) {
				repoMock.On("GetTeamByName", ctx, "any-team").Return(nil, errors.New("internal db error")).Once()
			},
			expectedTeam:  nil,
			expectedError: errors.New("internal db error"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			repoMock := new(TeamRepositoryMock)
			tc.setupMock(repoMock)

			service := NewTeamService(repoMock)

			resultTeam, err := service.GetTeam(ctx, tc.teamName)

			assert.Equal(t, tc.expectedTeam, resultTeam)

			if tc.expectedError != nil {
				assert.Error(t, err)
				assert.True(t, errors.Is(err, tc.expectedError) || err.Error() == fmt.Sprintf("repo.GetTeamByName failed: %v", tc.expectedError))
			} else {
				assert.NoError(t, err)
			}

			repoMock.AssertExpectations(t)
		})
	}
}
