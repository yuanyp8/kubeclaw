package handlers

import (
	"net/http"

	domainuser "kubeclaw/backend/internal/domain/user"
	"kubeclaw/backend/internal/httpapi/middleware"

	"github.com/gin-gonic/gin"
)

// requireCurrentUser 统一读取当前登录用户，避免各 Handler 重复处理。
func requireCurrentUser(c *gin.Context) (*domainuser.User, bool) {
	currentUser, ok := middleware.CurrentUser(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "current user is missing")
		return nil, false
	}
	return currentUser, true
}

func isAdminUser(user *domainuser.User) bool {
	return user != nil && user.Role == domainuser.RoleAdmin
}

// ensureTenantAccess 用于限制非管理员只能访问自己租户下的数据。
func ensureTenantAccess(c *gin.Context, currentUser *domainuser.User, tenantID *int64) bool {
	if isAdminUser(currentUser) {
		return true
	}

	if currentUser.TenantID == nil || tenantID == nil || *currentUser.TenantID != *tenantID {
		writeError(c, http.StatusForbidden, "FORBIDDEN", "you do not have permission to access this tenant resource")
		return false
	}

	return true
}

// ensureUserOwnedResource 用于限制普通用户只能访问自己的会话、运行和审批。
func ensureUserOwnedResource(c *gin.Context, currentUser *domainuser.User, ownerUserID int64) bool {
	if isAdminUser(currentUser) || currentUser.ID == ownerUserID {
		return true
	}

	writeError(c, http.StatusForbidden, "FORBIDDEN", "you do not have permission to access this resource")
	return false
}
