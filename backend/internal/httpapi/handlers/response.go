package handlers

import (
	"net/http"

	"kubeclaw/backend/internal/httpapi/middleware"

	"github.com/gin-gonic/gin"
)

// writeSuccess 统一成功响应格式，减少 handler 中的重复代码。
func writeSuccess(c *gin.Context, status int, message string, data any) {
	c.JSON(status, gin.H{
		"code":      http.StatusText(status),
		"message":   message,
		"requestId": requestIDFromContext(c),
		"data":      data,
	})
}

// writeError 统一错误响应格式。
func writeError(c *gin.Context, status int, code, message string) {
	c.JSON(status, gin.H{
		"code":      code,
		"message":   message,
		"requestId": requestIDFromContext(c),
	})
}

func requestIDFromContext(c *gin.Context) string {
	return c.GetString(middleware.RequestIDKey)
}
