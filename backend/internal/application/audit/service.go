package audit

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("audit log not found")

// Record 表示审计日志记录。
type Record struct {
	ID        int64     `json:"id"`
	TenantID  *int64    `json:"tenantId"`
	UserID    *int64    `json:"userId"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	Details   string    `json:"details"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// CreateInput 表示创建审计日志时的输入。
type CreateInput struct {
	TenantID *int64 `json:"tenantId"`
	UserID   *int64 `json:"userId"`
	Action   string `json:"action"`
	Target   string `json:"target"`
	Details  string `json:"details"`
	IP       string `json:"ip"`
}

type Repository interface {
	List(ctx context.Context) ([]Record, error)
	Get(ctx context.Context, id int64) (*Record, error)
	Create(ctx context.Context, input CreateInput) (*Record, error)
}

// Service 负责审计日志的查询与记录。
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

func (s *Service) Create(ctx context.Context, input CreateInput) (*Record, error) {
	return s.repo.Create(ctx, input)
}
