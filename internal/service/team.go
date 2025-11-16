package service

import (
	"context"
	"fmt"

	"github.com/YusovID/pr-reviewer-service/internal/domain"
	"github.com/YusovID/pr-reviewer-service/internal/repository"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/jmoiron/sqlx"
)

// TeamService defines the application's business logic for managing teams.
type TeamService interface {
	// CreateTeamWithUsers handles the creation of a new team and its members.
	// It ensures that a team with the same name does not already exist.
	CreateTeamWithUsers(ctx context.Context, team api.Team) (*api.Team, error)
	// GetTeam retrieves a team by its name, including all its members.
	GetTeam(ctx context.Context, name string) (*api.Team, error)
}

type TeamServiceImpl struct {
	repo repository.TeamRepository
	db   *sqlx.DB
}

// NewTeamService creates a new instance of TeamServiceImpl.
func NewTeamService(repo repository.TeamRepository, db *sqlx.DB) *TeamServiceImpl {
	return &TeamServiceImpl{
		repo: repo,
		db:   db,
	}
}

func (s *TeamServiceImpl) CreateTeamWithUsers(ctx context.Context, team api.Team) (*api.Team, error) {
	domainTeamWithMembers, err := s.repo.CreateTeamWithUsers(ctx, team)
	if err != nil {
		return nil, fmt.Errorf("repo.CreateTeamWithUsers failed: %w", err)
	}

	return toAPITeam(domainTeamWithMembers), nil
}

func (s *TeamServiceImpl) GetTeam(ctx context.Context, name string) (*api.Team, error) {
	domainTeam, err := s.repo.GetTeamByName(ctx, s.db, name)
	if err != nil {
		return nil, fmt.Errorf("repo.GetTeamByName failed: %w", err)
	}

	return toAPITeam(domainTeam), nil
}

func toAPITeam(domainTeam *domain.TeamWithMembers) *api.Team {
	apiMembers := make([]api.TeamMember, len(domainTeam.Members))
	for i, member := range domainTeam.Members {
		apiMembers[i] = api.TeamMember{
			UserId:   member.ID,
			Username: member.Username,
			IsActive: member.IsActive,
		}
	}

	return &api.Team{
		TeamName: domainTeam.Name,
		Members:  apiMembers,
	}
}
