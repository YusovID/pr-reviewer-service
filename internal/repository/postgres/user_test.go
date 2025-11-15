//go:build integration

package postgres

import (
	"context"
	"testing"

	"github.com/YusovID/pr-reviewer-service/internal/apperrors"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUserRepository_SetIsActive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	truncateTables(t, testDB)
	teamRepo := NewTeamRepository(testDB, logger)
	userRepo := NewUserRepository(testDB, logger)
	ctx := context.Background()

	_, err := teamRepo.CreateTeamWithUsers(ctx, api.Team{
		TeamName: "test-team",
		Members: []api.TeamMember{
			{UserId: "user-to-deactivate", Username: "Test User", IsActive: true},
		},
	})
	require.NoError(t, err)

	updatedUser, err := userRepo.SetIsActive(ctx, "user-to-deactivate", false)
	require.NoError(t, err)
	assert.Equal(t, "user-to-deactivate", updatedUser.UserId)
	assert.Equal(t, "test-team", updatedUser.TeamName)
	assert.False(t, updatedUser.IsActive)

	var isActive bool
	err = testDB.Get(&isActive, "SELECT is_active FROM users WHERE id = 'user-to-deactivate'")
	require.NoError(t, err)
	assert.False(t, isActive)

	updatedUser, err = userRepo.SetIsActive(ctx, "user-to-deactivate", true)
	require.NoError(t, err)
	assert.True(t, updatedUser.IsActive)

	_, err = userRepo.SetIsActive(ctx, "non-existent-user", false)
	require.Error(t, err)
	assert.ErrorContains(t, err, apperrors.ErrNotFound.Error())
}
