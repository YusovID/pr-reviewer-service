//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/YusovID/pr-reviewer-service/internal/domain"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPRTest(t *testing.T) {
	t.Helper()
	truncateTables(t, testDB)
	teamRepo := NewTeamRepository(testDB, logger)
	_, err := teamRepo.CreateTeamWithUsers(context.Background(), api.Team{
		TeamName: "pr-team",
		Members: []api.TeamMember{
			{UserId: "author", Username: "Author", IsActive: true},
			{UserId: "rev1", Username: "Reviewer1", IsActive: true},
			{UserId: "rev2", Username: "Reviewer2", IsActive: true},
			{UserId: "rev3-inactive", Username: "Reviewer3", IsActive: false},
			{UserId: "rev4", Username: "Reviewer4", IsActive: true},
		},
	})
	require.NoError(t, err)
}

func TestPullRequestRepository_Flow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	setupPRTest(t)
	repo := NewPullRequestRepository(testDB, logger)
	ctx := context.Background()

	teamID, err := repo.GetAuthorTeamID(ctx, "author")
	require.NoError(t, err)
	assert.NotZero(t, teamID)

	reviewers, err := repo.GetRandomActiveReviewers(ctx, teamID, []string{"author"}, 2)
	require.NoError(t, err)
	assert.Len(t, reviewers, 2)
	assert.NotContains(t, reviewers, "author")
	assert.NotContains(t, reviewers, "rev3-inactive")

	prToCreate := &domain.PullRequest{
		ID:       "pr-1",
		Name:     "My first PR",
		AuthorID: "author",
		Status:   api.PullRequestStatusOPEN,
	}
	tx, err := testDB.Beginx()
	require.NoError(t, err)
	err = repo.CreatePR(ctx, tx, prToCreate)
	require.NoError(t, err)
	err = repo.AssignReviewers(ctx, tx, "pr-1", reviewers)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	pr, err := repo.GetPRByIDWithReviewers(ctx, "pr-1")
	require.NoError(t, err)
	assert.Equal(t, "My first PR", pr.Name)
	assert.ElementsMatch(t, reviewers, pr.ReviewerIDs)

	oldReviewer := reviewers[0]
	excludeIDs := append(reviewers, "author")
	newCandidates, err := repo.GetRandomActiveReviewers(ctx, teamID, excludeIDs, 1)
	require.NoError(t, err)
	require.NotEmpty(t, newCandidates)
	newReviewer := newCandidates[0]

	tx, err = testDB.Beginx()
	require.NoError(t, err)
	err = repo.ReplaceReviewer(ctx, tx, "pr-1", oldReviewer, newReviewer)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	pr, err = repo.GetPRByIDWithReviewers(ctx, "pr-1")
	require.NoError(t, err)
	assert.NotContains(t, pr.ReviewerIDs, oldReviewer)
	assert.Contains(t, pr.ReviewerIDs, newReviewer)

	tx, err = testDB.Beginx()
	require.NoError(t, err)
	err = repo.UpdatePRStatus(ctx, tx, "pr-1", api.PullRequestStatusMERGED, time.Now())
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	pr, err = repo.GetPRByID(ctx, "pr-1")
	require.NoError(t, err)
	assert.Equal(t, api.PullRequestStatusMERGED, pr.Status)
	assert.NotNil(t, pr.MergedAt)

	assignments, err := repo.GetReviewAssignments(ctx, newReviewer)
	require.NoError(t, err)
	require.Len(t, assignments, 1)
	assert.Equal(t, "pr-1", assignments[0].ID)
}
