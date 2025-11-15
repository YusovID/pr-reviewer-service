package service

import (
	"context"
	"fmt"

	"github.com/YusovID/pr-reviewer-service/internal/domain"
	"github.com/YusovID/pr-reviewer-service/internal/repository"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
)

type TeamService interface {
	CreateTeam(ctx context.Context, team api.Team) (*api.Team, error)
	GetTeam(ctx context.Context, name string) (*api.Team, error)
}

type TeamServiceImpl struct {
	repo repository.TeamRepository
}

func NewTeamService(repo repository.TeamRepository) *TeamServiceImpl {
	return &TeamServiceImpl{repo: repo}
}

func (s *TeamServiceImpl) CreateTeam(ctx context.Context, team api.Team) (*api.Team, error) {
	domainTeamWithMembers, err := s.repo.CreateTeamWithUsers(ctx, team)
	if err != nil {
		return nil, fmt.Errorf("repo.CreateTeamWithUsers failed: %w", err)
	}

	return toAPITeam(domainTeamWithMembers), nil
}

func (s *TeamServiceImpl) GetTeam(ctx context.Context, name string) (*api.Team, error) {
	domainTeam, err := s.repo.GetTeamByName(ctx, name)
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
