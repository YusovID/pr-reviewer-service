package service

import (
	"context"
	"fmt"

	"github.com/YusovID/pr-reviewer-service/internal/repository"
	"github.com/YusovID/pr-reviewer-service/pkg/api"
)

type UserService interface {
	SetIsActive(ctx context.Context, userID string, isActive bool) (*api.User, error)
}

type UserServiceImpl struct {
	repo repository.UserRepository
}

func NewUserService(repo repository.UserRepository) *UserServiceImpl {
	return &UserServiceImpl{repo: repo}
}

func (s *UserServiceImpl) SetIsActive(ctx context.Context, userID string, isActive bool) (*api.User, error) {
	user, err := s.repo.SetIsActive(ctx, userID, isActive)
	if err != nil {
		return nil, fmt.Errorf("repo.SetIsActive failed: %w", err)
	}

	return user, nil
}
