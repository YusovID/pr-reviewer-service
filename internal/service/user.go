package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/YusovID/pr-reviewer-service/internal/domain"
	"github.com/YusovID/pr-reviewer-service/internal/repository"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
	"github.com/jmoiron/sqlx"
)

type UserService interface {
	SetIsActive(ctx context.Context, userID string, isActive bool) (*api.User, error)
	DeactivateTeam(ctx context.Context, teamName string) (deactivatedCount int, reassignedCount int, err error)
}

type UserServiceImpl struct {
	BaseService
	repo     repository.UserRepository
	teamRepo repository.TeamRepository
	prQuery  repository.PRQueryRepository
	prCmd    repository.PRCommandRepository
	userPR   repository.UserPRRepository
}

func NewUserService(
	repo repository.UserRepository,
	teamRepo repository.TeamRepository,
	prQuery repository.PRQueryRepository,
	prCmd repository.PRCommandRepository,
	userPR repository.UserPRRepository,
	db Transactor,
	log *slog.Logger,
) *UserServiceImpl {
	return &UserServiceImpl{
		BaseService: NewBaseService(db, log),
		repo:        repo,
		teamRepo:    teamRepo,
		prQuery:     prQuery,
		prCmd:       prCmd,
		userPR:      userPR,
	}
}

func (s *UserServiceImpl) SetIsActive(ctx context.Context, userID string, isActive bool) (*api.User, error) {
	user, err := s.repo.SetIsActive(ctx, userID, isActive)
	if err != nil {
		return nil, fmt.Errorf("repo.SetIsActive failed: %w", err)
	}

	return user, nil
}

func (s *UserServiceImpl) DeactivateTeam(ctx context.Context, teamName string) (deactivatedCount int, reassignedCount int, err error) {
	const op = "internal.service.user.DeactivateTeam"
	log := s.log.With(slog.String("op", op), slog.String("team_name", teamName))

	err = s.transaction(ctx, op, func(tx *sqlx.Tx) error {
		team, err := s.teamRepo.GetTeamByName(ctx, tx, teamName)
		if err != nil {
			return err
		}

		deactivatedUserIDs, err := s.repo.DeactivateUsersByTeamID(ctx, tx, team.ID)
		if err != nil {
			return fmt.Errorf("failed to deactivate users: %w", err)
		}

		deactivatedCount = len(deactivatedUserIDs)

		if deactivatedCount == 0 {
			log.Info("no active users to deactivate in this team")
			return nil
		}

		prsToReassign, err := s.prQuery.GetOpenPRsByReviewers(ctx, tx, deactivatedUserIDs)
		if err != nil {
			return fmt.Errorf("failed to get open PRs: %w", err)
		}

		reassignedCount = len(prsToReassign)
		if reassignedCount == 0 {
			log.Info("no open PRs to reassign for deactivated users")
			return nil
		}

		deactivatedSet := make(map[string]struct{}, len(deactivatedUserIDs))
		for _, id := range deactivatedUserIDs {
			deactivatedSet[id] = struct{}{}
		}

		if err := s.reassignPRsForDeactivatedUsers(ctx, tx, team, prsToReassign, deactivatedSet); err != nil {
			return fmt.Errorf("failed during PR reassignment: %w", err)
		}

		return nil
	})

	if err != nil {
		return 0, 0, err
	}

	return deactivatedCount, reassignedCount, nil
}

func (s *UserServiceImpl) reassignPRsForDeactivatedUsers(
	ctx context.Context,
	tx *sqlx.Tx,
	team *domain.TeamWithMembers,
	prsToReassign []domain.PullRequest,
	deactivatedSet map[string]struct{},
) error {
	log := s.log.With(slog.String("op", "internal.service.user.reassignPRs"))

	for _, pr := range prsToReassign {
		originalReviewers := make([]string, len(pr.ReviewerIDs))
		copy(originalReviewers, pr.ReviewerIDs)

		for _, oldReviewerID := range originalReviewers {
			if _, isDeactivated := deactivatedSet[oldReviewerID]; isDeactivated {
				excludeIDs := excludeIDs(&pr, pr.ReviewerIDs)

				candidates, err := s.userPR.GetRandomActiveReviewers(ctx, team.ID, excludeIDs, 1)
				if err != nil {
					return fmt.Errorf("failed to find replacement for pr %s: %w", pr.ID, err)
				}

				if len(candidates) > 0 {
					newReviewerID := candidates[0]
					if err := s.prCmd.ReplaceReviewer(ctx, tx, pr.ID, oldReviewerID, newReviewerID); err != nil {
						return fmt.Errorf("failed to replace reviewer for pr %s: %w", pr.ID, err)
					}

					for i, id := range pr.ReviewerIDs {
						if id == oldReviewerID {
							pr.ReviewerIDs[i] = newReviewerID
							break
						}
					}
				} else {
					log.Warn("no replacement candidate found", "pr_id", pr.ID, "old_reviewer_id", oldReviewerID)
				}
			}
		}
	}

	return nil
}
