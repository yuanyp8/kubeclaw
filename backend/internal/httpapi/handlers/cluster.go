package handlers

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	applicationagent "kubeclaw/backend/internal/application/agent"
	applicationcluster "kubeclaw/backend/internal/application/cluster"
	"kubeclaw/backend/internal/httpapi/middleware"

	"github.com/gin-gonic/gin"
)

type ClusterHandler struct {
	service         *applicationcluster.Service
	actionRequester *applicationagent.Service
}

func NewClusterHandler(service *applicationcluster.Service, actionRequester *applicationagent.Service) *ClusterHandler {
	return &ClusterHandler{
		service:         service,
		actionRequester: actionRequester,
	}
}

func (h *ClusterHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load clusters failed")
		return
	}
	writeSuccess(c, http.StatusOK, "clusters loaded", items)
}

func (h *ClusterHandler) Get(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, applicationcluster.ErrNotFound) {
			writeError(c, http.StatusNotFound, "CLUSTER_NOT_FOUND", "cluster was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load cluster failed")
		return
	}
	writeSuccess(c, http.StatusOK, "cluster loaded", item)
}

func (h *ClusterHandler) Create(c *gin.Context) {
	var req applicationcluster.CreateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid cluster payload")
		return
	}
	req.Environment = defaultString(req.Environment, "prod")
	req.Status = defaultString(req.Status, "active")

	item, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "create cluster failed")
		return
	}
	writeSuccess(c, http.StatusCreated, "cluster created", item)
}

func (h *ClusterHandler) Update(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req applicationcluster.UpdateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid cluster payload")
		return
	}

	item, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, applicationcluster.ErrNotFound) {
			writeError(c, http.StatusNotFound, "CLUSTER_NOT_FOUND", "cluster was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "update cluster failed")
		return
	}
	writeSuccess(c, http.StatusOK, "cluster updated", item)
}

func (h *ClusterHandler) Delete(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "delete cluster failed")
		return
	}
	writeSuccess(c, http.StatusOK, "cluster deleted", gin.H{"id": id})
}

func (h *ClusterHandler) Share(c *gin.Context) {
	clusterID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	var req applicationcluster.ShareInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid share payload")
		return
	}
	if req.UserID == nil && req.TeamID == nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "userId or teamId is required")
		return
	}
	req.Role = defaultString(req.Role, "viewer")

	item, err := h.service.Share(c.Request.Context(), clusterID, req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "share cluster failed")
		return
	}
	writeSuccess(c, http.StatusOK, "cluster shared", item)
}

func (h *ClusterHandler) ListPermissions(c *gin.Context) {
	clusterID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	items, err := h.service.ListPermissions(c.Request.Context(), clusterID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load cluster permissions failed")
		return
	}
	writeSuccess(c, http.StatusOK, "cluster permissions loaded", items)
}

func (h *ClusterHandler) Validate(c *gin.Context) {
	clusterID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	result, err := h.service.Validate(c.Request.Context(), clusterID)
	if err != nil {
		if errors.Is(err, applicationcluster.ErrNotFound) {
			writeError(c, http.StatusNotFound, "CLUSTER_NOT_FOUND", "cluster was not found")
			return
		}
		writeError(c, http.StatusBadGateway, "CLUSTER_VALIDATE_FAILED", err.Error())
		return
	}

	writeSuccess(c, http.StatusOK, "cluster connectivity verified", result)
}

func (h *ClusterHandler) Overview(c *gin.Context) {
	clusterID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	result, err := h.service.GetOverview(c.Request.Context(), clusterID, c.Query("namespace"))
	if err != nil {
		if errors.Is(err, applicationcluster.ErrNotFound) {
			writeError(c, http.StatusNotFound, "CLUSTER_NOT_FOUND", "cluster was not found")
			return
		}
		writeError(c, http.StatusBadGateway, "K8S_QUERY_FAILED", err.Error())
		return
	}

	writeSuccess(c, http.StatusOK, "cluster overview loaded", result)
}

func (h *ClusterHandler) ListNamespaces(c *gin.Context) {
	clusterID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	items, err := h.service.ListNamespaces(c.Request.Context(), clusterID)
	if err != nil {
		if errors.Is(err, applicationcluster.ErrNotFound) {
			writeError(c, http.StatusNotFound, "CLUSTER_NOT_FOUND", "cluster was not found")
			return
		}
		writeError(c, http.StatusBadGateway, "K8S_QUERY_FAILED", err.Error())
		return
	}

	writeSuccess(c, http.StatusOK, "namespaces loaded", items)
}

func (h *ClusterHandler) ListResources(c *gin.Context) {
	clusterID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	items, err := h.service.ListResources(c.Request.Context(), clusterID, applicationcluster.ResourceQuery{
		Type:      c.Query("type"),
		Namespace: c.Query("namespace"),
	})
	if err != nil {
		switch {
		case errors.Is(err, applicationcluster.ErrNotFound):
			writeError(c, http.StatusNotFound, "CLUSTER_NOT_FOUND", "cluster was not found")
		case errors.Is(err, applicationcluster.ErrInvalidResourceType):
			writeError(c, http.StatusBadRequest, "INVALID_RESOURCE_TYPE", "resource type is not supported")
		default:
			writeError(c, http.StatusBadGateway, "K8S_QUERY_FAILED", err.Error())
		}
		return
	}

	writeSuccess(c, http.StatusOK, "resources loaded", items)
}

func (h *ClusterHandler) GetResource(c *gin.Context) {
	clusterID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	item, err := h.service.GetResource(c.Request.Context(), clusterID, applicationcluster.ResourceQuery{
		Type:      c.Param("type"),
		Namespace: c.Query("namespace"),
	}, c.Param("name"))
	if err != nil {
		switch {
		case errors.Is(err, applicationcluster.ErrNotFound):
			writeError(c, http.StatusNotFound, "CLUSTER_NOT_FOUND", "cluster was not found")
		case errors.Is(err, applicationcluster.ErrInvalidResourceType):
			writeError(c, http.StatusBadRequest, "INVALID_RESOURCE_TYPE", "resource type is not supported")
		default:
			writeError(c, http.StatusBadGateway, "K8S_QUERY_FAILED", err.Error())
		}
		return
	}

	writeSuccess(c, http.StatusOK, "resource loaded", item)
}

func (h *ClusterHandler) ListEvents(c *gin.Context) {
	clusterID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	items, err := h.service.ListEvents(c.Request.Context(), clusterID, c.Query("namespace"))
	if err != nil {
		if errors.Is(err, applicationcluster.ErrNotFound) {
			writeError(c, http.StatusNotFound, "CLUSTER_NOT_FOUND", "cluster was not found")
			return
		}
		writeError(c, http.StatusBadGateway, "K8S_QUERY_FAILED", err.Error())
		return
	}

	writeSuccess(c, http.StatusOK, "events loaded", items)
}

func (h *ClusterHandler) StreamPodLogs(c *gin.Context) {
	clusterID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	tailLines := int64(200)
	if raw := c.DefaultQuery("tailLines", "200"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			tailLines = parsed
		}
	}

	sinceSeconds := int64(0)
	if raw := c.Query("sinceSeconds"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed > 0 {
			sinceSeconds = parsed
		}
	}

	stream, err := h.service.StreamPodLogs(c.Request.Context(), clusterID, applicationcluster.PodLogQuery{
		Namespace:    c.Query("namespace"),
		PodName:      c.Param("name"),
		Container:    c.Query("container"),
		Follow:       c.DefaultQuery("follow", "true") != "false",
		TailLines:    tailLines,
		SinceSeconds: sinceSeconds,
	})
	if err != nil {
		if errors.Is(err, applicationcluster.ErrNotFound) {
			writeError(c, http.StatusNotFound, "CLUSTER_NOT_FOUND", "cluster was not found")
			return
		}
		writeError(c, http.StatusBadGateway, "K8S_LOG_STREAM_FAILED", err.Error())
		return
	}
	defer stream.Close()

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)

	buffer := make([]byte, 4096)
	for {
		n, readErr := stream.Read(buffer)
		if n > 0 {
			if _, writeErr := c.Writer.Write(buffer[:n]); writeErr != nil {
				return
			}
			c.Writer.Flush()
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				return
			}
			return
		}
	}
}

func (h *ClusterHandler) RequestDeleteResource(c *gin.Context) {
	clusterID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "current user is missing")
		return
	}

	var req struct {
		Type      string `json:"type"`
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Type == "" || req.Name == "" {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "resource type and name are required")
		return
	}

	result, err := h.actionRequester.RequestClusterAction(c.Request.Context(), applicationagent.ClusterActionRequestInput{
		ClusterID:    clusterID,
		TenantID:     currentUser.TenantID,
		UserID:       currentUser.ID,
		Action:       "delete_resource",
		ResourceType: req.Type,
		ResourceName: req.Name,
		Namespace:    req.Namespace,
		RequestID:    requestIDFromContext(c),
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "create delete approval failed")
		return
	}

	writeSuccess(c, http.StatusAccepted, "delete approval requested", result)
}

func (h *ClusterHandler) RequestScaleDeployment(c *gin.Context) {
	clusterID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "current user is missing")
		return
	}

	var req struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		Replicas  int32  `json:"replicas"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" || req.Replicas <= 0 {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "deployment name and replicas are required")
		return
	}

	result, err := h.actionRequester.RequestClusterAction(c.Request.Context(), applicationagent.ClusterActionRequestInput{
		ClusterID:    clusterID,
		TenantID:     currentUser.TenantID,
		UserID:       currentUser.ID,
		Action:       "scale_deployment",
		ResourceName: req.Name,
		Namespace:    req.Namespace,
		Replicas:     req.Replicas,
		RequestID:    requestIDFromContext(c),
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "create scale approval failed")
		return
	}

	writeSuccess(c, http.StatusAccepted, "scale approval requested", result)
}

func (h *ClusterHandler) RequestRestartDeployment(c *gin.Context) {
	clusterID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "current user is missing")
		return
	}

	var req struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "deployment name is required")
		return
	}

	result, err := h.actionRequester.RequestClusterAction(c.Request.Context(), applicationagent.ClusterActionRequestInput{
		ClusterID:    clusterID,
		TenantID:     currentUser.TenantID,
		UserID:       currentUser.ID,
		Action:       "restart_deployment",
		ResourceName: req.Name,
		Namespace:    req.Namespace,
		RequestID:    requestIDFromContext(c),
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "create restart approval failed")
		return
	}

	writeSuccess(c, http.StatusAccepted, "restart approval requested", result)
}

func (h *ClusterHandler) RequestApplyYAML(c *gin.Context) {
	clusterID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "current user is missing")
		return
	}

	var req struct {
		Namespace string `json:"namespace"`
		Manifest  string `json:"manifest"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Manifest == "" {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "manifest is required")
		return
	}

	result, err := h.actionRequester.RequestClusterAction(c.Request.Context(), applicationagent.ClusterActionRequestInput{
		ClusterID: clusterID,
		TenantID:  currentUser.TenantID,
		UserID:    currentUser.ID,
		Action:    "apply_yaml",
		Namespace: req.Namespace,
		Manifest:  req.Manifest,
		RequestID: requestIDFromContext(c),
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "create apply approval failed")
		return
	}

	writeSuccess(c, http.StatusAccepted, "apply approval requested", result)
}
