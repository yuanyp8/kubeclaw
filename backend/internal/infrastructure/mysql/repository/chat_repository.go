package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	applicationchat "kubeclaw/backend/internal/application/chat"
	mysqlinfra "kubeclaw/backend/internal/infrastructure/mysql"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ChatRepository struct {
	db *gorm.DB
}

func NewChatRepository(db *gorm.DB) *ChatRepository {
	return &ChatRepository{db: db}
}

func (r *ChatRepository) ListSessions(ctx context.Context, userID int64) ([]applicationchat.Session, error) {
	var models []mysqlinfra.ChatSessionModel
	if err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at desc, id desc").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list chat sessions: %w", err)
	}

	result := make([]applicationchat.Session, 0, len(models))
	for _, item := range models {
		result = append(result, toChatSession(item))
	}
	return result, nil
}

func (r *ChatRepository) GetSession(ctx context.Context, sessionID int64) (*applicationchat.Session, error) {
	var model mysqlinfra.ChatSessionModel
	if err := r.db.WithContext(ctx).First(&model, sessionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, applicationchat.ErrSessionNotFound
		}
		return nil, fmt.Errorf("get chat session: %w", err)
	}

	session := toChatSession(model)
	return &session, nil
}

func (r *ChatRepository) CreateSession(ctx context.Context, input applicationchat.CreateSessionInput) (*applicationchat.Session, error) {
	contextPayload, err := json.Marshal(input.Context)
	if err != nil {
		return nil, fmt.Errorf("marshal chat session context: %w", err)
	}

	model := mysqlinfra.ChatSessionModel{
		TenantID:    input.TenantID,
		UserID:      input.UserID,
		Title:       input.Title,
		ContextJSON: datatypes.JSON(contextPayload),
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, fmt.Errorf("create chat session: %w", err)
	}

	session := toChatSession(model)
	return &session, nil
}

func (r *ChatRepository) UpdateSessionContext(ctx context.Context, sessionID int64, sessionContext applicationchat.SessionContext) (*applicationchat.Session, error) {
	var model mysqlinfra.ChatSessionModel
	if err := r.db.WithContext(ctx).First(&model, sessionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, applicationchat.ErrSessionNotFound
		}
		return nil, fmt.Errorf("load chat session: %w", err)
	}

	contextPayload, err := json.Marshal(sessionContext)
	if err != nil {
		return nil, fmt.Errorf("marshal chat session context: %w", err)
	}

	model.ContextJSON = datatypes.JSON(contextPayload)
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, fmt.Errorf("update chat session context: %w", err)
	}

	session := toChatSession(model)
	return &session, nil
}

func (r *ChatRepository) DeleteSession(ctx context.Context, sessionID int64) error {
	if err := r.db.WithContext(ctx).Delete(&mysqlinfra.ChatSessionModel{}, sessionID).Error; err != nil {
		return fmt.Errorf("delete chat session: %w", err)
	}
	return nil
}

func (r *ChatRepository) ListMessages(ctx context.Context, sessionID int64) ([]applicationchat.Message, error) {
	var models []mysqlinfra.ChatMessageModel
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("id asc").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list chat messages: %w", err)
	}

	result := make([]applicationchat.Message, 0, len(models))
	for _, item := range models {
		result = append(result, toChatMessage(item))
	}
	return result, nil
}

func (r *ChatRepository) CreateMessage(ctx context.Context, input applicationchat.CreateMessageInput) (*applicationchat.Message, error) {
	toolCalls, err := marshalJSON(input.ToolCalls)
	if err != nil {
		return nil, fmt.Errorf("marshal tool calls: %w", err)
	}

	model := mysqlinfra.ChatMessageModel{
		SessionID:     input.SessionID,
		Role:          input.Role,
		Content:       input.Content,
		ToolCallsJSON: datatypes.JSON(toolCalls),
		ToolCallID:    input.ToolCallID,
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, fmt.Errorf("create chat message: %w", err)
	}

	message := toChatMessage(model)
	return &message, nil
}

func toChatSession(model mysqlinfra.ChatSessionModel) applicationchat.Session {
	var sessionContext applicationchat.SessionContext
	_ = json.Unmarshal(model.ContextJSON, &sessionContext)

	return applicationchat.Session{
		ID:        model.ID,
		TenantID:  model.TenantID,
		UserID:    model.UserID,
		Title:     model.Title,
		Context:   sessionContext,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func toChatMessage(model mysqlinfra.ChatMessageModel) applicationchat.Message {
	var toolCalls []map[string]any
	_ = json.Unmarshal(model.ToolCallsJSON, &toolCalls)

	return applicationchat.Message{
		ID:         model.ID,
		SessionID:  model.SessionID,
		Role:       model.Role,
		Content:    model.Content,
		ToolCalls:  toolCalls,
		ToolCallID: model.ToolCallID,
		CreatedAt:  model.CreatedAt,
		UpdatedAt:  model.UpdatedAt,
	}
}
