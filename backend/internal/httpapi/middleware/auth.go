package middleware

import (
	"errors"
	"net/http"
	"strings"

	applicationauth "kubeclaw/backend/internal/application/auth"
	domainuser "kubeclaw/backend/internal/domain/user"

	"github.com/gin-gonic/gin"
)

const CurrentUserKey = "current_user"

// AuthMiddleware 统一处理访问令牌校验和角色校验。
type AuthMiddleware struct {
	authService *applicationauth.Service
}

func NewAuthMiddleware(authService *applicationauth.Service) *AuthMiddleware {
	return &AuthMiddleware{authService: authService}
}

// RequireAuth 校验 Bearer Token，并把当前用户写入上下文。
func (m *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := ExtractBearerToken(c.GetHeader("Authorization"))
		if token == "" {
			writeAuthError(c, http.StatusUnauthorized, "UNAUTHORIZED", "缺少 Bearer Token")
			c.Abort()
			return
		}

		currentUser, err := m.authService.AuthenticateAccessToken(c.Request.Context(), token)
		if err != nil {
			switch {
			case errors.Is(err, applicationauth.ErrInvalidAccessToken):
				writeAuthError(c, http.StatusUnauthorized, "INVALID_ACCESS_TOKEN", "访问令牌无效或已过期")
			case errors.Is(err, applicationauth.ErrInactiveUser):
				writeAuthError(c, http.StatusForbidden, "USER_DISABLED", "当前用户已被禁用")
			default:
				writeAuthError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "认证过程发生内部错误")
			}

			c.Abort()
			return
		}

		c.Set(CurrentUserKey, currentUser)
		c.Next()
	}
}

// RequireRoles 限制只有指定角色的用户才能访问。
func (m *AuthMiddleware) RequireRoles(roles ...domainuser.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		currentUser, ok := CurrentUser(c)
		if !ok {
			writeAuthError(c, http.StatusUnauthorized, "UNAUTHORIZED", "当前请求未完成身份认证")
			c.Abort()
			return
		}

		for _, role := range roles {
			if currentUser.Role == role {
				c.Next()
				return
			}
		}

		writeAuthError(c, http.StatusForbidden, "FORBIDDEN", "当前角色无权访问该接口")
		c.Abort()
	}
}

func CurrentUser(c *gin.Context) (*domainuser.User, bool) {
	value, ok := c.Get(CurrentUserKey)
	if !ok {
		return nil, false
	}

	currentUser, ok := value.(*domainuser.User)
	return currentUser, ok
}

func ExtractBearerToken(authorization string) string {
	parts := strings.SplitN(strings.TrimSpace(authorization), " ", 2)
	if len(parts) != 2 {
		return ""
	}

	if !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}

	return strings.TrimSpace(parts[1])
}

func writeAuthError(c *gin.Context, status int, code, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"code":      code,
		"message":   message,
		"requestId": c.GetString(RequestIDKey),
	})
}
