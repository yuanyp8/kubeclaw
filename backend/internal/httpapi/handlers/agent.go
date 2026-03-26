package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	applicationagent "kubeclaw/backend/internal/application/agent"
	applicationchat "kubeclaw/backend/internal/application/chat"
	"kubeclaw/backend/internal/httpapi/middleware"
	"kubeclaw/backend/internal/infrastructure/agentruntime"

	"github.com/gin-gonic/gin"
)

type AgentHandler struct {
	service *applicationagent.Service
	streams *agentruntime.Hub[applicationagent.Event]
}

func NewAgentHandler(service *applicationagent.Service, streams *agentruntime.Hub[applicationagent.Event]) *AgentHandler {
	return &AgentHandler{service: service, streams: streams}
}

func (h *AgentHandler) CreateSession(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	var req struct {
		Title     string `json:"title"`
		ModelID   *int64 `json:"modelId"`
		ClusterID *int64 `json:"clusterId"`
		Namespace string `json:"namespace"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid session payload")
		return
	}

	session, err := h.service.CreateSession(c.Request.Context(), applicationagent.CreateSessionInput{
		TenantID: currentUser.TenantID,
		UserID:   currentUser.ID,
		Title:    req.Title,
		Context: applicationchat.SessionContext{
			ModelID:   req.ModelID,
			ClusterID: req.ClusterID,
			Namespace: req.Namespace,
		},
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "create agent session failed")
		return
	}

	writeSuccess(c, http.StatusCreated, "agent session created", session)
}

func (h *AgentHandler) ListSessions(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	items, err := h.service.ListSessions(c.Request.Context(), currentUser.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load agent sessions failed")
		return
	}
	writeSuccess(c, http.StatusOK, "agent sessions loaded", items)
}

func (h *AgentHandler) GetSession(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	sessionID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	item, err := h.service.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		if errors.Is(err, applicationchat.ErrSessionNotFound) {
			writeError(c, http.StatusNotFound, "SESSION_NOT_FOUND", "agent session was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load agent session failed")
		return
	}
	if !ensureUserOwnedResource(c, currentUser, item.UserID) {
		return
	}
	writeSuccess(c, http.StatusOK, "agent session loaded", item)
}

func (h *AgentHandler) ListMessages(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	sessionID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	session, err := h.service.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		if errors.Is(err, applicationchat.ErrSessionNotFound) {
			writeError(c, http.StatusNotFound, "SESSION_NOT_FOUND", "agent session was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load agent session failed")
		return
	}
	if !ensureUserOwnedResource(c, currentUser, session.UserID) {
		return
	}

	items, err := h.service.ListMessages(c.Request.Context(), sessionID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load agent messages failed")
		return
	}
	writeSuccess(c, http.StatusOK, "agent messages loaded", items)
}

func (h *AgentHandler) SendMessage(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	sessionID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	session, err := h.service.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		if errors.Is(err, applicationchat.ErrSessionNotFound) {
			writeError(c, http.StatusNotFound, "SESSION_NOT_FOUND", "agent session was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load agent session failed")
		return
	}
	if !ensureUserOwnedResource(c, currentUser, session.UserID) {
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Content == "" {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "message content is required")
		return
	}

	result, err := h.service.SendMessage(c.Request.Context(), applicationagent.SendMessageInput{
		SessionID: sessionID,
		UserID:    currentUser.ID,
		Content:   req.Content,
		RequestID: requestIDFromContext(c),
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "send agent message failed")
		return
	}

	writeSuccess(c, http.StatusAccepted, "agent run started", result)
}

func (h *AgentHandler) DeleteSession(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	sessionID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	session, err := h.service.GetSession(c.Request.Context(), sessionID)
	if err != nil {
		if errors.Is(err, applicationchat.ErrSessionNotFound) {
			writeError(c, http.StatusNotFound, "SESSION_NOT_FOUND", "agent session was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load agent session failed")
		return
	}
	if !ensureUserOwnedResource(c, currentUser, session.UserID) {
		return
	}

	if err := h.service.DeleteSession(c.Request.Context(), sessionID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "delete agent session failed")
		return
	}

	writeSuccess(c, http.StatusOK, "agent session deleted", gin.H{"id": sessionID})
}

func (h *AgentHandler) ListRunEvents(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	runID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	run, err := h.service.GetRun(c.Request.Context(), runID)
	if err != nil {
		if errors.Is(err, applicationagent.ErrRunNotFound) {
			writeError(c, http.StatusNotFound, "RUN_NOT_FOUND", "agent run was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load agent run failed")
		return
	}
	if !ensureUserOwnedResource(c, currentUser, run.UserID) {
		return
	}

	items, err := h.service.ListRunEvents(c.Request.Context(), runID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load run events failed")
		return
	}

	writeSuccess(c, http.StatusOK, "run events loaded", items)
}

func (h *AgentHandler) StreamRun(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	runID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	run, err := h.service.GetRun(c.Request.Context(), runID)
	if err != nil {
		if errors.Is(err, applicationagent.ErrRunNotFound) {
			writeError(c, http.StatusNotFound, "RUN_NOT_FOUND", "agent run was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load run stream failed")
		return
	}
	if !ensureUserOwnedResource(c, currentUser, run.UserID) {
		return
	}

	existing, err := h.service.ListRunEvents(c.Request.Context(), runID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load run stream failed")
		return
	}

	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")

	for _, item := range existing {
		if err := writeSSE(c, "event", item); err != nil {
			return
		}
	}

	subscribe, cancel := h.streams.Subscribe(runID)
	defer cancel()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case item, ok := <-subscribe:
			if !ok {
				return
			}
			if err := writeSSE(c, "event", item); err != nil {
				return
			}
		}
	}
}

func (h *AgentHandler) Approve(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	approvalID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	approval, err := h.service.GetApproval(c.Request.Context(), approvalID)
	if err != nil {
		if errors.Is(err, applicationagent.ErrApprovalNotFound) {
			writeError(c, http.StatusNotFound, "APPROVAL_NOT_FOUND", "approval request was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load approval request failed")
		return
	}
	if !ensureUserOwnedResource(c, currentUser, approval.UserID) {
		return
	}

	item, err := h.service.Approve(c.Request.Context(), approvalID, currentUser.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "approve request failed")
		return
	}

	middleware.AppendAuditEntry(c, middleware.AuditEntry{
		Action: "agent.approval.approve",
		Target: fmt.Sprintf("approval:%d", approvalID),
		Details: map[string]any{
			"approvalId": approvalID,
			"runId":      item.RunID,
			"sessionId":  item.SessionID,
			"type":       item.Type,
			"title":      item.Title,
		},
	})
	writeSuccess(c, http.StatusOK, "approval accepted", item)
}

func (h *AgentHandler) Reject(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	approvalID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	approval, err := h.service.GetApproval(c.Request.Context(), approvalID)
	if err != nil {
		if errors.Is(err, applicationagent.ErrApprovalNotFound) {
			writeError(c, http.StatusNotFound, "APPROVAL_NOT_FOUND", "approval request was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load approval request failed")
		return
	}
	if !ensureUserOwnedResource(c, currentUser, approval.UserID) {
		return
	}

	item, err := h.service.Reject(c.Request.Context(), approvalID, currentUser.ID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "reject request failed")
		return
	}

	middleware.AppendAuditEntry(c, middleware.AuditEntry{
		Action: "agent.approval.reject",
		Target: fmt.Sprintf("approval:%d", approvalID),
		Details: map[string]any{
			"approvalId": approvalID,
			"runId":      item.RunID,
			"sessionId":  item.SessionID,
			"type":       item.Type,
			"title":      item.Title,
		},
	})
	writeSuccess(c, http.StatusOK, "approval rejected", item)
}

func writeSSE(c *gin.Context, event string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(c.Writer, "event: %s\n", event); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", string(body)); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}
