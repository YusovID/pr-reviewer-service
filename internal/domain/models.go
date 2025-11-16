// package domain contains the core business models of the service.
package domain

import (
	"time"

	"github.com/YusovID/pr-reviewer-service/pkg/api"
)

// User represents a user entity, typically a developer or a reviewer.
type User struct {
	ID       string `db:"id"`
	Username string `db:"username"`
	TeamID   int    `db:"team_id"`
	IsActive bool   `db:"is_active"`
}

// Team represents a group of users.
type Team struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}

// TeamWithMembers is a composite model that holds a team and its members.
type TeamWithMembers struct {
	ID      int
	Name    string
	Members []User
}

// PullRequest represents a pull request entity in the system.
type PullRequest struct {
	ID       string                `db:"id"`
	Name     string                `db:"name"`
	AuthorID string                `db:"author_id"`
	Status   api.PullRequestStatus `db:"status"`
	// NeedMoreReviewers is a flag indicating that the system could not find
	// the desired number of reviewers (less than 2) when the PR was created.
	NeedMoreReviewers bool       `db:"need_more_reviewers"`
	CreatedAt         time.Time  `db:"created_at"`
	MergedAt          *time.Time `db:"merged_at"`
	// ReviewerIDs contains the identifiers of assigned reviewers.
	// This field is not persisted in the 'pull_requests' table directly
	// but is populated from the 'reviewers' association table.
	ReviewerIDs []string
}

// Reviewer represents the association between a PullRequest and a User (reviewer).
type Reviewer struct {
	PullRequestID string `db:"pull_request_id"`
	UserID        string `db:"user_id"`
}

// Stats represents user statistics regarding their review activities.
type Stats struct {
	UserID        string `db:"user_id"`
	Username      string `db:"username"`
	OpenReviews   int    `db:"open_reviews"`
	MergedReviews int    `db:"merged_reviews"`
}
