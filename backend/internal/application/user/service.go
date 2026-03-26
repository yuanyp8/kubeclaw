package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	domainuser "kubeclaw/backend/internal/domain/user"

	"golang.org/x/crypto/bcrypt"
)

var ErrNotFound = errors.New("user not found")

type TenantSummary struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type TeamMembership struct {
	TeamID   int64  `json:"teamId"`
	TeamName string `json:"teamName"`
	Role     string `json:"role"`
}

type Profile struct {
	ID          int64            `json:"id"`
	TenantID    *int64           `json:"tenantId"`
	Tenant      *TenantSummary   `json:"tenant"`
	Teams       []TeamMembership `json:"teams"`
	Username    string           `json:"username"`
	Email       string           `json:"email"`
	DisplayName string           `json:"displayName"`
	Phone       string           `json:"phone"`
	AvatarURL   string           `json:"avatarUrl"`
	Role        string           `json:"role"`
	Status      string           `json:"status"`
	LastLoginAt *time.Time       `json:"lastLoginAt"`
	Enabled     bool             `json:"enabled"`
	CreatedAt   time.Time        `json:"createdAt"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

type CreateInput struct {
	TenantID     *int64
	Username     string
	Email        string
	DisplayName  string
	Phone        string
	AvatarURL    string
	Role         string
	Status       string
	Password     string
	PasswordHash string
}

type UpdateInput struct {
	TenantID     *int64
	Email        string
	DisplayName  string
	Phone        string
	AvatarURL    string
	Role         string
	Status       string
	Password     string
	PasswordHash string
}

type Repository interface {
	domainuser.Repository
	List(ctx context.Context) ([]Profile, error)
	Get(ctx context.Context, id int64) (*Profile, error)
	Create(ctx context.Context, input CreateInput) (*Profile, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*Profile, error)
	Delete(ctx context.Context, id int64) error
}

type Service struct {
	userRepo Repository
}

func NewService(userRepo Repository) *Service {
	return &Service{userRepo: userRepo}
}

func (s *Service) GetProfile(ctx context.Context, userID int64) (*Profile, error) {
	profile, err := s.userRepo.Get(ctx, userID)
	if err != nil {
		if errors.Is(err, ErrNotFound) || errors.Is(err, domainuser.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}

	return profile, nil
}

func (s *Service) List(ctx context.Context) ([]Profile, error) {
	return s.userRepo.List(ctx)
}

func (s *Service) Get(ctx context.Context, id int64) (*Profile, error) {
	return s.userRepo.Get(ctx, id)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*Profile, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash user password: %w", err)
	}

	input.PasswordHash = string(hash)
	return s.userRepo.Create(ctx, input)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*Profile, error) {
	if input.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash updated password: %w", err)
		}

		input.PasswordHash = string(hash)
	}

	return s.userRepo.Update(ctx, id, input)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.userRepo.Delete(ctx, id)
}
