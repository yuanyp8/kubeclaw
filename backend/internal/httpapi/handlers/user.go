package handlers

import (
	"errors"
	"fmt"
	"net/http"

	applicationuser "kubeclaw/backend/internal/application/user"
	domainuser "kubeclaw/backend/internal/domain/user"
	"kubeclaw/backend/internal/httpapi/middleware"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	userService *applicationuser.Service
}

type createUserRequest struct {
	TenantID    *int64 `json:"tenantId"`
	Username    string `json:"username" binding:"required"`
	Email       string `json:"email" binding:"required"`
	DisplayName string `json:"displayName"`
	Phone       string `json:"phone"`
	AvatarURL   string `json:"avatarUrl"`
	Role        string `json:"role"`
	Status      string `json:"status"`
	Password    string `json:"password" binding:"required"`
}

type updateUserRequest struct {
	TenantID    *int64 `json:"tenantId"`
	Email       string `json:"email" binding:"required"`
	DisplayName string `json:"displayName"`
	Phone       string `json:"phone"`
	AvatarURL   string `json:"avatarUrl"`
	Role        string `json:"role" binding:"required"`
	Status      string `json:"status" binding:"required"`
	Password    string `json:"password"`
}

func NewUserHandler(userService *applicationuser.Service) *UserHandler {
	return &UserHandler{userService: userService}
}

func (h *UserHandler) GetMe(c *gin.Context) {
	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "current user is missing")
		return
	}

	profile, err := h.userService.GetProfile(c.Request.Context(), currentUser.ID)
	if err != nil {
		if errors.Is(err, applicationuser.ErrNotFound) {
			writeError(c, http.StatusNotFound, "USER_NOT_FOUND", "user was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load current user failed")
		return
	}

	writeSuccess(c, http.StatusOK, "current user loaded", profile)
}

func (h *UserHandler) List(c *gin.Context) {
	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "current user is missing")
		return
	}

	var (
		profiles []applicationuser.Profile
		err      error
	)

	switch currentUser.Role {
	case domainuser.RoleAdmin:
		profiles, err = h.userService.List(c.Request.Context())
	case domainuser.RoleClusterAdmin:
		if currentUser.TenantID == nil {
			writeError(c, http.StatusForbidden, "FORBIDDEN", "cluster admin is not bound to a tenant")
			return
		}
		profiles, err = h.userService.ListByTenant(c.Request.Context(), *currentUser.TenantID)
	default:
		writeError(c, http.StatusForbidden, "FORBIDDEN", "you do not have permission to access users")
		return
	}

	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load users failed")
		return
	}
	writeSuccess(c, http.StatusOK, "users loaded", profiles)
}

func (h *UserHandler) Get(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	profile, err := h.userService.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, applicationuser.ErrNotFound) {
			writeError(c, http.StatusNotFound, "USER_NOT_FOUND", "user was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load user failed")
		return
	}

	writeSuccess(c, http.StatusOK, "user loaded", profile)
}

func (h *UserHandler) Create(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid user payload")
		return
	}

	profile, err := h.userService.Create(c.Request.Context(), applicationuser.CreateInput{
		TenantID:    req.TenantID,
		Username:    req.Username,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Phone:       req.Phone,
		AvatarURL:   req.AvatarURL,
		Role:        defaultString(req.Role, "user"),
		Status:      defaultString(req.Status, "active"),
		Password:    req.Password,
	})
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "create user failed")
		return
	}

	middleware.AppendAuditEntry(c, middleware.AuditEntry{
		Action: "user.create",
		Target: fmt.Sprintf("user:%d", profile.ID),
		Details: map[string]any{
			"resourceId": profile.ID,
			"after": map[string]any{
				"tenantId":    profile.TenantID,
				"username":    profile.Username,
				"email":       profile.Email,
				"displayName": profile.DisplayName,
				"role":        profile.Role,
				"status":      profile.Status,
			},
		},
	})

	writeSuccess(c, http.StatusCreated, "user created", profile)
}

func (h *UserHandler) Update(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	before, err := h.userService.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, applicationuser.ErrNotFound) {
			writeError(c, http.StatusNotFound, "USER_NOT_FOUND", "user was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load user before update failed")
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid user payload")
		return
	}

	profile, err := h.userService.Update(c.Request.Context(), id, applicationuser.UpdateInput{
		TenantID:    req.TenantID,
		Email:       req.Email,
		DisplayName: req.DisplayName,
		Phone:       req.Phone,
		AvatarURL:   req.AvatarURL,
		Role:        req.Role,
		Status:      req.Status,
		Password:    req.Password,
	})
	if err != nil {
		if errors.Is(err, applicationuser.ErrNotFound) {
			writeError(c, http.StatusNotFound, "USER_NOT_FOUND", "user was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "update user failed")
		return
	}

	middleware.AppendAuditEntry(c, middleware.AuditEntry{
		Action: "user.update",
		Target: fmt.Sprintf("user:%d", profile.ID),
		Details: map[string]any{
			"resourceId": profile.ID,
			"before": map[string]any{
				"tenantId":    before.TenantID,
				"email":       before.Email,
				"displayName": before.DisplayName,
				"role":        before.Role,
				"status":      before.Status,
			},
			"after": map[string]any{
				"tenantId":    profile.TenantID,
				"email":       profile.Email,
				"displayName": profile.DisplayName,
				"role":        profile.Role,
				"status":      profile.Status,
			},
		},
	})

	writeSuccess(c, http.StatusOK, "user updated", profile)
}

func (h *UserHandler) Delete(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	before, err := h.userService.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, applicationuser.ErrNotFound) {
			writeError(c, http.StatusNotFound, "USER_NOT_FOUND", "user was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load user before delete failed")
		return
	}

	if err := h.userService.Delete(c.Request.Context(), id); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "delete user failed")
		return
	}

	middleware.AppendAuditEntry(c, middleware.AuditEntry{
		Action: "user.delete",
		Target: fmt.Sprintf("user:%d", id),
		Details: map[string]any{
			"resourceId": id,
			"before": map[string]any{
				"tenantId":    before.TenantID,
				"username":    before.Username,
				"email":       before.Email,
				"displayName": before.DisplayName,
				"role":        before.Role,
				"status":      before.Status,
			},
		},
	})

	writeSuccess(c, http.StatusOK, "user deleted", gin.H{"id": id})
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
