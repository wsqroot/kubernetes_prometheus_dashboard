package service

import (
	"context"
	"errors"
	"fmt"

	"login/internal/repository"
)

var ErrInvalidCredentials = errors.New("invalid credentials")

type userFinder interface {
	FindByUsername(ctx context.Context, username string) (*UserRecordView, error)
}

type UserRecordView struct {
	ID           int64
	Username     string
	PasswordText string
}

type repoAdapter struct {
	repo *repository.UserRepository
}

func (a repoAdapter) FindByUsername(ctx context.Context, username string) (*UserRecordView, error) {
	record, err := a.repo.FindByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return &UserRecordView{
		ID:           record.ID,
		Username:     record.Username,
		PasswordText: record.PasswordText,
	}, nil
}

type AuthService struct {
	repo         userFinder
	passwordMode string
}

func NewAuthService(repo *repository.UserRepository, passwordMode string) *AuthService {
	return &AuthService{
		repo:         repoAdapter{repo: repo},
		passwordMode: passwordMode,
	}
}

func (s *AuthService) Login(ctx context.Context, username, password string) (int64, string, error) {
	user, err := s.repo.FindByUsername(ctx, username)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) {
			return 0, "", ErrInvalidCredentials
		}
		return 0, "", fmt.Errorf("find user: %w", err)
	}

	switch s.passwordMode {
	case "", "plaintext":
		if user.PasswordText != password {
			return 0, "", ErrInvalidCredentials
		}
	default:
		return 0, "", fmt.Errorf("unsupported password mode: %s", s.passwordMode)
	}

	return user.ID, user.Username, nil
}
