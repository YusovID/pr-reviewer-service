package apperrors

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound          = errors.New("resource not found")
	ErrAlreadyExists     = errors.New("resource already exists")
	ErrTeamAlreadyExists = errors.New("team already exists")
	ErrPRAlreadyExists   = errors.New("pull request already exists")

	ErrInvalidRequest = errors.New("invalid request body")

	ErrPRMerged            = errors.New("cannot modify merged pull request")
	ErrReviewerNotAssigned = errors.New("reviewer is not assigned to this PR")
	ErrNoCandidate         = errors.New("no active replacement candidate found in team")
)

type TeamAlreadyExistsError struct{ TeamName string }

func (e *TeamAlreadyExistsError) Error() string {
	return fmt.Sprintf("team '%s' already exists", e.TeamName)
}
func (e *TeamAlreadyExistsError) Is(target error) bool { return target == ErrAlreadyExists }

type PRAlreadyExistsError struct{ PRID string }

func (e *PRAlreadyExistsError) Error() string {
	return fmt.Sprintf("pull request '%s' already exists", e.PRID)
}
func (e *PRAlreadyExistsError) Is(target error) bool { return target == ErrAlreadyExists }
