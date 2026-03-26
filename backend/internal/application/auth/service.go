package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	domainauth "kubeclaw/backend/internal/domain/auth"
	domainuser "kubeclaw/backend/internal/domain/user"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrInvalidAccessToken  = errors.New("invalid access token")
	ErrInactiveUser        = errors.New("inactive user")
)

type loginActivityRecorder interface {
	UpdateLastLoginAt(ctx context.Context, userID int64, loginAt time.Time) error
}

// Service 负责认证相关用例编排。
type Service struct {
	userRepo     domainuser.Repository
	tokenManager domainauth.TokenManager
}

type UserProfile struct {
	ID          int64      `json:"id"`
	TenantID    *int64     `json:"tenantId"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	DisplayName string     `json:"displayName"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	LastLoginAt *time.Time `json:"lastLoginAt"`
	Enabled     bool       `json:"enabled"`
}

type LoginResult struct {
	User   UserProfile          `json:"user"`
	Tokens domainauth.TokenPair `json:"tokens"`
}

func NewService(userRepo domainuser.Repository, tokenManager domainauth.TokenManager) *Service {
	return &Service{
		userRepo:     userRepo,
		tokenManager: tokenManager,
	}
}

func (s *Service) Login(ctx context.Context, login, password string) (*LoginResult, error) {
	userEntity, err := s.userRepo.FindByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, domainuser.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("find user by login: %w", err)
	}

	if !userEntity.IsActive() {
		return nil, ErrInactiveUser
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userEntity.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	if recorder, ok := s.userRepo.(loginActivityRecorder); ok {
		_ = recorder.UpdateLastLoginAt(ctx, userEntity.ID, time.Now())
	}

	return s.issueLoginResult(*userEntity)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*LoginResult, error) {
	claims, err := s.tokenManager.Parse(refreshToken)
	if err != nil || claims.TokenType != domainauth.TokenTypeRefresh {
		return nil, ErrInvalidRefreshToken
	}

	userEntity, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, domainuser.ErrNotFound) {
			return nil, ErrInvalidRefreshToken
		}

		return nil, fmt.Errorf("find user by id when refresh token: %w", err)
	}

	if !userEntity.IsActive() {
		return nil, ErrInactiveUser
	}

	return s.issueLoginResult(*userEntity)
}

func (s *Service) AuthenticateAccessToken(ctx context.Context, accessToken string) (*domainuser.User, error) {
	claims, err := s.tokenManager.Parse(accessToken)
	if err != nil || claims.TokenType != domainauth.TokenTypeAccess {
		return nil, ErrInvalidAccessToken
	}

	userEntity, err := s.userRepo.FindByID(ctx, claims.UserID)
	if err != nil {
		if errors.Is(err, domainuser.ErrNotFound) {
			return nil, ErrInvalidAccessToken
		}

		return nil, fmt.Errorf("find user by id when authenticate access token: %w", err)
	}

	if !userEntity.IsActive() {
		return nil, ErrInactiveUser
	}

	return userEntity, nil
}

func (s *Service) issueLoginResult(userEntity domainuser.User) (*LoginResult, error) {
	tokens, err := s.tokenManager.Issue(domainauth.Identity{
		UserID:   userEntity.ID,
		Username: userEntity.Username,
		Role:     string(userEntity.Role),
	})
	if err != nil {
		return nil, fmt.Errorf("issue token pair: %w", err)
	}

	return &LoginResult{
		User: UserProfile{
			ID:          userEntity.ID,
			TenantID:    userEntity.TenantID,
			Username:    userEntity.Username,
			Email:       userEntity.Email,
			DisplayName: userEntity.DisplayName,
			Role:        string(userEntity.Role),
			Status:      userEntity.Status,
			LastLoginAt: userEntity.LastLoginAt,
			Enabled:     userEntity.Enabled,
		},
		Tokens: tokens,
	}, nil
}
