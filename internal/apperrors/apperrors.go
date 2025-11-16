// package apperrors provides a centralized registry for domain-specific errors.
package apperrors

import (
	"errors"
	"fmt"
)

var (
	// ErrNotFound indicates that a requested resource was not found.
	ErrNotFound = errors.New("resource not found")
	// ErrAlreadyExists indicates that a resource being created already exists.
	ErrAlreadyExists = errors.New("resource already exists")
	// ErrTeamAlreadyExists is a specific error for existing teams.
	ErrTeamAlreadyExists = errors.New("team already exists")
	// ErrPRAlreadyExists is a specific error for existing pull requests.
	ErrPRAlreadyExists = errors.New("pull request already exists")

	// ErrInvalidRequest indicates a malformed request body (e.g., bad JSON).
	ErrInvalidRequest = errors.New("invalid request body")
	// ErrValidation indicates that request data failed business rule validation.
	ErrValidation = errors.New("validation failed")

	// ErrPRMerged indicates an attempt to modify a pull request that has already been merged.
	ErrPRMerged = errors.New("cannot modify merged pull request")
	// ErrReviewerNotAssigned indicates an attempt to reassign a reviewer who is not assigned to the PR.
	ErrReviewerNotAssigned = errors.New("reviewer is not assigned to this PR")
	// ErrNoCandidate indicates that no suitable active user could be found to become a new reviewer.
	ErrNoCandidate = errors.New("no active replacement candidate found in team")
)

// TeamAlreadyExistsError is a structured error for when a team with a given name already exists.
type TeamAlreadyExistsError struct{ TeamName string }

func (e *TeamAlreadyExistsError) Error() string {
	return fmt.Sprintf("team '%s' already exists", e.TeamName)
}
func (e *TeamAlreadyExistsError) Is(target error) bool { return target == ErrAlreadyExists }

// PRAlreadyExistsError is a structured error for when a PR with a given ID already exists.
type PRAlreadyExistsError struct{ PRID string }

func (e *PRAlreadyExistsError) Error() string {
	return fmt.Sprintf("pull request '%s' already exists", e.PRID)
}
func (e *PRAlreadyExistsError) Is(target error) bool { return target == ErrAlreadyExists }
