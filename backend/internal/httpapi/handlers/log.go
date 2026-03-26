package handlers

import (
	"net/http"
	"strconv"

	applicationaudit "kubeclaw/backend/internal/application/audit"
	applicationlogs "kubeclaw/backend/internal/application/logs"

	"github.com/gin-gonic/gin"
)

type LogHandler struct {
	service      *applicationlogs.Service
	auditService *applicationaudit.Service
}

func NewLogHandler(service *applicationlogs.Service, auditService *applicationaudit.Service) *LogHandler {
	return &LogHandler{
		service:      service,
		auditService: auditService,
	}
}

func (h *LogHandler) ListScopes(c *gin.Context) {
	writeSuccess(c, http.StatusOK, "log scopes loaded", h.service.ListScopes())
}

func (h *LogHandler) List(c *gin.Context) {
	cursor, _ := strconv.ParseInt(c.Query("cursor"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))

	var auditItems []applicationaudit.Record
	if c.Query("scope") == "audit" {
		items, err := h.auditService.List(c.Request.Context())
		if err != nil {
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load audit logs failed")
			return
		}
		auditItems = items
	}

	result, err := h.service.Query(c.Request.Context(), applicationlogs.QueryInput{
		Scope:  c.Query("scope"),
		Cursor: cursor,
		Limit:  limit,
	}, auditItems)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load logs failed")
		return
	}

	writeSuccess(c, http.StatusOK, "logs loaded", result)
}

func (h *LogHandler) CreateClientLog(c *gin.Context) {
	var req applicationlogs.ClientLogInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid client log payload")
		return
	}

	entry := h.service.RecordClient(c.Request.Context(), req)
	writeSuccess(c, http.StatusCreated, "client log stored", entry)
}
