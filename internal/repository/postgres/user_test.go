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

func TestUserRepository_DeactivateUsersByTeamID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	truncateTables(t, testDB)
	teamRepo := NewTeamRepository(testDB, logger)
	userRepo := NewUserRepository(testDB, logger)
	ctx := context.Background()

	targetTeam, err := teamRepo.CreateTeamWithUsers(ctx, api.Team{
		TeamName: "target-team",
		Members: []api.TeamMember{
			{UserId: "u1-active", Username: "User 1", IsActive: true},
			{UserId: "u2-active", Username: "User 2", IsActive: true},
			{UserId: "u3-inactive", Username: "User 3", IsActive: false},
		},
	})
	require.NoError(t, err)

	_, err = teamRepo.CreateTeamWithUsers(ctx, api.Team{
		TeamName: "other-team",
		Members: []api.TeamMember{
			{UserId: "u4-other-team", Username: "User 4", IsActive: true},
		},
	})
	require.NoError(t, err)

	tx, err := testDB.Beginx()
	require.NoError(t, err)

	deactivatedIDs, err := userRepo.DeactivateUsersByTeamID(ctx, tx, targetTeam.ID)
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{"u1-active", "u2-active"}, deactivatedIDs)

	var u1active, u2active, u3active, u4active bool
	require.NoError(t, tx.Get(&u1active, "SELECT is_active FROM users WHERE id = 'u1-active'"))
	require.NoError(t, tx.Get(&u2active, "SELECT is_active FROM users WHERE id = 'u2-active'"))
	require.NoError(t, tx.Get(&u3active, "SELECT is_active FROM users WHERE id = 'u3-inactive'"))
	require.NoError(t, tx.Get(&u4active, "SELECT is_active FROM users WHERE id = 'u4-other-team'"))

	assert.False(t, u1active, "u1 should be deactivated")
	assert.False(t, u2active, "u2 should be deactivated")
	assert.False(t, u3active, "u3 should remain inactive")
	assert.True(t, u4active, "u4 from other team should not be affected")

	deactivatedAgain, err := userRepo.DeactivateUsersByTeamID(ctx, tx, targetTeam.ID)
	require.NoError(t, err)
	assert.Empty(t, deactivatedAgain, "calling again on the same team should return no users")

	require.NoError(t, tx.Rollback())
}
