package handlers

import (
	"errors"
	"fmt"
	"net/http"

	applicationtenant "kubeclaw/backend/internal/application/tenant"
	"kubeclaw/backend/internal/httpapi/middleware"

	"github.com/gin-gonic/gin"
)

type TenantHandler struct {
	service *applicationtenant.Service
}

type tenantRequest struct {
	Name        string `json:"name" binding:"required"`
	Slug        string `json:"slug" binding:"required"`
	Description string `json:"description"`
	Status      string `json:"status"`
	IsSystem    bool   `json:"isSystem"`
	OwnerUserID *int64 `json:"ownerUserId"`
}

func NewTenantHandler(service *applicationtenant.Service) *TenantHandler {
	return &TenantHandler{service: service}
}

func (h *TenantHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load tenants failed")
		return
	}
	writeSuccess(c, http.StatusOK, "tenants loaded", items)
}

func (h *TenantHandler) Get(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, applicationtenant.ErrNotFound) {
			writeError(c, http.StatusNotFound, "TENANT_NOT_FOUND", "tenant was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load tenant failed")
		return
	}

	writeSuccess(c, http.StatusOK, "tenant loaded", item)
}

func (h *TenantHandler) Create(c *gin.Context) {
	var req tenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid tenant payload")
		return
	}

	item, err := h.service.Create(c.Request.Context(), applicationtenant.Input{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Status:      defaultString(req.Status, "active"),
		IsSystem:    req.IsSystem,
		OwnerUserID: req.OwnerUserID,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "create tenant failed")
		return
	}

	middleware.AppendAuditEntry(c, middleware.AuditEntry{
		Action: "tenant.create",
		Target: fmt.Sprintf("tenant:%d", item.ID),
		Details: map[string]any{
			"resourceId":  item.ID,
			"name":        item.Name,
			"slug":        item.Slug,
			"status":      item.Status,
			"isSystem":    item.IsSystem,
			"ownerUserId": item.OwnerUserID,
		},
	})

	writeSuccess(c, http.StatusCreated, "tenant created", item)
}

func (h *TenantHandler) Update(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	before, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, applicationtenant.ErrNotFound) {
			writeError(c, http.StatusNotFound, "TENANT_NOT_FOUND", "tenant was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load tenant before update failed")
		return
	}

	var req tenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid tenant payload")
		return
	}

	item, err := h.service.Update(c.Request.Context(), id, applicationtenant.Input{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Status:      defaultString(req.Status, "active"),
		IsSystem:    req.IsSystem,
		OwnerUserID: req.OwnerUserID,
	})
	if err != nil {
		if errors.Is(err, applicationtenant.ErrNotFound) {
			writeError(c, http.StatusNotFound, "TENANT_NOT_FOUND", "tenant was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "update tenant failed")
		return
	}

	middleware.AppendAuditEntry(c, middleware.AuditEntry{
		Action: "tenant.update",
		Target: fmt.Sprintf("tenant:%d", item.ID),
		Details: map[string]any{
			"resourceId": item.ID,
			"before": map[string]any{
				"name":        before.Name,
				"slug":        before.Slug,
				"status":      before.Status,
				"isSystem":    before.IsSystem,
				"ownerUserId": before.OwnerUserID,
			},
			"after": map[string]any{
				"name":        item.Name,
				"slug":        item.Slug,
				"status":      item.Status,
				"isSystem":    item.IsSystem,
				"ownerUserId": item.OwnerUserID,
			},
		},
	})

	writeSuccess(c, http.StatusOK, "tenant updated", item)
}

func (h *TenantHandler) Delete(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	before, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, applicationtenant.ErrNotFound) {
			writeError(c, http.StatusNotFound, "TENANT_NOT_FOUND", "tenant was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load tenant before delete failed")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "delete tenant failed")
		return
	}

	middleware.AppendAuditEntry(c, middleware.AuditEntry{
		Action: "tenant.delete",
		Target: fmt.Sprintf("tenant:%d", id),
		Details: map[string]any{
			"resourceId": id,
			"before": map[string]any{
				"name":      before.Name,
				"slug":      before.Slug,
				"status":    before.Status,
				"userCount": before.UserCount,
				"teamCount": before.TeamCount,
			},
		},
	})

	writeSuccess(c, http.StatusOK, "tenant deleted", gin.H{"id": id})
}
