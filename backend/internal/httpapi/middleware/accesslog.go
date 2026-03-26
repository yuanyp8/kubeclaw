package middleware

import (
	"time"

	"kubeclaw/backend/internal/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func AccessLog() gin.HandlerFunc {
	log := logger.ForScope(logger.ScopeAccess)

	return func(c *gin.Context) {
		startedAt := time.Now()
		path := c.Request.URL.Path
		rawQuery := c.Request.URL.RawQuery

		c.Next()

		if rawQuery != "" {
			path += "?" + rawQuery
		}

		log.Info(
			"http access",
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.String("latency", time.Since(startedAt).String()),
			zap.String("client_ip", c.ClientIP()),
			zap.String("request_id", c.GetString(RequestIDKey)),
		)
	}
}
