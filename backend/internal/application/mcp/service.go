package mcp

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("mcp server not found")

type Record struct {
	ID           int64             `json:"id"`
	TenantID     *int64            `json:"tenantId"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Transport    string            `json:"transport"`
	Endpoint     string            `json:"endpoint"`
	Command      string            `json:"command"`
	Args         []string          `json:"args"`
	Headers      map[string]string `json:"headers"`
	AuthType     string            `json:"authType"`
	Description  string            `json:"description"`
	HealthStatus string            `json:"healthStatus"`
	IsEnabled    bool              `json:"isEnabled"`
	HasSecret    bool              `json:"hasSecret"`
	MaskedSecret string            `json:"maskedSecret"`
	CreatedAt    time.Time         `json:"createdAt"`
	UpdatedAt    time.Time         `json:"updatedAt"`
}

type CreateInput struct {
	TenantID     *int64            `json:"tenantId"`
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	Transport    string            `json:"transport"`
	Endpoint     string            `json:"endpoint"`
	Command      string            `json:"command"`
	Args         []string          `json:"args"`
	Headers      map[string]string `json:"headers"`
	AuthType     string            `json:"authType"`
	Secret       string            `json:"secret"`
	Description  string            `json:"description"`
	HealthStatus string            `json:"healthStatus"`
	IsEnabled    bool              `json:"isEnabled"`
}

type UpdateInput = CreateInput

type Repository interface {
	List(ctx context.Context) ([]Record, error)
	Get(ctx context.Context, id int64) (*Record, error)
	Create(ctx context.Context, input CreateInput) (*Record, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*Record, error)
	Delete(ctx context.Context, id int64) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]Record, error)         { return s.repo.List(ctx) }
func (s *Service) Get(ctx context.Context, id int64) (*Record, error) { return s.repo.Get(ctx, id) }
func (s *Service) Create(ctx context.Context, input CreateInput) (*Record, error) {
	return s.repo.Create(ctx, input)
}
func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*Record, error) {
	return s.repo.Update(ctx, id, input)
}
func (s *Service) Delete(ctx context.Context, id int64) error { return s.repo.Delete(ctx, id) }
