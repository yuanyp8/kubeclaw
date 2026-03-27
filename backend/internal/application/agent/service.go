package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	applicationcapability "kubeclaw/backend/internal/application/capability"
	applicationchat "kubeclaw/backend/internal/application/chat"
	applicationcluster "kubeclaw/backend/internal/application/cluster"
	applicationmcp "kubeclaw/backend/internal/application/mcp"
	applicationmodel "kubeclaw/backend/internal/application/model"
	applicationsecurity "kubeclaw/backend/internal/application/security"
	appskill "kubeclaw/backend/internal/application/skill"
	"kubeclaw/backend/internal/infrastructure/llm"
	"kubeclaw/backend/internal/logger"

	"go.uber.org/zap"
)

var (
	ErrRunNotFound      = errors.New("agent run not found")
	ErrApprovalNotFound = errors.New("approval request not found")
	thinkBlockPattern   = regexp.MustCompile(`(?is)<think>.*?</think>`)
)

const (
	StatusQueued          = "queued"
	StatusRunning         = "running"
	StatusCompleted       = "completed"
	StatusFailed          = "failed"
	StatusWaitingApproval = "waiting_approval"
	StatusRejected        = "rejected"
)

type Run struct {
	ID                 int64          `json:"id"`
	SessionID          int64          `json:"sessionId"`
	UserID             int64          `json:"userId"`
	ModelID            *int64         `json:"modelId"`
	ClusterID          *int64         `json:"clusterId"`
	Status             string         `json:"status"`
	UserMessageID      *int64         `json:"userMessageId"`
	AssistantMessageID *int64         `json:"assistantMessageId"`
	Input              string         `json:"input"`
	Output             string         `json:"output"`
	ErrorMessage       string         `json:"errorMessage"`
	Context            map[string]any `json:"context"`
	StartedAt          *time.Time     `json:"startedAt"`
	FinishedAt         *time.Time     `json:"finishedAt"`
	CreatedAt          time.Time      `json:"createdAt"`
	UpdatedAt          time.Time      `json:"updatedAt"`
}

type Event struct {
	ID        int64          `json:"id"`
	RunID     int64          `json:"runId"`
	SessionID int64          `json:"sessionId"`
	EventType string         `json:"eventType"`
	Role      string         `json:"role"`
	Status    string         `json:"status"`
	Message   string         `json:"message"`
	Payload   map[string]any `json:"payload"`
	RequestID string         `json:"requestId"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
}

type ApprovalRequest struct {
	ID         int64          `json:"id"`
	RunID      int64          `json:"runId"`
	SessionID  int64          `json:"sessionId"`
	UserID     int64          `json:"userId"`
	Type       string         `json:"type"`
	Title      string         `json:"title"`
	Reason     string         `json:"reason"`
	Status     string         `json:"status"`
	Payload    map[string]any `json:"payload"`
	ApprovedBy *int64         `json:"approvedBy"`
	ResolvedAt *time.Time     `json:"resolvedAt"`
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}

type SendMessageInput struct {
	SessionID int64  `json:"sessionId"`
	UserID    int64  `json:"userId"`
	Content   string `json:"content"`
	RequestID string `json:"requestId"`
}

type SendMessageResult struct {
	SessionID     int64  `json:"sessionId"`
	UserMessageID int64  `json:"userMessageId"`
	RunID         int64  `json:"runId"`
	Status        string `json:"status"`
}

type ClusterActionRequestInput struct {
	ClusterID    int64  `json:"clusterId"`
	TenantID     *int64 `json:"tenantId"`
	UserID       int64  `json:"userId"`
	ResourceType string `json:"resourceType"`
	ResourceName string `json:"resourceName"`
	Namespace    string `json:"namespace"`
	Replicas     int32  `json:"replicas"`
	Manifest     string `json:"manifest"`
	Action       string `json:"action"`
	RequestID    string `json:"requestId"`
}

type CreateSessionInput struct {
	TenantID *int64                         `json:"tenantId"`
	UserID   int64                          `json:"userId"`
	Title    string                         `json:"title"`
	Context  applicationchat.SessionContext `json:"context"`
}

type Repository interface {
	CreateRun(ctx context.Context, run Run) (*Run, error)
	GetRun(ctx context.Context, runID int64) (*Run, error)
	UpdateRunStatus(ctx context.Context, runID int64, status string, errorMessage string) error
	CompleteRun(ctx context.Context, runID int64, status string, output string, assistantMessageID *int64, errorMessage string) (*Run, error)
	CreateEvent(ctx context.Context, event Event) (*Event, error)
	ListEvents(ctx context.Context, runID int64) ([]Event, error)
	CreateApproval(ctx context.Context, approval ApprovalRequest) (*ApprovalRequest, error)
	GetApproval(ctx context.Context, approvalID int64) (*ApprovalRequest, error)
	ResolveApproval(ctx context.Context, approvalID int64, status string, approvedBy *int64) (*ApprovalRequest, error)
	CreateToolExecution(ctx context.Context, runID *int64, userID *int64, clusterID *int64, toolName string, parameters map[string]any) (int64, error)
	CompleteToolExecution(ctx context.Context, toolExecutionID int64, status string, result string, durationMS int64) error
}

type ChatService interface {
	ListSessions(ctx context.Context, userID int64) ([]applicationchat.Session, error)
	GetSession(ctx context.Context, sessionID int64) (*applicationchat.Session, error)
	CreateSession(ctx context.Context, input applicationchat.CreateSessionInput) (*applicationchat.Session, error)
	UpdateSessionContext(ctx context.Context, sessionID int64, sessionContext applicationchat.SessionContext) (*applicationchat.Session, error)
	DeleteSession(ctx context.Context, sessionID int64) error
	ListMessages(ctx context.Context, sessionID int64) ([]applicationchat.Message, error)
	CreateMessage(ctx context.Context, input applicationchat.CreateMessageInput) (*applicationchat.Message, error)
}

type ModelService interface {
	List(ctx context.Context) ([]applicationmodel.Record, error)
	Resolve(ctx context.Context, id int64) (*applicationmodel.ResolvedRecord, error)
	ResolveDefault(ctx context.Context) (*applicationmodel.ResolvedRecord, error)
}

type ClusterService interface {
	ListNamespaces(ctx context.Context, clusterID int64) ([]applicationcluster.NamespaceRecord, error)
	ListResources(ctx context.Context, clusterID int64, query applicationcluster.ResourceQuery) ([]applicationcluster.ResourceRecord, error)
	ListEvents(ctx context.Context, clusterID int64, namespace string) ([]applicationcluster.EventRecord, error)
	DeleteResource(ctx context.Context, clusterID int64, query applicationcluster.ResourceQuery, name string) error
	ScaleDeployment(ctx context.Context, clusterID int64, namespace string, name string, replicas int32) error
	RestartDeployment(ctx context.Context, clusterID int64, namespace string, name string) error
	ApplyYAML(ctx context.Context, clusterID int64, manifest string) (*applicationcluster.ApplyResult, error)
}

type SkillService interface {
	List(ctx context.Context) ([]appskill.Record, error)
}

type MCPService interface {
	List(ctx context.Context) ([]applicationmcp.Record, error)
}

type SecurityService interface {
	ListSensitiveWords(ctx context.Context) ([]applicationsecurity.SensitiveWordRecord, error)
}

type LLMClient interface {
	Chat(ctx context.Context, input llm.ChatInput) (*llm.ChatResult, error)
}

type StreamHub interface {
	Publish(id int64, value Event)
}

type Service struct {
	repo     Repository
	chat     ChatService
	models   ModelService
	clusters ClusterService
	caps     *applicationcapability.Service
	skills   SkillService
	mcp      MCPService
	security SecurityService
	llm      LLMClient
	streams  StreamHub
	log      *zap.Logger
}

func NewService(
	repo Repository,
	chat ChatService,
	models ModelService,
	clusters ClusterService,
	capabilities *applicationcapability.Service,
	skills SkillService,
	mcp MCPService,
	security SecurityService,
	llmClient LLMClient,
	streams StreamHub,
) *Service {
	return &Service{
		repo:     repo,
		chat:     chat,
		models:   models,
		clusters: clusters,
		caps:     capabilities,
		skills:   skills,
		mcp:      mcp,
		security: security,
		llm:      llmClient,
		streams:  streams,
		log:      logger.ForScope(logger.ScopeAgent),
	}
}

func (s *Service) CreateSession(ctx context.Context, input CreateSessionInput) (*applicationchat.Session, error) {
	if strings.TrimSpace(input.Title) == "" {
		input.Title = "New agent session"
	}
	return s.chat.CreateSession(ctx, applicationchat.CreateSessionInput{
		TenantID: input.TenantID,
		UserID:   input.UserID,
		Title:    input.Title,
		Context:  input.Context,
	})
}

func (s *Service) ListSessions(ctx context.Context, userID int64) ([]applicationchat.Session, error) {
	return s.chat.ListSessions(ctx, userID)
}

func (s *Service) GetSession(ctx context.Context, sessionID int64) (*applicationchat.Session, error) {
	return s.chat.GetSession(ctx, sessionID)
}

func (s *Service) DeleteSession(ctx context.Context, sessionID int64) error {
	return s.chat.DeleteSession(ctx, sessionID)
}

func (s *Service) ListMessages(ctx context.Context, sessionID int64) ([]applicationchat.Message, error) {
	return s.chat.ListMessages(ctx, sessionID)
}

func (s *Service) ListRunEvents(ctx context.Context, runID int64) ([]Event, error) {
	return s.repo.ListEvents(ctx, runID)
}

func (s *Service) GetRun(ctx context.Context, runID int64) (*Run, error) {
	return s.repo.GetRun(ctx, runID)
}

func (s *Service) GetApproval(ctx context.Context, approvalID int64) (*ApprovalRequest, error) {
	return s.repo.GetApproval(ctx, approvalID)
}

func (s *Service) SendMessage(ctx context.Context, input SendMessageInput) (*SendMessageResult, error) {
	session, err := s.chat.GetSession(ctx, input.SessionID)
	if err != nil {
		return nil, err
	}

	if session.Context.ModelID == nil {
		if defaultModel, resolveErr := s.models.ResolveDefault(ctx); resolveErr == nil {
			session.Context.ModelID = &defaultModel.ID
			session, _ = s.chat.UpdateSessionContext(ctx, session.ID, session.Context)
		}
	}

	userMessage, err := s.chat.CreateMessage(ctx, applicationchat.CreateMessageInput{
		SessionID: input.SessionID,
		Role:      "user",
		Content:   input.Content,
	})
	if err != nil {
		return nil, err
	}

	run := Run{
		SessionID:     session.ID,
		UserID:        input.UserID,
		ModelID:       session.Context.ModelID,
		ClusterID:     session.Context.ClusterID,
		Status:        StatusQueued,
		UserMessageID: &userMessage.ID,
		Input:         input.Content,
		Context: map[string]any{
			"requestId": input.RequestID,
			"namespace": session.Context.Namespace,
		},
	}

	createdRun, err := s.repo.CreateRun(ctx, run)
	if err != nil {
		return nil, err
	}

	go s.executeRun(context.Background(), *createdRun, *session, *userMessage, input.RequestID)

	return &SendMessageResult{
		SessionID:     session.ID,
		UserMessageID: userMessage.ID,
		RunID:         createdRun.ID,
		Status:        createdRun.Status,
	}, nil
}

func (s *Service) RequestClusterAction(ctx context.Context, input ClusterActionRequestInput) (*SendMessageResult, error) {
	session, err := s.chat.CreateSession(ctx, applicationchat.CreateSessionInput{
		TenantID: input.TenantID,
		UserID:   input.UserID,
		Title:    buildActionSessionTitle(input),
		Context: applicationchat.SessionContext{
			ClusterID: &input.ClusterID,
			Namespace: defaultString(input.Namespace, "default"),
		},
	})
	if err != nil {
		return nil, err
	}

	return s.SendMessage(ctx, SendMessageInput{
		SessionID: session.ID,
		UserID:    input.UserID,
		Content:   buildClusterActionMessage(input),
		RequestID: input.RequestID,
	})
}

func (s *Service) Approve(ctx context.Context, approvalID int64, approverID int64) (*ApprovalRequest, error) {
	approval, err := s.repo.ResolveApproval(ctx, approvalID, "approved", &approverID)
	if err != nil {
		return nil, err
	}

	run, err := s.repo.GetRun(ctx, approval.RunID)
	if err != nil {
		return nil, err
	}
	session, err := s.chat.GetSession(ctx, approval.SessionID)
	if err != nil {
		return nil, err
	}

	go s.executeApprovedAction(context.Background(), *approval, *run, *session)

	return approval, nil
}

func (s *Service) Reject(ctx context.Context, approvalID int64, approverID int64) (*ApprovalRequest, error) {
	approval, err := s.repo.ResolveApproval(ctx, approvalID, "rejected", &approverID)
	if err != nil {
		return nil, err
	}

	assistantMessage, createErr := s.chat.CreateMessage(ctx, applicationchat.CreateMessageInput{
		SessionID: approval.SessionID,
		Role:      "assistant",
		Content:   "The requested action was rejected and no change was executed.",
	})
	if createErr != nil {
		return nil, createErr
	}

	_, err = s.repo.CompleteRun(ctx, approval.RunID, StatusRejected, assistantMessage.Content, &assistantMessage.ID, "approval rejected")
	if err != nil {
		return nil, err
	}

	s.publishEvent(ctx, approval.RunID, approval.SessionID, "turn_end", "safety_reviewer", StatusRejected, "approval rejected", map[string]any{
		"approvalId": approval.ID,
	})

	return approval, nil
}

func (s *Service) executeRun(ctx context.Context, run Run, session applicationchat.Session, userMessage applicationchat.Message, requestID string) {
	now := time.Now()
	_ = s.repo.UpdateRunStatus(ctx, run.ID, StatusRunning, "")
	run.Status = StatusRunning
	run.StartedAt = &now

	s.publishEvent(ctx, run.ID, run.SessionID, "turn_start", "orchestrator", StatusRunning, "agent run started", map[string]any{
		"userMessageId": userMessage.ID,
	}, requestID)

	intent := s.analyzeIntent(ctx, session, userMessage.Content)
	s.publishEvent(ctx, run.ID, run.SessionID, "planning", "orchestrator", StatusRunning, "execution plan created", map[string]any{
		"intent":         intent.Kind,
		"tool":           intent.Tool,
		"risk":           intent.RequiresApproval,
		"capabilityType": intent.CapabilityType,
		"capabilityId":   intent.CapabilityID,
		"capabilityName": intent.CapabilityName,
	}, requestID)

	if intent.RequiresApproval {
		approval, err := s.repo.CreateApproval(ctx, ApprovalRequest{
			RunID:     run.ID,
			SessionID: run.SessionID,
			UserID:    run.UserID,
			Type:      defaultString(intent.ApprovalType, intent.Kind),
			Title:     intent.Title,
			Reason:    intent.Reason,
			Status:    "pending",
			Payload:   intent.Payload,
		})
		if err != nil {
			s.failRun(ctx, run, session, requestID, err)
			return
		}

		_ = s.repo.UpdateRunStatus(ctx, run.ID, StatusWaitingApproval, "")
		s.publishEvent(ctx, run.ID, run.SessionID, "approval_required", "safety_reviewer", StatusWaitingApproval, approval.Reason, map[string]any{
			"approvalId": approval.ID,
			"type":       approval.Type,
			"title":      approval.Title,
			"payload":    approval.Payload,
		}, requestID)
		s.publishEvent(ctx, run.ID, run.SessionID, "turn_end", "orchestrator", StatusWaitingApproval, "waiting for approval", map[string]any{
			"approvalId": approval.ID,
		}, requestID)
		return
	}

	content, err := s.executeIntent(ctx, run, session, userMessage, intent, requestID)
	if err != nil {
		s.failRun(ctx, run, session, requestID, err)
		return
	}

	assistantMessage, err := s.chat.CreateMessage(ctx, applicationchat.CreateMessageInput{
		SessionID: run.SessionID,
		Role:      "assistant",
		Content:   content,
	})
	if err != nil {
		s.failRun(ctx, run, session, requestID, err)
		return
	}

	if _, err := s.repo.CompleteRun(ctx, run.ID, StatusCompleted, content, &assistantMessage.ID, ""); err != nil {
		s.failRun(ctx, run, session, requestID, err)
		return
	}

	s.publishEvent(ctx, run.ID, run.SessionID, "message_delta", "orchestrator", StatusRunning, content, nil, requestID)
	s.publishEvent(ctx, run.ID, run.SessionID, "message_done", "orchestrator", StatusCompleted, "assistant message stored", map[string]any{
		"assistantMessageId": assistantMessage.ID,
	}, requestID)
	s.publishEvent(ctx, run.ID, run.SessionID, "turn_end", "orchestrator", StatusCompleted, "agent run completed", nil, requestID)
}

func (s *Service) executeApprovedAction(ctx context.Context, approval ApprovalRequest, run Run, session applicationchat.Session) {
	requestID, _ := run.Context["requestId"].(string)

	s.publishEvent(ctx, run.ID, run.SessionID, "agent_spawn", "safety_reviewer", StatusRunning, "approval accepted, executing action", map[string]any{
		"approvalId": approval.ID,
		"type":       approval.Type,
	}, requestID)

	approvedIntent := approvalIntentFromPayload(approval.Payload)
	userMessage := applicationchat.Message{
		ID:        defaultInt64(run.UserMessageID),
		SessionID: run.SessionID,
		Role:      "user",
		Content:   run.Input,
	}
	content, err := s.executeIntent(ctx, run, session, userMessage, approvedIntent, requestID)
	if err != nil {
		s.failRun(ctx, run, session, requestID, err)
		return
	}

	assistantMessage, err := s.chat.CreateMessage(ctx, applicationchat.CreateMessageInput{
		SessionID: run.SessionID,
		Role:      "assistant",
		Content:   content,
	})
	if err != nil {
		s.failRun(ctx, run, session, requestID, err)
		return
	}

	if _, err := s.repo.CompleteRun(ctx, run.ID, StatusCompleted, content, &assistantMessage.ID, ""); err != nil {
		s.failRun(ctx, run, session, requestID, err)
		return
	}

	s.publishEvent(ctx, run.ID, run.SessionID, "message_done", "orchestrator", StatusCompleted, "approved action completed", map[string]any{
		"assistantMessageId": assistantMessage.ID,
	}, requestID)
	s.publishEvent(ctx, run.ID, run.SessionID, "turn_end", "orchestrator", StatusCompleted, "agent run completed", nil, requestID)
}

func (s *Service) executeIntent(ctx context.Context, run Run, session applicationchat.Session, userMessage applicationchat.Message, intent intent, requestID string) (string, error) {
	if intent.Kind == "llm" || intent.CapabilityType == "llm" {
		return s.executeLLMIntent(ctx, session, userMessage)
	}
	return s.executeCapabilityIntent(ctx, run, session, userMessage, intent, requestID)
}

func (s *Service) executeToolIntent(ctx context.Context, run Run, session applicationchat.Session, intent intent, requestID string) (string, error) {
	clusterID := session.Context.ClusterID
	toolExecutionID, err := s.repo.CreateToolExecution(ctx, &run.ID, &run.UserID, clusterID, intent.Tool, intent.Payload)
	if err != nil {
		return "", err
	}

	startedAt := time.Now()
	s.publishEvent(ctx, run.ID, run.SessionID, "agent_spawn", s.intentRole(intent), StatusRunning, "specialist picked the task", map[string]any{
		"tool": intent.Tool,
	}, requestID)
	s.publishEvent(ctx, run.ID, run.SessionID, "tool_start", s.intentRole(intent), StatusRunning, "tool execution started", map[string]any{
		"toolExecutionId": toolExecutionID,
		"tool":            intent.Tool,
	}, requestID)

	result, execErr := s.runTool(ctx, session, intent)
	status := "succeeded"
	if execErr != nil {
		status = "failed"
	}
	_ = s.repo.CompleteToolExecution(ctx, toolExecutionID, status, result, time.Since(startedAt).Milliseconds())
	if execErr != nil {
		return "", execErr
	}

	s.publishEvent(ctx, run.ID, run.SessionID, "tool_end", s.intentRole(intent), StatusCompleted, "tool execution completed", map[string]any{
		"toolExecutionId": toolExecutionID,
		"tool":            intent.Tool,
		"result":          result,
	}, requestID)
	s.publishEvent(ctx, run.ID, run.SessionID, "agent_result", s.intentRole(intent), StatusCompleted, "specialist returned a result", map[string]any{
		"tool": intent.Tool,
	}, requestID)

	return result, nil
}

func (s *Service) executeLLMIntent(ctx context.Context, session applicationchat.Session, userMessage applicationchat.Message) (string, error) {
	resolvedModel, err := s.resolveSessionModel(ctx, session.Context.ModelID)
	if err != nil {
		return "", err
	}

	history, err := s.chat.ListMessages(ctx, session.ID)
	if err != nil {
		return "", err
	}

	messages := []llm.Message{
		{
			Role:    "system",
			Content: orchestratorSystemPrompt(userMessage.Content),
		},
	}
	for _, item := range history {
		if item.Role == "user" || item.Role == "assistant" || item.Role == "system" {
			messages = append(messages, llm.Message{
				Role:    item.Role,
				Content: item.Content,
			})
		}
	}
	if len(history) == 0 || history[len(history)-1].ID != userMessage.ID {
		messages = append(messages, llm.Message{
			Role:    "user",
			Content: userMessage.Content,
		})
	}

	result, err := s.llm.Chat(ctx, llm.ChatInput{
		Model:    *resolvedModel,
		Messages: messages,
	})
	if err != nil {
		return "", fmt.Errorf("model %s chat failed: %w", defaultString(resolvedModel.Model, resolvedModel.Name), err)
	}

	content, strippedThink := sanitizeLLMOutput(result.Content)
	if content == "" && strippedThink {
		return "", fmt.Errorf("model %s returned reasoning only without a final answer", defaultString(resolvedModel.Model, resolvedModel.Name))
	}
	if content == "" {
		return "", fmt.Errorf("model %s returned an empty answer", defaultString(resolvedModel.Model, resolvedModel.Name))
	}

	return content, nil
}

func (s *Service) executeApprovedPayload(ctx context.Context, run Run, session applicationchat.Session, approval ApprovalRequest) (string, error) {
	clusterID := session.Context.ClusterID
	if clusterID == nil {
		return "", fmt.Errorf("cluster context is required for this action")
	}

	toolExecutionID, err := s.repo.CreateToolExecution(ctx, &run.ID, &run.UserID, clusterID, approval.Type, approval.Payload)
	if err != nil {
		return "", err
	}

	startedAt := time.Now()
	status := "succeeded"
	var result string

	switch approval.Type {
	case "delete_resource":
		resourceType, _ := approval.Payload["type"].(string)
		name, _ := approval.Payload["name"].(string)
		namespace, _ := approval.Payload["namespace"].(string)
		err = s.clusters.DeleteResource(ctx, *clusterID, applicationcluster.ResourceQuery{
			Type:      resourceType,
			Namespace: namespace,
		}, name)
		result = fmt.Sprintf("Deleted %s %s in namespace %s.", resourceType, name, namespace)
	case "scale_deployment":
		name, _ := approval.Payload["name"].(string)
		namespace, _ := approval.Payload["namespace"].(string)
		replicas := int32(numberField(approval.Payload["replicas"]))
		err = s.clusters.ScaleDeployment(ctx, *clusterID, namespace, name, replicas)
		result = fmt.Sprintf("Scaled deployment %s in namespace %s to %d replicas.", name, namespace, replicas)
	case "restart_deployment":
		name, _ := approval.Payload["name"].(string)
		namespace, _ := approval.Payload["namespace"].(string)
		err = s.clusters.RestartDeployment(ctx, *clusterID, namespace, name)
		result = fmt.Sprintf("Restarted deployment %s in namespace %s.", name, namespace)
	case "apply_yaml":
		manifest, _ := approval.Payload["manifest"].(string)
		applyResult, applyErr := s.clusters.ApplyYAML(ctx, *clusterID, manifest)
		err = applyErr
		if applyResult != nil {
			result = applyResult.Summary
		}
	default:
		err = fmt.Errorf("unsupported approved action: %s", approval.Type)
	}

	if err != nil {
		status = "failed"
		result = err.Error()
	}

	_ = s.repo.CompleteToolExecution(ctx, toolExecutionID, status, result, time.Since(startedAt).Milliseconds())
	return result, err
}

func (s *Service) runTool(ctx context.Context, session applicationchat.Session, intent intent) (string, error) {
	switch intent.Kind {
	case "list_namespaces":
		if session.Context.ClusterID == nil {
			return "", fmt.Errorf("select a cluster before asking for namespaces")
		}
		items, err := s.clusters.ListNamespaces(ctx, *session.Context.ClusterID)
		if err != nil {
			return "", err
		}
		return renderJSON(items)
	case "list_resources":
		if session.Context.ClusterID == nil {
			return "", fmt.Errorf("select a cluster before asking for kubernetes resources")
		}
		items, err := s.clusters.ListResources(ctx, *session.Context.ClusterID, applicationcluster.ResourceQuery{
			Type:      stringField(intent.Payload["type"]),
			Namespace: namespaceFromContext(session, intent),
		})
		if err != nil {
			return "", err
		}
		return renderJSON(items)
	case "list_events":
		if session.Context.ClusterID == nil {
			return "", fmt.Errorf("select a cluster before asking for kubernetes events")
		}
		items, err := s.clusters.ListEvents(ctx, *session.Context.ClusterID, namespaceFromContext(session, intent))
		if err != nil {
			return "", err
		}
		return renderJSON(items)
	case "list_models":
		items, err := s.models.List(ctx)
		if err != nil {
			return "", err
		}
		return renderJSON(items)
	case "list_skills":
		items, err := s.skills.List(ctx)
		if err != nil {
			return "", err
		}
		return renderJSON(items)
	case "list_mcp":
		items, err := s.mcp.List(ctx)
		if err != nil {
			return "", err
		}
		return renderJSON(items)
	default:
		return "", fmt.Errorf("unsupported intent: %s", intent.Kind)
	}
}

func (s *Service) resolveSessionModel(ctx context.Context, modelID *int64) (*applicationmodel.ResolvedRecord, error) {
	if modelID != nil {
		return s.models.Resolve(ctx, *modelID)
	}
	return s.models.ResolveDefault(ctx)
}

func (s *Service) failRun(ctx context.Context, run Run, session applicationchat.Session, requestID string, err error) {
	s.log.Error("agent run failed", zap.Int64("run_id", run.ID), zap.Error(err))

	userFacingMessage := s.userFacingRunError(ctx, session, err)
	var assistantMessageID *int64
	if userFacingMessage != "" {
		if assistantMessage, createErr := s.chat.CreateMessage(ctx, applicationchat.CreateMessageInput{
			SessionID: run.SessionID,
			Role:      "assistant",
			Content:   userFacingMessage,
		}); createErr == nil {
			assistantMessageID = &assistantMessage.ID
			s.publishEvent(ctx, run.ID, session.ID, "message_done", "orchestrator", StatusFailed, "assistant failure message stored", map[string]any{
				"assistantMessageId": assistantMessage.ID,
			}, requestID)
		}
	}

	_, _ = s.repo.CompleteRun(ctx, run.ID, StatusFailed, userFacingMessage, assistantMessageID, err.Error())
	s.publishEvent(ctx, run.ID, session.ID, "error", "orchestrator", StatusFailed, userFacingMessage, map[string]any{
		"rawError": err.Error(),
	}, requestID)
	s.publishEvent(ctx, run.ID, session.ID, "turn_end", "orchestrator", StatusFailed, "agent run failed", nil, requestID)
}

func (s *Service) userFacingRunError(ctx context.Context, session applicationchat.Session, err error) string {
	if errors.Is(err, applicationmodel.ErrNotFound) {
		return "No model is available for this session. Configure a default model or bind a model to the session first."
	}
	if strings.Contains(err.Error(), "Model not found") {
		modelName := "current model"
		if resolvedModel, resolveErr := s.resolveSessionModel(ctx, session.Context.ModelID); resolveErr == nil {
			modelName = fmt.Sprintf("%s (%s)", defaultString(resolvedModel.Name, resolvedModel.Model), resolvedModel.Model)
		}
		return fmt.Sprintf("%s is unavailable because the model service returned 'Model not found'. Check the model name, Base URL, and default model configuration.", modelName)
	}
	if strings.Contains(err.Error(), "returned reasoning only without a final answer") {
		return "The model returned reasoning content only and did not produce a final answer. Increase the output limit or switch to a model that is better for direct answers."
	}
	if strings.Contains(err.Error(), "returned an empty answer") {
		return "The model call succeeded but returned an empty answer. Please verify the model configuration and run a connectivity test."
	}
	if strings.Contains(err.Error(), "llm endpoint returned") || strings.Contains(err.Error(), "request llm endpoint") {
		return fmt.Sprintf("Model request failed: %s. Check the model name, endpoint, and network connectivity.", err.Error())
	}
	return fmt.Sprintf("Agent execution failed: %s", err.Error())
}
func sanitizeLLMOutput(content string) (string, bool) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", false
	}

	lower := strings.ToLower(trimmed)
	if !strings.Contains(lower, "<think>") {
		return trimmed, false
	}
	if !strings.Contains(lower, "</think>") {
		return "", true
	}

	trimmed = strings.TrimSpace(thinkBlockPattern.ReplaceAllString(trimmed, " "))
	return trimmed, true
}

func orchestratorSystemPrompt(userInput string) string {
	_ = userInput
	return "You are the KubeClaw orchestrator. Answer in the same language as the user, keep answers practical and concise, and do not expose hidden chain-of-thought or <think> content. If you are a reasoning model, return only the final answer."
}
func (s *Service) publishEvent(ctx context.Context, runID int64, sessionID int64, eventType string, role string, status string, message string, payload map[string]any, requestID ...string) {
	reqID := ""
	if len(requestID) > 0 {
		reqID = requestID[0]
	}

	event, err := s.repo.CreateEvent(ctx, Event{
		RunID:     runID,
		SessionID: sessionID,
		EventType: eventType,
		Role:      role,
		Status:    status,
		Message:   message,
		Payload:   payload,
		RequestID: reqID,
	})
	if err != nil {
		s.log.Error("store agent event failed", zap.Int64("run_id", runID), zap.Error(err))
		return
	}

	s.log.Info("agent event", zap.Int64("run_id", runID), zap.String("event_type", eventType), zap.String("status", status), zap.String("request_id", reqID))
	if s.streams != nil {
		s.streams.Publish(runID, *event)
	}
}

type intent struct {
	Kind             string
	Tool             string
	CapabilityType   string
	CapabilityID     int64
	CapabilityName   string
	CapabilityRole   string
	RequiresApproval bool
	ApprovalType     string
	Title            string
	Reason           string
	Payload          map[string]any
}

func (s *Service) analyzeIntent(ctx context.Context, session applicationchat.Session, content string) intent {
	raw := strings.TrimSpace(content)
	text := strings.ToLower(raw)

	capabilities := s.loadCapabilities(ctx)
	selected, ok := s.planIntentWithModel(ctx, session, raw, capabilities)
	if !ok {
		if focused, matched := s.resourceFocusedHeuristic(raw, text); matched {
			selected = focused
		} else {
			selected = s.heuristicIntent(raw, text)
		}
	}
	selected = s.resolveIntentCapability(raw, selected, capabilities)

	if risky, approvalType, reason, extraPayload := s.detectSensitiveReview(ctx, text); risky {
		selected.RequiresApproval = true
		selected.ApprovalType = approvalType
		selected.Title = strings.ReplaceAll(approvalType, "_", " ")
		selected.Reason = reason
		selected.Payload = mergePayloads(buildApprovalIntentPayload(selected), extraPayload)
		return selected
	}

	if risky, approvalType, reason, payload := s.detectRiskyIntent(raw, text); risky {
		selected.RequiresApproval = true
		selected.ApprovalType = approvalType
		selected.Title = plannedTitle(approvalType)
		selected.Reason = reason
		selected.Payload = mergePayloads(buildApprovalIntentPayload(selected), payload)
		return selected
	}

	return selected
}

func (s *Service) detectExplicitCapabilityExecutionIntent(ctx context.Context, raw string, text string) (intent, bool) {
	_ = ctx
	_ = raw
	_ = text
	return intent{}, false
}

func containsExecutionVerb(text string) bool {
	return strings.Contains(text, "use ") ||
		strings.Contains(text, "using ") ||
		strings.Contains(text, "call ") ||
		strings.Contains(text, "invoke ") ||
		strings.Contains(text, "trigger ") ||
		strings.Contains(text, "run ")
}
func (s *Service) explainCapabilityExecutionUnavailable(ctx context.Context, selected intent) (string, error) {
	_ = ctx
	_ = selected
	return "Capability execution is now routed through the unified agent executor.", nil
}
func (s *Service) planIntentWithModel(ctx context.Context, session applicationchat.Session, raw string, capabilities []capability) (intent, bool) {
	resolvedModel, err := s.resolveSessionModel(ctx, session.Context.ModelID)
	if err != nil {
		return intent{}, false
	}

	contextData := map[string]any{
		"clusterId": session.Context.ClusterID,
		"namespace": session.Context.Namespace,
		"modelId":   session.Context.ModelID,
	}
	contextJSON, _ := json.Marshal(contextData)
	capabilitiesJSON, _ := json.Marshal(s.plannerCapabilityCatalog(capabilities))

	result, err := s.llm.Chat(ctx, llm.ChatInput{
		Model: *resolvedModel,
		Messages: []llm.Message{
			{
				Role: "system",
				Content: "You are the KubeClaw planner. Return one JSON object only. " +
					"Schema: {\"kind\":\"list_namespaces|list_resources|list_events|list_models|list_skills|list_mcp|delete_resource|scale_deployment|restart_deployment|apply_yaml|llm\"," +
					"\"capabilityType\":\"builtin|skill|mcp|llm\",\"capabilityId\":0,\"capabilityName\":\"string\",\"tool\":\"string\",\"resourceType\":\"string\",\"resourceName\":\"string\",\"namespace\":\"string\",\"replicas\":0,\"reason\":\"string\"}. " +
					"Prefer an explicitly named skill or MCP when it matches. Prefer skills over MCP when both fit. Fall back to builtin tools when no external capability clearly matches. Use llm for normal conversation.",
			},
			{
				Role:    "user",
				Content: fmt.Sprintf("session_context=%s\ncapabilities=%s\nuser_message=%s", string(contextJSON), string(capabilitiesJSON), raw),
			},
		},
	})
	if err != nil {
		return intent{}, false
	}

	var plan struct {
		Kind           string `json:"kind"`
		CapabilityType string `json:"capabilityType"`
		CapabilityID   int64  `json:"capabilityId"`
		CapabilityName string `json:"capabilityName"`
		Tool           string `json:"tool"`
		ResourceType   string `json:"resourceType"`
		ResourceName   string `json:"resourceName"`
		Namespace      string `json:"namespace"`
		Replicas       int    `json:"replicas"`
		Reason         string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(extractJSONObject(result.Content)), &plan); err != nil {
		return intent{}, false
	}

	mapped := intent{
		Kind:           normalizePlannedKind(plan.Kind),
		Tool:           strings.TrimSpace(plan.Tool),
		CapabilityType: strings.ToLower(strings.TrimSpace(plan.CapabilityType)),
		CapabilityID:   plan.CapabilityID,
		CapabilityName: strings.TrimSpace(plan.CapabilityName),
		Reason:         strings.TrimSpace(plan.Reason),
		Payload: map[string]any{
			"type":      normalizeResourceType(plan.ResourceType),
			"name":      strings.TrimSpace(plan.ResourceName),
			"namespace": defaultString(strings.TrimSpace(plan.Namespace), session.Context.Namespace),
			"replicas":  plan.Replicas,
		},
	}
	if mapped.Tool == "" {
		mapped.Tool = defaultToolForKind(mapped.Kind)
	}

	switch mapped.Kind {
	case "list_resources":
		if stringField(mapped.Payload["type"]) == "" {
			mapped.Payload["type"] = inferResourceType(raw)
		}
	case "llm":
		return mapped, true
	}

	if mapped.Kind == "" {
		return intent{}, false
	}

	return mapped, true
}

func (s *Service) plannerCapabilityCatalog(capabilities []capability) []map[string]any {
	items := make([]map[string]any, 0, len(capabilities))
	for _, item := range capabilities {
		items = append(items, map[string]any{
			"type":             item.CapabilityType,
			"id":               item.ID,
			"name":             item.Name,
			"tool":             item.Tool,
			"summary":          item.Summary,
			"actions":          item.Actions,
			"resourceTypes":    item.ResourceTypes,
			"level":            item.Level,
			"audiences":        item.Audiences,
			"mutation":         item.Mutation,
			"requiresApproval": item.RequiresApproval,
			"requestMode":      item.RequestMode,
		})
	}
	return items
}

func (s *Service) resourceFocusedHeuristic(raw string, text string) (intent, bool) {
	switch {
	case strings.Contains(text, "pod"), strings.Contains(text, "pods"):
		return intent{Kind: "list_resources", Tool: "cluster.list_resources", CapabilityType: "builtin", Payload: map[string]any{
			"type":      "pods",
			"namespace": inferNamespace(raw),
		}}, true
	case strings.Contains(text, "deployment"), strings.Contains(text, "deployments"):
		return intent{Kind: "list_resources", Tool: "cluster.list_resources", CapabilityType: "builtin", Payload: map[string]any{
			"type":      "deployments",
			"namespace": inferNamespace(raw),
		}}, true
	case strings.Contains(text, "service"):
		return intent{Kind: "list_resources", Tool: "cluster.list_resources", CapabilityType: "builtin", Payload: map[string]any{
			"type":      "services",
			"namespace": inferNamespace(raw),
		}}, true
	case strings.Contains(text, "event"):
		return intent{Kind: "list_events", Tool: "cluster.list_events", CapabilityType: "builtin", Payload: map[string]any{
			"namespace": inferNamespace(raw),
		}}, true
	default:
		return intent{}, false
	}
}
func (s *Service) heuristicIntent(raw string, text string) intent {
	switch {
	case strings.Contains(text, "pod"), strings.Contains(text, "pods"):
		return intent{Kind: "list_resources", Tool: "cluster.list_resources", CapabilityType: "builtin", Payload: map[string]any{
			"type":      "pods",
			"namespace": inferNamespace(raw),
		}}
	case strings.Contains(text, "deployment"), strings.Contains(text, "deployments"):
		return intent{Kind: "list_resources", Tool: "cluster.list_resources", CapabilityType: "builtin", Payload: map[string]any{
			"type":      "deployments",
			"namespace": inferNamespace(raw),
		}}
	case strings.Contains(text, "service"):
		return intent{Kind: "list_resources", Tool: "cluster.list_resources", CapabilityType: "builtin", Payload: map[string]any{
			"type":      "services",
			"namespace": inferNamespace(raw),
		}}
	case strings.Contains(text, "event"):
		return intent{Kind: "list_events", Tool: "cluster.list_events", CapabilityType: "builtin", Payload: map[string]any{
			"namespace": inferNamespace(raw),
		}}
	case strings.Contains(text, "namespace"):
		return intent{Kind: "list_namespaces", Tool: "cluster.list_namespaces", CapabilityType: "builtin", Payload: map[string]any{}}
	case strings.Contains(text, "model"):
		return intent{Kind: "list_models", Tool: "model.list", CapabilityType: "builtin", Payload: map[string]any{}}
	case strings.Contains(text, "skill"):
		return intent{Kind: "list_skills", Tool: "skill.list", CapabilityType: "builtin", Payload: map[string]any{}}
	case strings.Contains(text, "mcp"):
		return intent{Kind: "list_mcp", Tool: "mcp.list", CapabilityType: "builtin", Payload: map[string]any{}}
	default:
		return intent{Kind: "llm", Tool: "llm.chat", CapabilityType: "llm", Payload: map[string]any{}}
	}
}
func (s *Service) detectSensitiveReview(ctx context.Context, text string) (bool, string, string, map[string]any) {
	words, err := s.security.ListSensitiveWords(ctx)
	if err == nil {
		for _, item := range words {
			if item.IsEnabled && strings.Contains(text, strings.ToLower(item.Word)) {
				payload := map[string]any{"matchedWord": item.Word}
				return true, "manual_review", fmt.Sprintf("Matched sensitive word rule %s.", item.Word), payload
			}
		}
	}
	return false, "", "", nil
}

func (s *Service) detectRiskyIntent(raw string, text string) (bool, string, string, map[string]any) {
	switch {
	case strings.Contains(text, "delete "):
		return true, "delete_resource", "Deleting Kubernetes resources requires human approval.", parseResourceCommand(raw)
	case strings.Contains(text, "scale "), strings.Contains(text, "replica"):
		payload := parseScaleCommand(raw)
		return true, "scale_deployment", "Scaling a workload changes runtime state and requires approval.", payload
	case strings.Contains(text, "restart "):
		payload := parseRestartCommand(raw)
		return true, "restart_deployment", "Restarting a workload affects live traffic and requires approval.", payload
	case strings.Contains(text, "apply ") || strings.Contains(text, "kubectl apply") || strings.Contains(text, "apply yaml"):
		return true, "apply_yaml", "Applying manifests can mutate cluster state and requires approval.", map[string]any{
			"manifest": raw,
		}
	default:
		return false, "", "", nil
	}
}
func (s *Service) intentRole(selected intent) string {
	if strings.TrimSpace(selected.CapabilityRole) != "" {
		return selected.CapabilityRole
	}
	switch selected.CapabilityType {
	case "skill":
		return "skill_worker"
	case "mcp":
		return "mcp_worker"
	case "builtin":
		switch selected.Kind {
		case "list_namespaces", "list_resources", "list_events", "delete_resource", "scale_deployment", "restart_deployment", "apply_yaml":
			return "k8s_expert"
		case "list_skills", "list_mcp", "list_models":
			return "skill_mcp_expert"
		default:
			return "orchestrator"
		}
	default:
		return "orchestrator"
	}
}

func namespaceFromContext(session applicationchat.Session, intent intent) string {
	if namespace := stringField(intent.Payload["namespace"]); namespace != "" {
		return namespace
	}
	return session.Context.Namespace
}

func renderJSON(value any) (string, error) {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func stringField(value any) string {
	text, _ := value.(string)
	return text
}

func numberField(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func parseResourceCommand(text string) map[string]any {
	payload := map[string]any{
		"type":      inferResourceType(text),
		"name":      "unknown",
		"namespace": defaultString(inferNamespace(text), "default"),
	}
	parts := strings.Fields(text)
	for idx, item := range parts {
		switch item {
		case "pod", "pods":
			payload["type"] = "pods"
			if idx+1 < len(parts) {
				payload["name"] = parts[idx+1]
			}
		case "deployment", "deployments":
			payload["type"] = "deployments"
			if idx+1 < len(parts) {
				payload["name"] = parts[idx+1]
			}
		case "service", "services":
			payload["type"] = "services"
			if idx+1 < len(parts) {
				payload["name"] = parts[idx+1]
			}
		case "namespace":
			if idx+1 < len(parts) {
				payload["namespace"] = parts[idx+1]
			}
		}
	}
	return payload
}

func parseScaleCommand(text string) map[string]any {
	payload := map[string]any{
		"name":      "unknown",
		"namespace": defaultString(inferNamespace(text), "default"),
		"replicas":  1,
	}
	parts := strings.Fields(text)
	for idx, item := range parts {
		switch item {
		case "deployment", "deployments", "scale":
			if idx+1 < len(parts) {
				payload["name"] = parts[idx+1]
			}
		case "namespace":
			if idx+1 < len(parts) {
				payload["namespace"] = parts[idx+1]
			}
		case "to", "replicas":
			if idx+1 < len(parts) {
				var replicas int
				fmt.Sscanf(parts[idx+1], "%d", &replicas)
				if replicas > 0 {
					payload["replicas"] = replicas
				}
			}
		}
	}
	return payload
}

func parseRestartCommand(text string) map[string]any {
	payload := map[string]any{
		"name":      "unknown",
		"namespace": defaultString(inferNamespace(text), "default"),
	}
	parts := strings.Fields(text)
	for idx, item := range parts {
		switch item {
		case "deployment", "deployments", "restart":
			if idx+1 < len(parts) {
				payload["name"] = parts[idx+1]
			}
		case "namespace":
			if idx+1 < len(parts) {
				payload["namespace"] = parts[idx+1]
			}
		}
	}
	return payload
}

func extractJSONObject(content string) string {
	trimmed := strings.TrimSpace(content)
	trimmed = strings.TrimPrefix(trimmed, "`json")
	trimmed = strings.TrimPrefix(trimmed, "`")
	trimmed = strings.TrimSuffix(trimmed, "`")
	trimmed = strings.TrimSpace(trimmed)

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start >= 0 && end > start {
		return trimmed[start : end+1]
	}
	return trimmed
}

func normalizePlannedKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "list_namespaces", "namespace", "namespaces":
		return "list_namespaces"
	case "list_resources", "list_resource", "resource", "resources":
		return "list_resources"
	case "list_events", "events", "event":
		return "list_events"
	case "list_models", "models", "model":
		return "list_models"
	case "list_skills", "skills", "skill":
		return "list_skills"
	case "list_mcp", "mcp":
		return "list_mcp"
	case "delete_resource", "delete":
		return "delete_resource"
	case "scale_deployment", "scale":
		return "scale_deployment"
	case "restart_deployment", "restart":
		return "restart_deployment"
	case "apply_yaml", "apply":
		return "apply_yaml"
	default:
		return "llm"
	}
}

func normalizeResourceType(resourceType string) string {
	switch strings.ToLower(strings.TrimSpace(resourceType)) {
	case "pod", "pods":
		return "pods"
	case "deployment", "deployments":
		return "deployments"
	case "service", "services":
		return "services"
	case "configmap", "configmaps":
		return "configmaps"
	case "secret", "secrets":
		return "secrets"
	default:
		return ""
	}
}

func inferResourceType(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "pod"):
		return "pods"
	case strings.Contains(lower, "service"):
		return "services"
	case strings.Contains(lower, "configmap"):
		return "configmaps"
	case strings.Contains(lower, "secret"):
		return "secrets"
	default:
		return "deployments"
	}
}

func inferNamespace(text string) string {
	if value := captureMatch("namespace\\s+([a-zA-Z0-9\\-_.]+)", text); value != "" {
		return value
	}
	return ""
}

func captureMatch(pattern string, text string) string {
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func defaultToolForKind(kind string) string {
	switch kind {
	case "list_namespaces":
		return "cluster.list_namespaces"
	case "list_resources":
		return "cluster.list_resources"
	case "list_events":
		return "cluster.list_events"
	case "list_models":
		return "model.list"
	case "list_skills":
		return "skill.list"
	case "list_mcp":
		return "mcp.list"
	case "delete_resource":
		return "cluster.delete_resource"
	case "scale_deployment":
		return "cluster.scale_deployment"
	case "restart_deployment":
		return "cluster.restart_deployment"
	case "apply_yaml":
		return "cluster.apply_yaml"
	default:
		return "llm.chat"
	}
}

func plannedTitle(kind string) string {
	switch kind {
	case "delete_resource":
		return "delete resource"
	case "scale_deployment":
		return "scale deployment"
	case "restart_deployment":
		return "restart deployment"
	case "apply_yaml":
		return "apply yaml"
	default:
		return strings.ReplaceAll(kind, "_", " ")
	}
}

func plannedReason(kind string) string {
	switch kind {
	case "delete_resource":
		return "Deleting Kubernetes resources requires human approval."
	case "scale_deployment":
		return "Scaling a workload changes runtime state and requires approval."
	case "restart_deployment":
		return "Restarting a workload affects live traffic and requires approval."
	case "apply_yaml":
		return "Applying manifests can mutate cluster state and requires approval."
	default:
		return ""
	}
}

func buildActionSessionTitle(input ClusterActionRequestInput) string {
	return fmt.Sprintf("Cluster action %s / cluster %d", input.Action, input.ClusterID)
}

func buildClusterActionMessage(input ClusterActionRequestInput) string {
	namespace := defaultString(input.Namespace, "default")
	switch strings.ToLower(strings.TrimSpace(input.Action)) {
	case "delete_resource":
		return fmt.Sprintf("Delete %s %s in namespace %s.", input.ResourceType, input.ResourceName, namespace)
	case "scale_deployment":
		return fmt.Sprintf("Scale deployment %s in namespace %s to %d replicas.", input.ResourceName, namespace, input.Replicas)
	case "restart_deployment":
		return fmt.Sprintf("Restart deployment %s in namespace %s.", input.ResourceName, namespace)
	case "apply_yaml":
		return input.Manifest
	default:
		return defaultString(input.Action, "cluster action request")
	}
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func defaultInt64(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
