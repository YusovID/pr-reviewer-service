package domain

import (
	"time"

	"github.com/YusovID/pr-reviewer-service/pkg/api"
)

type User struct {
	ID       string `db:"id"`
	Username string `db:"username"`
	TeamID   int    `db:"team_id"`
	IsActive bool   `db:"is_active"`
}

type Team struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

type TeamWithMembers struct {
	ID      int
	Name    string
	Members []User
}

type PullRequest struct {
	ID                string                `db:"id"`
	Name              string                `db:"name"`
	AuthorID          string                `db:"author_id"`
	Status            api.PullRequestStatus `db:"status"`
	NeedMoreReviewers bool                  `db:"need_more_reviewers"`
	CreatedAt         time.Time             `db:"created_at"`
	MergedAt          *time.Time            `db:"merged_at"`
	ReviewerIDs       []string
}

type Reviewer struct {
	PullRequestID string `db:"pull_request_id"`
	UserID        string `db:"user_id"`
}

type Stats struct {
	UserID        string `db:"user_id"`
	Username      string `db:"username"`
	OpenReviews   int    `db:"open_reviews"`
	MergedReviews int    `db:"merged_reviews"`
}
