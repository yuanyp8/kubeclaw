package handlers

import (
	"errors"
	"net/http"

	applicationauth "kubeclaw/backend/internal/application/auth"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService *applicationauth.Service
}

type loginRequest struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

func NewAuthHandler(authService *applicationauth.Service) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "登录参数不完整")
		return
	}

	result, err := h.authService.Login(c.Request.Context(), req.Login, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, applicationauth.ErrInvalidCredentials):
			writeError(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "用户名/邮箱或密码错误")
		case errors.Is(err, applicationauth.ErrInactiveUser):
			writeError(c, http.StatusForbidden, "USER_DISABLED", "当前用户已被禁用")
		default:
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "登录过程中发生内部错误")
		}
		return
	}

	writeSuccess(c, http.StatusOK, "登录成功", result)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "缺少 refreshToken 参数")
		return
	}

	result, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		switch {
		case errors.Is(err, applicationauth.ErrInvalidRefreshToken):
			writeError(c, http.StatusUnauthorized, "INVALID_REFRESH_TOKEN", "刷新令牌无效或已过期")
		case errors.Is(err, applicationauth.ErrInactiveUser):
			writeError(c, http.StatusForbidden, "USER_DISABLED", "当前用户已被禁用")
		default:
			writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "刷新令牌时发生内部错误")
		}
		return
	}

	writeSuccess(c, http.StatusOK, "刷新令牌成功", result)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	writeSuccess(c, http.StatusOK, "已接受登出请求", gin.H{
		"stateless": true,
		"note":      "当前版本使用无状态 JWT，后续接入 Redis 后可支持服务端失效控制",
	})
}
