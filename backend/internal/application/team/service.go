package team

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("team not found")

type TenantSummary struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type Record struct {
	ID          int64          `json:"id"`
	TenantID    *int64         `json:"tenantId"`
	Tenant      *TenantSummary `json:"tenant"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	OwnerUserID *int64         `json:"ownerUserId"`
	Visibility  string         `json:"visibility"`
	MemberCount int            `json:"memberCount"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

type MemberRecord struct {
	ID          int64     `json:"id"`
	TeamID      int64     `json:"teamId"`
	UserID      int64     `json:"userId"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	DisplayName string    `json:"displayName"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Input struct {
	TenantID    *int64 `json:"tenantId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	OwnerUserID *int64 `json:"ownerUserId"`
	Visibility  string `json:"visibility"`
}

type AddMemberInput struct {
	UserID int64  `json:"userId"`
	Role   string `json:"role"`
}

type Repository interface {
	List(ctx context.Context) ([]Record, error)
	Get(ctx context.Context, id int64) (*Record, error)
	Create(ctx context.Context, input Input) (*Record, error)
	Update(ctx context.Context, id int64, input Input) (*Record, error)
	Delete(ctx context.Context, id int64) error
	ListMembers(ctx context.Context, teamID int64) ([]MemberRecord, error)
	AddMember(ctx context.Context, teamID int64, input AddMemberInput) (*MemberRecord, error)
	RemoveMember(ctx context.Context, teamID int64, userID int64) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]Record, error) {
	return s.repo.List(ctx)
}

func (s *Service) Get(ctx context.Context, id int64) (*Record, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) Create(ctx context.Context, input Input) (*Record, error) {
	return s.repo.Create(ctx, input)
}

func (s *Service) Update(ctx context.Context, id int64, input Input) (*Record, error) {
	return s.repo.Update(ctx, id, input)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) ListMembers(ctx context.Context, teamID int64) ([]MemberRecord, error) {
	return s.repo.ListMembers(ctx, teamID)
}

func (s *Service) AddMember(ctx context.Context, teamID int64, input AddMemberInput) (*MemberRecord, error) {
	return s.repo.AddMember(ctx, teamID, input)
}

func (s *Service) RemoveMember(ctx context.Context, teamID int64, userID int64) error {
	return s.repo.RemoveMember(ctx, teamID, userID)
}
