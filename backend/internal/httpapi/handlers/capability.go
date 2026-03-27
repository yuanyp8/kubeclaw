package handlers

import (
	"context"
	"net/http"
	"strings"

	applicationagent "kubeclaw/backend/internal/application/agent"
	applicationcapability "kubeclaw/backend/internal/application/capability"

	"github.com/gin-gonic/gin"
)

type capabilityRequester interface {
	RequestClusterAction(ctx context.Context, input applicationagent.ClusterActionRequestInput) (*applicationagent.SendMessageResult, error)
}

type CapabilityHandler struct {
	service   *applicationcapability.Service
	requester capabilityRequester
}

func NewCapabilityHandler(service *applicationcapability.Service, requester capabilityRequester) *CapabilityHandler {
	return &CapabilityHandler{service: service, requester: requester}
}

func (h *CapabilityHandler) List(c *gin.Context) {
	if h == nil || h.service == nil {
		writeError(c, http.StatusServiceUnavailable, "CAPABILITY_SERVICE_UNAVAILABLE", "capability service is unavailable")
		return
	}

	audience := strings.ToLower(strings.TrimSpace(c.Query("audience")))
	switch audience {
	case "":
		writeSuccess(c, http.StatusOK, "capabilities loaded", h.service.List(c.Request.Context()))
	case string(applicationcapability.AudienceAgent):
		writeSuccess(c, http.StatusOK, "capabilities loaded", h.service.ListForAudience(c.Request.Context(), applicationcapability.AudienceAgent))
	case string(applicationcapability.AudienceHTTP):
		writeSuccess(c, http.StatusOK, "capabilities loaded", h.service.ListForAudience(c.Request.Context(), applicationcapability.AudienceHTTP))
	case string(applicationcapability.AudienceMCP):
		writeSuccess(c, http.StatusOK, "capabilities loaded", h.service.ListForAudience(c.Request.Context(), applicationcapability.AudienceMCP))
	default:
		writeError(c, http.StatusBadRequest, "INVALID_CAPABILITY_AUDIENCE", "audience must be one of: agent, http, mcp")
	}
}

func (h *CapabilityHandler) Get(c *gin.Context) {
	if h == nil || h.service == nil {
		writeError(c, http.StatusServiceUnavailable, "CAPABILITY_SERVICE_UNAVAILABLE", "capability service is unavailable")
		return
	}

	audience := applicationcapability.Audience(strings.ToLower(strings.TrimSpace(c.Query("audience"))))
	if audience == "" {
		item, err := h.service.ResolveReference(c.Request.Context(), c.Param("ref"), "")
		if err != nil {
			writeError(c, http.StatusNotFound, "CAPABILITY_NOT_FOUND", err.Error())
			return
		}
		writeSuccess(c, http.StatusOK, "capability loaded", item)
		return
	}
	if audience != applicationcapability.AudienceAgent && audience != applicationcapability.AudienceHTTP && audience != applicationcapability.AudienceMCP {
		writeError(c, http.StatusBadRequest, "INVALID_CAPABILITY_AUDIENCE", "audience must be one of: agent, http, mcp")
		return
	}
	item, err := h.service.ResolveReference(c.Request.Context(), c.Param("ref"), audience)
	if err != nil {
		writeError(c, http.StatusNotFound, "CAPABILITY_NOT_FOUND", err.Error())
		return
	}
	writeSuccess(c, http.StatusOK, "capability loaded", item)
}

type capabilityInvokeRequest struct {
	Action    string         `json:"action"`
	UserInput string         `json:"userInput"`
	ClusterID *int64         `json:"clusterId"`
	Namespace string         `json:"namespace"`
	ModelID   *int64         `json:"modelId"`
	Payload   map[string]any `json:"payload"`
}

func (h *CapabilityHandler) Invoke(c *gin.Context) {
	if h == nil || h.service == nil {
		writeError(c, http.StatusServiceUnavailable, "CAPABILITY_SERVICE_UNAVAILABLE", "capability service is unavailable")
		return
	}

	var req capabilityInvokeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid capability invoke payload")
		return
	}

	result, err := h.service.Invoke(c.Request.Context(), applicationcapability.InvokeInput{
		Audience:  applicationcapability.AudienceHTTP,
		Reference: c.Param("ref"),
		Selection: applicationcapability.Selection{
			Action:  req.Action,
			Payload: req.Payload,
		},
		Context: applicationcapability.InvokeContext{
			ClusterID: req.ClusterID,
			Namespace: req.Namespace,
			ModelID:   req.ModelID,
		},
		UserInput: req.UserInput,
	})
	if err != nil {
		if _, anyErr := h.service.ResolveReference(c.Request.Context(), c.Param("ref"), ""); anyErr == nil {
			writeError(c, http.StatusBadRequest, "CAPABILITY_DIRECT_INVOKE_DISABLED", "capability exists but is not available for direct HTTP invoke")
			return
		}
		writeError(c, http.StatusBadRequest, "CAPABILITY_INVOKE_FAILED", err.Error())
		return
	}

	writeSuccess(c, http.StatusOK, "capability invoked", gin.H{
		"reference": c.Param("ref"),
		"result":    result,
	})
}

type capabilityRequestRequest struct {
	Action    string         `json:"action"`
	ClusterID *int64         `json:"clusterId"`
	Namespace string         `json:"namespace"`
	Payload   map[string]any `json:"payload"`
}

func (h *CapabilityHandler) Request(c *gin.Context) {
	if h == nil || h.service == nil || h.requester == nil {
		writeError(c, http.StatusServiceUnavailable, "CAPABILITY_REQUEST_UNAVAILABLE", "capability request pipeline is unavailable")
		return
	}

	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	item, err := h.service.ResolveReference(c.Request.Context(), c.Param("ref"), "")
	if err != nil {
		writeError(c, http.StatusNotFound, "CAPABILITY_NOT_FOUND", err.Error())
		return
	}
	if !item.RequiresApproval || strings.TrimSpace(item.RequestMode) != "agent_approval" {
		writeError(c, http.StatusBadRequest, "CAPABILITY_NOT_REQUESTABLE", "capability does not support agent-mediated approval requests")
		return
	}

	var req capabilityRequestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid capability request payload")
		return
	}

	action := strings.TrimSpace(req.Action)
	if action == "" {
		if len(item.Actions) == 1 {
			action = item.Actions[0]
		} else {
			action = item.TargetAction
		}
	}
	if action == "" {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "capability action is required")
		return
	}
	if req.ClusterID == nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "clusterId is required")
		return
	}

	input, err := buildCapabilityClusterActionRequest(*item, action, req, currentUser.ID, currentUser.TenantID, requestIDFromContext(c))
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", err.Error())
		return
	}

	result, err := h.requester.RequestClusterAction(c.Request.Context(), input)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "capability request failed")
		return
	}

	writeSuccess(c, http.StatusAccepted, "capability request created", gin.H{
		"reference": c.Param("ref"),
		"mode":      item.RequestMode,
		"request":   result,
	})
}

func buildCapabilityClusterActionRequest(item applicationcapability.Descriptor, action string, req capabilityRequestRequest, userID int64, tenantID *int64, requestID string) (applicationagent.ClusterActionRequestInput, error) {
	payload := req.Payload
	namespace := strings.TrimSpace(req.Namespace)
	if namespace == "" {
		namespace = strings.TrimSpace(stringField(payload["namespace"]))
	}

	input := applicationagent.ClusterActionRequestInput{
		ClusterID: *req.ClusterID,
		TenantID:  tenantID,
		UserID:    userID,
		Action:    action,
		Namespace: namespace,
		RequestID: requestID,
	}

	switch action {
	case "delete_resource":
		input.ResourceType = strings.TrimSpace(stringField(payload["type"]))
		input.ResourceName = strings.TrimSpace(stringField(payload["name"]))
		if input.ResourceType == "" || input.ResourceName == "" {
			return applicationagent.ClusterActionRequestInput{}, httpError("resource type and name are required")
		}
	case "scale_deployment":
		input.ResourceName = strings.TrimSpace(stringField(payload["name"]))
		input.Replicas = int32(numberField(payload["replicas"]))
		if input.ResourceName == "" || input.Replicas <= 0 {
			return applicationagent.ClusterActionRequestInput{}, httpError("deployment name and replicas are required")
		}
	case "restart_deployment":
		input.ResourceName = strings.TrimSpace(stringField(payload["name"]))
		if input.ResourceName == "" {
			return applicationagent.ClusterActionRequestInput{}, httpError("deployment name is required")
		}
	case "apply_yaml":
		input.Manifest = strings.TrimSpace(stringField(payload["manifest"]))
		if input.Manifest == "" {
			return applicationagent.ClusterActionRequestInput{}, httpError("manifest is required")
		}
	default:
		return applicationagent.ClusterActionRequestInput{}, httpError("unsupported requestable capability action: " + action)
	}

	_ = item
	return input, nil
}

type handlerError string

func (e handlerError) Error() string { return string(e) }

func httpError(message string) error { return handlerError(message) }

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
