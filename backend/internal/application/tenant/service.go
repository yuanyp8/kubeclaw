package tenant

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("tenant not found")

type Record struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Slug        string    `json:"slug"`
	Description string    `json:"description"`
	Status      string    `json:"status"`
	IsSystem    bool      `json:"isSystem"`
	OwnerUserID *int64    `json:"ownerUserId"`
	UserCount   int       `json:"userCount"`
	TeamCount   int       `json:"teamCount"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Input struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Status      string `json:"status"`
	IsSystem    bool   `json:"isSystem"`
	OwnerUserID *int64 `json:"ownerUserId"`
}

type Repository interface {
	List(ctx context.Context) ([]Record, error)
	Get(ctx context.Context, id int64) (*Record, error)
	Create(ctx context.Context, input Input) (*Record, error)
	Update(ctx context.Context, id int64, input Input) (*Record, error)
	Delete(ctx context.Context, id int64) error
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
