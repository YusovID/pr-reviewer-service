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

func TestTeamRepository_CreateTeamWithUsers_And_GetTeamByName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	truncateTables(t, testDB)
	repo := NewTeamRepository(testDB, logger)
	ctx := context.Background()

	teamToCreate := api.Team{
		TeamName: "backend",
		Members: []api.TeamMember{
			{UserId: "u1", Username: "Alice", IsActive: true},
			{UserId: "u2", Username: "Bob", IsActive: false},
		},
	}

	createdTeam, err := repo.CreateTeamWithUsers(ctx, teamToCreate)
	require.NoError(t, err)
	assert.Equal(t, "backend", createdTeam.Name)
	require.Len(t, createdTeam.Members, 2)
	assert.Equal(t, "u1", createdTeam.Members[0].ID)

	_, err = repo.CreateTeamWithUsers(ctx, teamToCreate)
	require.Error(t, err)
	var teamExistsErr *apperrors.TeamAlreadyExistsError
	assert.ErrorAs(t, err, &teamExistsErr)

	fetchedTeam, err := repo.GetTeamByName(ctx, "backend")
	require.NoError(t, err)
	assert.Equal(t, createdTeam.ID, fetchedTeam.ID)
	assert.Equal(t, "backend", fetchedTeam.Name)
	require.Len(t, fetchedTeam.Members, 2)
	assert.Equal(t, "Alice", fetchedTeam.Members[0].Username)
	assert.Equal(t, "Bob", fetchedTeam.Members[1].Username)
	assert.False(t, fetchedTeam.Members[1].IsActive)

	_, err = repo.GetTeamByName(ctx, "non-existent")
	require.Error(t, err)
	assert.ErrorContains(t, err, apperrors.ErrNotFound.Error())
}
