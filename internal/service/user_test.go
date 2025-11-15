package service

import (
	"context"
	"testing"

	"github.com/YusovID/pr-reviewer-service/internal/apperrors"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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

			service := NewUserService(repoMock)

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
