package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	applicationteam "kubeclaw/backend/internal/application/team"
	"kubeclaw/backend/internal/httpapi/middleware"

	"github.com/gin-gonic/gin"
)

type TeamHandler struct {
	service *applicationteam.Service
}

type teamRequest struct {
	TenantID    *int64 `json:"tenantId"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	OwnerUserID *int64 `json:"ownerUserId"`
	Visibility  string `json:"visibility"`
}

type teamMemberRequest struct {
	UserID int64  `json:"userId" binding:"required"`
	Role   string `json:"role"`
}

func NewTeamHandler(service *applicationteam.Service) *TeamHandler {
	return &TeamHandler{service: service}
}

func (h *TeamHandler) List(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	items, err := h.service.List(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load teams failed")
		return
	}

	if !isAdminUser(currentUser) {
		filtered := make([]applicationteam.Record, 0, len(items))
		for _, item := range items {
			if currentUser.TenantID != nil && item.TenantID != nil && *currentUser.TenantID == *item.TenantID {
				filtered = append(filtered, item)
			}
		}
		items = filtered
	}
	writeSuccess(c, http.StatusOK, "teams loaded", items)
}

func (h *TeamHandler) Get(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, applicationteam.ErrNotFound) {
			writeError(c, http.StatusNotFound, "TEAM_NOT_FOUND", "team was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load team failed")
		return
	}
	if !ensureTenantAccess(c, currentUser, item.TenantID) {
		return
	}

	writeSuccess(c, http.StatusOK, "team loaded", item)
}

func (h *TeamHandler) Create(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	var req teamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid team payload")
		return
	}

	if req.OwnerUserID == nil {
		req.OwnerUserID = &currentUser.ID
	}
	if req.TenantID == nil {
		req.TenantID = currentUser.TenantID
	}
	if !ensureTenantAccess(c, currentUser, req.TenantID) {
		return
	}

	item, err := h.service.Create(c.Request.Context(), applicationteam.Input{
		TenantID:    req.TenantID,
		Name:        req.Name,
		Description: req.Description,
		OwnerUserID: req.OwnerUserID,
		Visibility:  defaultString(req.Visibility, "private"),
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "create team failed")
		return
	}

	middleware.AppendAuditEntry(c, middleware.AuditEntry{
		Action: "team.create",
		Target: fmt.Sprintf("team:%d", item.ID),
		Details: map[string]any{
			"resourceId":  item.ID,
			"tenantId":    item.TenantID,
			"name":        item.Name,
			"ownerUserId": item.OwnerUserID,
			"visibility":  item.Visibility,
		},
	})

	writeSuccess(c, http.StatusCreated, "team created", item)
}

func (h *TeamHandler) Update(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	before, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, applicationteam.ErrNotFound) {
			writeError(c, http.StatusNotFound, "TEAM_NOT_FOUND", "team was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load team before update failed")
		return
	}
	if !ensureTenantAccess(c, currentUser, before.TenantID) {
		return
	}

	var req teamRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid team payload")
		return
	}
	if req.TenantID == nil {
		req.TenantID = before.TenantID
	}
	if !ensureTenantAccess(c, currentUser, req.TenantID) {
		return
	}

	item, err := h.service.Update(c.Request.Context(), id, applicationteam.Input{
		TenantID:    req.TenantID,
		Name:        req.Name,
		Description: req.Description,
		OwnerUserID: req.OwnerUserID,
		Visibility:  defaultString(req.Visibility, "private"),
	})
	if err != nil {
		if errors.Is(err, applicationteam.ErrNotFound) {
			writeError(c, http.StatusNotFound, "TEAM_NOT_FOUND", "team was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "update team failed")
		return
	}

	middleware.AppendAuditEntry(c, middleware.AuditEntry{
		Action: "team.update",
		Target: fmt.Sprintf("team:%d", item.ID),
		Details: map[string]any{
			"resourceId": item.ID,
			"before": map[string]any{
				"name":        before.Name,
				"description": before.Description,
				"ownerUserId": before.OwnerUserID,
				"visibility":  before.Visibility,
			},
			"after": map[string]any{
				"name":        item.Name,
				"description": item.Description,
				"ownerUserId": item.OwnerUserID,
				"visibility":  item.Visibility,
			},
		},
	})

	writeSuccess(c, http.StatusOK, "team updated", item)
}

func (h *TeamHandler) Delete(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	before, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, applicationteam.ErrNotFound) {
			writeError(c, http.StatusNotFound, "TEAM_NOT_FOUND", "team was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load team before delete failed")
		return
	}
	if !ensureTenantAccess(c, currentUser, before.TenantID) {
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "delete team failed")
		return
	}

	middleware.AppendAuditEntry(c, middleware.AuditEntry{
		Action: "team.delete",
		Target: fmt.Sprintf("team:%d", id),
		Details: map[string]any{
			"resourceId": id,
			"before": map[string]any{
				"name":        before.Name,
				"description": before.Description,
				"ownerUserId": before.OwnerUserID,
				"visibility":  before.Visibility,
				"memberCount": before.MemberCount,
			},
		},
	})

	writeSuccess(c, http.StatusOK, "team deleted", gin.H{"id": id})
}

func (h *TeamHandler) ListMembers(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	teamID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	team, err := h.service.Get(c.Request.Context(), teamID)
	if err != nil {
		if errors.Is(err, applicationteam.ErrNotFound) {
			writeError(c, http.StatusNotFound, "TEAM_NOT_FOUND", "team was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load team failed")
		return
	}
	if !ensureTenantAccess(c, currentUser, team.TenantID) {
		return
	}

	items, err := h.service.ListMembers(c.Request.Context(), teamID)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load team members failed")
		return
	}

	writeSuccess(c, http.StatusOK, "team members loaded", items)
}

func (h *TeamHandler) AddMember(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	teamID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	team, err := h.service.Get(c.Request.Context(), teamID)
	if err != nil {
		if errors.Is(err, applicationteam.ErrNotFound) {
			writeError(c, http.StatusNotFound, "TEAM_NOT_FOUND", "team was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load team failed")
		return
	}
	if !ensureTenantAccess(c, currentUser, team.TenantID) {
		return
	}

	var req teamMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid team member payload")
		return
	}

	item, err := h.service.AddMember(c.Request.Context(), teamID, applicationteam.AddMemberInput{
		UserID: req.UserID,
		Role:   defaultString(req.Role, "member"),
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "add team member failed")
		return
	}

	middleware.AppendAuditEntry(c, middleware.AuditEntry{
		Action: "team.member.add",
		Target: fmt.Sprintf("team:%d/member:%d", teamID, req.UserID),
		Details: map[string]any{
			"teamId":      teamID,
			"userId":      req.UserID,
			"displayName": item.DisplayName,
			"role":        item.Role,
		},
	})

	writeSuccess(c, http.StatusCreated, "team member created", item)
}

func (h *TeamHandler) RemoveMember(c *gin.Context) {
	currentUser, ok := requireCurrentUser(c)
	if !ok {
		return
	}

	teamID, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	team, err := h.service.Get(c.Request.Context(), teamID)
	if err != nil {
		if errors.Is(err, applicationteam.ErrNotFound) {
			writeError(c, http.StatusNotFound, "TEAM_NOT_FOUND", "team was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load team failed")
		return
	}
	if !ensureTenantAccess(c, currentUser, team.TenantID) {
		return
	}

	userID, err := strconv.ParseInt(c.Param("userId"), 10, 64)
	if err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "team member user id is invalid")
		return
	}

	if err := h.service.RemoveMember(c.Request.Context(), teamID, userID); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "remove team member failed")
		return
	}

	middleware.AppendAuditEntry(c, middleware.AuditEntry{
		Action: "team.member.remove",
		Target: fmt.Sprintf("team:%d/member:%d", teamID, userID),
		Details: map[string]any{
			"teamId": teamID,
			"userId": userID,
		},
	})

	writeSuccess(c, http.StatusOK, "team member removed", gin.H{
		"teamId": teamID,
		"userId": userID,
	})
}
