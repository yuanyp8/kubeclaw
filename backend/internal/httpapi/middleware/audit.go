package middleware

import (
	"encoding/json"
	"strings"
	"time"

	applicationaudit "kubeclaw/backend/internal/application/audit"
	"kubeclaw/backend/internal/logger"

	"github.com/gin-gonic/gin"
)

// AuditMiddleware 负责把 API 请求写入审计日志表。
type AuditMiddleware struct {
	service *applicationaudit.Service
}

func NewAuditMiddleware(service *applicationaudit.Service) *AuditMiddleware {
	return &AuditMiddleware{service: service}
}

// Record 在请求完成后记录审计信息，失败仅写日志，不影响主请求。
func (m *AuditMiddleware) Record() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()

		c.Next()

		fullPath := c.FullPath()
		if fullPath == "" || !strings.HasPrefix(fullPath, "/api") {
			return
		}
		if len(auditEntriesFromContext(c)) == 0 {
			return
		}

		var userID *int64
		var tenantID *int64
		var actor map[string]any
		if currentUser, ok := CurrentUser(c); ok {
			userID = &currentUser.ID
			tenantID = currentUser.TenantID
			actor = map[string]any{
				"id":          currentUser.ID,
				"username":    currentUser.Username,
				"displayName": currentUser.DisplayName,
				"role":        currentUser.Role,
				"tenantId":    currentUser.TenantID,
			}
		}

		for _, entry := range auditEntriesFromContext(c) {
			details := map[string]any{
				"method":    c.Request.Method,
				"route":     fullPath,
				"path":      c.Request.URL.Path,
				"query":     c.Request.URL.RawQuery,
				"status":    c.Writer.Status(),
				"latencyMs": time.Since(startedAt).Milliseconds(),
				"requestId": c.GetString(RequestIDKey),
				"userAgent": c.Request.UserAgent(),
				"actor":     actor,
				"change":    entry.Details,
			}

			detailsBytes, err := json.Marshal(details)
			if err != nil {
				logger.S().Warnw("marshal audit details failed", "error", err, "path", c.Request.URL.Path, "action", entry.Action)
				continue
			}

			if _, err := m.service.Create(c.Request.Context(), applicationaudit.CreateInput{
				TenantID: tenantID,
				UserID:   userID,
				Action:   entry.Action,
				Target:   entry.Target,
				Details:  string(detailsBytes),
				IP:       c.ClientIP(),
			}); err != nil {
				logger.S().Warnw("write audit log failed", "error", err, "path", c.Request.URL.Path, "action", entry.Action)
			}
		}
	}
}
