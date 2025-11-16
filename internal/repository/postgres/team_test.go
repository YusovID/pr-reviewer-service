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
	assert.ErrorAs(t, err, &teamExistsErr, "expected TeamAlreadyExistsError")
	assert.Equal(t, "backend", teamExistsErr.TeamName)

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
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

func TestTeamRepository_CreateTeam_NoMembers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	truncateTables(t, testDB)
	repo := NewTeamRepository(testDB, logger)
	ctx := context.Background()

	teamToCreate := api.Team{
		TeamName: "empty-team",
		Members:  []api.TeamMember{},
	}

	createdTeam, err := repo.CreateTeamWithUsers(ctx, teamToCreate)
	require.NoError(t, err)
	assert.Equal(t, "empty-team", createdTeam.Name)
	assert.Empty(t, createdTeam.Members)

	fetchedTeam, err := repo.GetTeamByName(ctx, "empty-team")
	require.NoError(t, err)
	assert.Equal(t, createdTeam.ID, fetchedTeam.ID)
	assert.Empty(t, fetchedTeam.Members)
}

func TestTeamRepository_UpsertTeamMembers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	truncateTables(t, testDB)
	repo := NewTeamRepository(testDB, logger)
	ctx := context.Background()

	team1 := api.Team{
		TeamName: "team-alpha",
		Members:  []api.TeamMember{{UserId: "u1", Username: "Alice", IsActive: true}},
	}
	_, err := repo.CreateTeamWithUsers(ctx, team1)
	require.NoError(t, err)

	team2 := api.Team{
		TeamName: "team-beta",
		Members:  []api.TeamMember{{UserId: "u1", Username: "Alice-Updated", IsActive: false}},
	}
	createdTeam2, err := repo.CreateTeamWithUsers(ctx, team2)
	require.NoError(t, err)

	fetchedTeam2, err := repo.GetTeamByName(ctx, "team-beta")
	require.NoError(t, err)
	require.Len(t, fetchedTeam2.Members, 1)
	assert.Equal(t, "u1", fetchedTeam2.Members[0].ID)
	assert.Equal(t, "Alice-Updated", fetchedTeam2.Members[0].Username)
	assert.False(t, fetchedTeam2.Members[0].IsActive)
	assert.Equal(t, createdTeam2.ID, fetchedTeam2.Members[0].TeamID)

	fetchedTeam1, err := repo.GetTeamByName(ctx, "team-alpha")
	require.NoError(t, err)
	assert.Empty(t, fetchedTeam1.Members)
}
