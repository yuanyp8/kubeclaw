package chat

import (
	"context"
	"errors"
	"time"
)

var ErrSessionNotFound = errors.New("chat session not found")

type SessionContext struct {
	ModelID   *int64 `json:"modelId"`
	ClusterID *int64 `json:"clusterId"`
	Namespace string `json:"namespace"`
}

type Session struct {
	ID        int64          `json:"id"`
	TenantID  *int64         `json:"tenantId"`
	UserID    int64          `json:"userId"`
	Title     string         `json:"title"`
	Context   SessionContext `json:"context"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

type Message struct {
	ID         int64            `json:"id"`
	SessionID  int64            `json:"sessionId"`
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []map[string]any `json:"toolCalls"`
	ToolCallID string           `json:"toolCallId"`
	CreatedAt  time.Time        `json:"createdAt"`
	UpdatedAt  time.Time        `json:"updatedAt"`
}

type CreateSessionInput struct {
	TenantID *int64         `json:"tenantId"`
	UserID   int64          `json:"userId"`
	Title    string         `json:"title"`
	Context  SessionContext `json:"context"`
}

type CreateMessageInput struct {
	SessionID  int64            `json:"sessionId"`
	Role       string           `json:"role"`
	Content    string           `json:"content"`
	ToolCalls  []map[string]any `json:"toolCalls"`
	ToolCallID string           `json:"toolCallId"`
}

type Repository interface {
	ListSessions(ctx context.Context, userID int64) ([]Session, error)
	GetSession(ctx context.Context, sessionID int64) (*Session, error)
	CreateSession(ctx context.Context, input CreateSessionInput) (*Session, error)
	UpdateSessionContext(ctx context.Context, sessionID int64, sessionContext SessionContext) (*Session, error)
	DeleteSession(ctx context.Context, sessionID int64) error
	ListMessages(ctx context.Context, sessionID int64) ([]Message, error)
	CreateMessage(ctx context.Context, input CreateMessageInput) (*Message, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) ListSessions(ctx context.Context, userID int64) ([]Session, error) {
	return s.repo.ListSessions(ctx, userID)
}

func (s *Service) GetSession(ctx context.Context, sessionID int64) (*Session, error) {
	return s.repo.GetSession(ctx, sessionID)
}

func (s *Service) CreateSession(ctx context.Context, input CreateSessionInput) (*Session, error) {
	return s.repo.CreateSession(ctx, input)
}

func (s *Service) UpdateSessionContext(ctx context.Context, sessionID int64, sessionContext SessionContext) (*Session, error) {
	return s.repo.UpdateSessionContext(ctx, sessionID, sessionContext)
}

func (s *Service) DeleteSession(ctx context.Context, sessionID int64) error {
	return s.repo.DeleteSession(ctx, sessionID)
}

func (s *Service) ListMessages(ctx context.Context, sessionID int64) ([]Message, error) {
	return s.repo.ListMessages(ctx, sessionID)
}

func (s *Service) CreateMessage(ctx context.Context, input CreateMessageInput) (*Message, error) {
	return s.repo.CreateMessage(ctx, input)
}
