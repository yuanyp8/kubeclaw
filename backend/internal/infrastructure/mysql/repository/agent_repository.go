package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	applicationagent "kubeclaw/backend/internal/application/agent"
	mysqlinfra "kubeclaw/backend/internal/infrastructure/mysql"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type AgentRepository struct {
	db *gorm.DB
}

func NewAgentRepository(db *gorm.DB) *AgentRepository {
	return &AgentRepository{db: db}
}

func (r *AgentRepository) CreateRun(ctx context.Context, run applicationagent.Run) (*applicationagent.Run, error) {
	contextPayload, err := json.Marshal(run.Context)
	if err != nil {
		return nil, fmt.Errorf("marshal run context: %w", err)
	}

	startedAt := time.Now()
	model := mysqlinfra.AgentRunModel{
		SessionID:     run.SessionID,
		UserID:        run.UserID,
		ModelID:       run.ModelID,
		ClusterID:     run.ClusterID,
		Status:        defaultRunStatus(run.Status),
		UserMessageID: run.UserMessageID,
		Input:         run.Input,
		ContextJSON:   datatypes.JSON(contextPayload),
		StartedAt:     &startedAt,
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, fmt.Errorf("create agent run: %w", err)
	}

	record := toAgentRun(model)
	return &record, nil
}

func (r *AgentRepository) GetRun(ctx context.Context, runID int64) (*applicationagent.Run, error) {
	var model mysqlinfra.AgentRunModel
	if err := r.db.WithContext(ctx).First(&model, runID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, applicationagent.ErrRunNotFound
		}
		return nil, fmt.Errorf("get agent run: %w", err)
	}

	record := toAgentRun(model)
	return &record, nil
}

func (r *AgentRepository) UpdateRunStatus(ctx context.Context, runID int64, status string, errorMessage string) error {
	updates := map[string]any{
		"status":        status,
		"error_message": errorMessage,
	}
	if status == applicationagent.StatusRunning {
		now := time.Now()
		updates["started_at"] = &now
	}
	if err := r.db.WithContext(ctx).Model(&mysqlinfra.AgentRunModel{}).Where("id = ?", runID).Updates(updates).Error; err != nil {
		return fmt.Errorf("update agent run status: %w", err)
	}
	return nil
}

func (r *AgentRepository) CompleteRun(ctx context.Context, runID int64, status string, output string, assistantMessageID *int64, errorMessage string) (*applicationagent.Run, error) {
	now := time.Now()
	updates := map[string]any{
		"status":               status,
		"output":               output,
		"assistant_message_id": assistantMessageID,
		"error_message":        errorMessage,
		"finished_at":          &now,
	}
	if err := r.db.WithContext(ctx).Model(&mysqlinfra.AgentRunModel{}).Where("id = ?", runID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("complete agent run: %w", err)
	}
	return r.GetRun(ctx, runID)
}

func (r *AgentRepository) CreateEvent(ctx context.Context, event applicationagent.Event) (*applicationagent.Event, error) {
	payload, err := json.Marshal(event.Payload)
	if err != nil {
		return nil, fmt.Errorf("marshal agent event payload: %w", err)
	}

	model := mysqlinfra.AgentEventModel{
		RunID:       event.RunID,
		SessionID:   event.SessionID,
		EventType:   event.EventType,
		Role:        event.Role,
		Status:      event.Status,
		Message:     event.Message,
		PayloadJSON: datatypes.JSON(payload),
		RequestID:   event.RequestID,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, fmt.Errorf("create agent event: %w", err)
	}

	record := toAgentEvent(model)
	return &record, nil
}

func (r *AgentRepository) ListEvents(ctx context.Context, runID int64) ([]applicationagent.Event, error) {
	var models []mysqlinfra.AgentEventModel
	if err := r.db.WithContext(ctx).Where("run_id = ?", runID).Order("id asc").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list agent events: %w", err)
	}

	result := make([]applicationagent.Event, 0, len(models))
	for _, item := range models {
		result = append(result, toAgentEvent(item))
	}
	return result, nil
}

func (r *AgentRepository) CreateApproval(ctx context.Context, approval applicationagent.ApprovalRequest) (*applicationagent.ApprovalRequest, error) {
	payload, err := json.Marshal(approval.Payload)
	if err != nil {
		return nil, fmt.Errorf("marshal approval payload: %w", err)
	}

	model := mysqlinfra.ApprovalRequestModel{
		RunID:       approval.RunID,
		SessionID:   approval.SessionID,
		UserID:      approval.UserID,
		Type:        approval.Type,
		Title:       approval.Title,
		Reason:      approval.Reason,
		Status:      defaultApprovalStatus(approval.Status),
		PayloadJSON: datatypes.JSON(payload),
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, fmt.Errorf("create approval request: %w", err)
	}

	record := toApproval(model)
	return &record, nil
}

func (r *AgentRepository) GetApproval(ctx context.Context, approvalID int64) (*applicationagent.ApprovalRequest, error) {
	var model mysqlinfra.ApprovalRequestModel
	if err := r.db.WithContext(ctx).First(&model, approvalID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, applicationagent.ErrApprovalNotFound
		}
		return nil, fmt.Errorf("get approval request: %w", err)
	}

	record := toApproval(model)
	return &record, nil
}

func (r *AgentRepository) ResolveApproval(ctx context.Context, approvalID int64, status string, approvedBy *int64) (*applicationagent.ApprovalRequest, error) {
	now := time.Now()
	updates := map[string]any{
		"status":      status,
		"approved_by": approvedBy,
		"resolved_at": &now,
	}
	if err := r.db.WithContext(ctx).Model(&mysqlinfra.ApprovalRequestModel{}).Where("id = ?", approvalID).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("resolve approval request: %w", err)
	}
	return r.GetApproval(ctx, approvalID)
}

func (r *AgentRepository) CreateToolExecution(ctx context.Context, runID *int64, userID *int64, clusterID *int64, toolName string, parameters map[string]any) (int64, error) {
	payload, err := json.Marshal(parameters)
	if err != nil {
		return 0, fmt.Errorf("marshal tool execution parameters: %w", err)
	}

	model := mysqlinfra.ToolExecutionModel{
		RunID:          runID,
		UserID:         userID,
		ClusterID:      clusterID,
		ToolName:       toolName,
		ParametersJSON: datatypes.JSON(payload),
		Status:         "running",
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return 0, fmt.Errorf("create tool execution: %w", err)
	}
	return model.ID, nil
}

func (r *AgentRepository) CompleteToolExecution(ctx context.Context, toolExecutionID int64, status string, result string, durationMS int64) error {
	if err := r.db.WithContext(ctx).Model(&mysqlinfra.ToolExecutionModel{}).Where("id = ?", toolExecutionID).Updates(map[string]any{
		"status":      status,
		"result":      result,
		"duration_ms": durationMS,
	}).Error; err != nil {
		return fmt.Errorf("complete tool execution: %w", err)
	}
	return nil
}

func toAgentRun(model mysqlinfra.AgentRunModel) applicationagent.Run {
	var contextData map[string]any
	_ = json.Unmarshal(model.ContextJSON, &contextData)

	return applicationagent.Run{
		ID:                 model.ID,
		SessionID:          model.SessionID,
		UserID:             model.UserID,
		ModelID:            model.ModelID,
		ClusterID:          model.ClusterID,
		Status:             model.Status,
		UserMessageID:      model.UserMessageID,
		AssistantMessageID: model.AssistantMessageID,
		Input:              model.Input,
		Output:             model.Output,
		ErrorMessage:       model.ErrorMessage,
		Context:            contextData,
		StartedAt:          model.StartedAt,
		FinishedAt:         model.FinishedAt,
		CreatedAt:          model.CreatedAt,
		UpdatedAt:          model.UpdatedAt,
	}
}

func toAgentEvent(model mysqlinfra.AgentEventModel) applicationagent.Event {
	var payload map[string]any
	_ = json.Unmarshal(model.PayloadJSON, &payload)

	return applicationagent.Event{
		ID:        model.ID,
		RunID:     model.RunID,
		SessionID: model.SessionID,
		EventType: model.EventType,
		Role:      model.Role,
		Status:    model.Status,
		Message:   model.Message,
		Payload:   payload,
		RequestID: model.RequestID,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}

func toApproval(model mysqlinfra.ApprovalRequestModel) applicationagent.ApprovalRequest {
	var payload map[string]any
	_ = json.Unmarshal(model.PayloadJSON, &payload)

	return applicationagent.ApprovalRequest{
		ID:         model.ID,
		RunID:      model.RunID,
		SessionID:  model.SessionID,
		UserID:     model.UserID,
		Type:       model.Type,
		Title:      model.Title,
		Reason:     model.Reason,
		Status:     model.Status,
		Payload:    payload,
		ApprovedBy: model.ApprovedBy,
		ResolvedAt: model.ResolvedAt,
		CreatedAt:  model.CreatedAt,
		UpdatedAt:  model.UpdatedAt,
	}
}

func defaultRunStatus(status string) string {
	if status == "" {
		return applicationagent.StatusQueued
	}
	return status
}

func defaultApprovalStatus(status string) string {
	if status == "" {
		return "pending"
	}
	return status
}
