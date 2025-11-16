//go:build integration

package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/YusovID/pr-reviewer-service/internal/apperrors"
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
		t.Skip("skipping integration test in short mode.")
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

func TestPullRequestRepository_GetUserStats(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode.")
	}
	setupPRTest(t)
	repo := NewPullRequestRepository(testDB, logger)
	ctx := context.Background()

	pr1 := &domain.PullRequest{ID: "pr-1", Name: "PR 1", AuthorID: "author", Status: api.PullRequestStatusOPEN}
	pr2 := &domain.PullRequest{ID: "pr-2", Name: "PR 2", AuthorID: "author", Status: api.PullRequestStatusMERGED}
	pr3 := &domain.PullRequest{ID: "pr-3", Name: "PR 3", AuthorID: "author", Status: api.PullRequestStatusOPEN}

	tx, err := testDB.Beginx()
	require.NoError(t, err)
	require.NoError(t, repo.CreatePR(ctx, tx, pr1))
	require.NoError(t, repo.CreatePR(ctx, tx, pr2))
	require.NoError(t, repo.CreatePR(ctx, tx, pr3))

	require.NoError(t, repo.AssignReviewers(ctx, tx, "pr-1", []string{"rev1"}))
	require.NoError(t, repo.AssignReviewers(ctx, tx, "pr-2", []string{"rev1"}))
	require.NoError(t, repo.AssignReviewers(ctx, tx, "pr-3", []string{"rev2"}))
	require.NoError(t, tx.Commit())

	stats, err := repo.GetUserStats(ctx)
	require.NoError(t, err)

	statsMap := make(map[string]domain.Stats)
	for _, s := range stats {
		statsMap[s.UserID] = s
	}

	assert.Equal(t, 1, statsMap["rev1"].OpenReviews)
	assert.Equal(t, 1, statsMap["rev1"].MergedReviews)
	assert.Equal(t, 1, statsMap["rev2"].OpenReviews)
	assert.Equal(t, 0, statsMap["rev2"].MergedReviews)
	assert.Equal(t, 0, statsMap["author"].OpenReviews)
	assert.Equal(t, 0, statsMap["author"].MergedReviews)
}

func TestPullRequestRepository_CreatePR_Constraints(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	setupPRTest(t)
	repo := NewPullRequestRepository(testDB, logger)
	ctx := context.Background()

	pr := &domain.PullRequest{
		ID:       "pr-duplicate",
		Name:     "Test",
		AuthorID: "author",
		Status:   api.PullRequestStatusOPEN,
	}
	tx, _ := testDB.Beginx()
	require.NoError(t, repo.CreatePR(ctx, tx, pr))
	require.NoError(t, tx.Commit())

	tx, _ = testDB.Beginx()
	err := repo.CreatePR(ctx, tx, pr)
	require.Error(t, err)
	var prExistsErr *apperrors.PRAlreadyExistsError
	assert.ErrorAs(t, err, &prExistsErr, "expected PRAlreadyExistsError")
	tx.Rollback()

	prInvalidAuthor := &domain.PullRequest{
		ID:       "pr-invalid-author",
		Name:     "Test",
		AuthorID: "non-existent-author",
		Status:   api.PullRequestStatusOPEN,
	}
	tx, _ = testDB.Beginx()
	err = repo.CreatePR(ctx, tx, prInvalidAuthor)
	require.Error(t, err)
	assert.ErrorContains(t, err, apperrors.ErrNotFound.Error())
	tx.Rollback()
}

func TestPullRequestRepository_UpdatePRStatus_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	setupPRTest(t)
	repo := NewPullRequestRepository(testDB, logger)
	ctx := context.Background()

	tx, _ := testDB.Beginx()
	err := repo.UpdatePRStatus(ctx, tx, "non-existent-pr", api.PullRequestStatusMERGED, time.Now())
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	tx.Rollback()
}

func TestPullRequestRepository_GetAuthorTeamID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	setupPRTest(t)
	repo := NewPullRequestRepository(testDB, logger)
	ctx := context.Background()

	_, err := repo.GetAuthorTeamID(ctx, "non-existent-author")
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

func TestPullRequestRepository_GetReviewerTeamID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	setupPRTest(t)
	repo := NewPullRequestRepository(testDB, logger)
	ctx := context.Background()

	var expectedTeamID int
	err := testDB.Get(&expectedTeamID, "SELECT id FROM teams WHERE name = 'pr-team'")
	require.NoError(t, err)

	actualTeamID, err := repo.GetReviewerTeamID(ctx, "rev1")
	require.NoError(t, err)
	assert.Equal(t, expectedTeamID, actualTeamID)

	_, err = repo.GetReviewerTeamID(ctx, "non-existent-reviewer")
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

func TestPullRequestRepository_GetPRByIDWithLock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}
	setupPRTest(t)
	repo := NewPullRequestRepository(testDB, logger)
	ctx := context.Background()

	prToCreate := &domain.PullRequest{
		ID:       "pr-for-lock",
		Name:     "Test Lock",
		AuthorID: "author",
		Status:   api.PullRequestStatusOPEN,
	}
	tx, err := testDB.Beginx()
	require.NoError(t, err)
	require.NoError(t, repo.CreatePR(ctx, tx, prToCreate))
	require.NoError(t, tx.Commit())

	tx, err = testDB.Beginx()
	require.NoError(t, err)

	pr, err := repo.GetPRByIDWithLock(ctx, tx, "pr-for-lock")
	require.NoError(t, err)
	require.NotNil(t, pr)
	assert.Equal(t, "pr-for-lock", pr.ID)
	assert.Equal(t, "Test Lock", pr.Name)

	require.NoError(t, tx.Rollback())

	tx, err = testDB.Beginx()
	require.NoError(t, err)

	_, err = repo.GetPRByIDWithLock(ctx, tx, "non-existent-pr-for-lock")
	require.Error(t, err)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)

	require.NoError(t, tx.Rollback())
}

func TestPullRequestRepository_GetOpenPRsByReviewers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode.")
	}
	setupPRTest(t)
	repo := NewPullRequestRepository(testDB, logger)
	ctx := context.Background()

	pr1 := &domain.PullRequest{ID: "pr-open-1", Name: "Open PR 1", AuthorID: "author", Status: api.PullRequestStatusOPEN}
	pr2 := &domain.PullRequest{ID: "pr-open-2", Name: "Open PR 2", AuthorID: "author", Status: api.PullRequestStatusOPEN}
	pr3 := &domain.PullRequest{ID: "pr-merged-1", Name: "Merged PR 1", AuthorID: "author", Status: api.PullRequestStatusMERGED}
	pr4 := &domain.PullRequest{ID: "pr-open-other", Name: "Other Open PR", AuthorID: "author", Status: api.PullRequestStatusOPEN}

	tx, err := testDB.Beginx()
	require.NoError(t, err)
	require.NoError(t, repo.CreatePR(ctx, tx, pr1))
	require.NoError(t, repo.CreatePR(ctx, tx, pr2))
	require.NoError(t, repo.CreatePR(ctx, tx, pr3))
	require.NoError(t, repo.CreatePR(ctx, tx, pr4))

	require.NoError(t, repo.AssignReviewers(ctx, tx, "pr-open-1", []string{"rev1"}))
	require.NoError(t, repo.AssignReviewers(ctx, tx, "pr-open-2", []string{"rev1", "rev2"}))
	require.NoError(t, repo.AssignReviewers(ctx, tx, "pr-merged-1", []string{"rev1"}))
	require.NoError(t, repo.AssignReviewers(ctx, tx, "pr-open-other", []string{"rev4"}))
	require.NoError(t, tx.Commit())

	tx, err = testDB.Beginx()
	require.NoError(t, err)

	openPRs, err := repo.GetOpenPRsByReviewers(ctx, tx, []string{"rev1"})
	require.NoError(t, err)
	require.Len(t, openPRs, 2, "rev1 should have two open PRs")

	prIDs := []string{openPRs[0].ID, openPRs[1].ID}
	assert.Contains(t, prIDs, "pr-open-1")
	assert.Contains(t, prIDs, "pr-open-2")
	assert.NotContains(t, prIDs, "pr-merged-1")
	assert.NotContains(t, prIDs, "pr-open-other")

	for _, pr := range openPRs {
		if pr.ID == "pr-open-2" {
			assert.ElementsMatch(t, []string{"rev1", "rev2"}, pr.ReviewerIDs)
		}
	}

	noOpenPRs, err := repo.GetOpenPRsByReviewers(ctx, tx, []string{"author"})
	require.NoError(t, err)
	assert.Empty(t, noOpenPRs)

	require.NoError(t, tx.Rollback())
}
