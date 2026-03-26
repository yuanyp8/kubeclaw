package skill

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrNotFound = errors.New("skill not found")

type Record struct {
	ID          int64           `json:"id"`
	TenantID    *int64          `json:"tenantId"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        string          `json:"type"`
	Version     int             `json:"version"`
	Status      string          `json:"status"`
	Definition  json.RawMessage `json:"definition"`
	IsPublic    bool            `json:"isPublic"`
	CreatorID   *int64          `json:"creatorId"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

type CreateInput struct {
	TenantID    *int64          `json:"tenantId"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Type        string          `json:"type"`
	Version     int             `json:"version"`
	Status      string          `json:"status"`
	Definition  json.RawMessage `json:"definition"`
	IsPublic    bool            `json:"isPublic"`
	CreatorID   *int64          `json:"creatorId"`
}

type UpdateInput = CreateInput

type Repository interface {
	List(ctx context.Context) ([]Record, error)
	Get(ctx context.Context, id int64) (*Record, error)
	Create(ctx context.Context, input CreateInput) (*Record, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*Record, error)
	Delete(ctx context.Context, id int64) error
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service                             { return &Service{repo: repo} }
func (s *Service) List(ctx context.Context) ([]Record, error)         { return s.repo.List(ctx) }
func (s *Service) Get(ctx context.Context, id int64) (*Record, error) { return s.repo.Get(ctx, id) }
func (s *Service) Create(ctx context.Context, input CreateInput) (*Record, error) {
	return s.repo.Create(ctx, input)
}
func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*Record, error) {
	return s.repo.Update(ctx, id, input)
}
func (s *Service) Delete(ctx context.Context, id int64) error { return s.repo.Delete(ctx, id) }
