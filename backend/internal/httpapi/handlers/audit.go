package handlers

import (
	"errors"
	"net/http"

	applicationaudit "kubeclaw/backend/internal/application/audit"

	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	service *applicationaudit.Service
}

func NewAuditHandler(service *applicationaudit.Service) *AuditHandler {
	return &AuditHandler{service: service}
}

func (h *AuditHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取审计日志失败")
		return
	}
	writeSuccess(c, http.StatusOK, "获取审计日志成功", items)
}

func (h *AuditHandler) Get(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, applicationaudit.ErrNotFound) {
			writeError(c, http.StatusNotFound, "AUDIT_LOG_NOT_FOUND", "审计日志不存在")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取审计日志详情失败")
		return
	}

	writeSuccess(c, http.StatusOK, "获取审计日志详情成功", item)
}
