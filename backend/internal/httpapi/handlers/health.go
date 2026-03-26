package handlers

import (
	"time"

	"kubeclaw/backend/internal/config"

	"github.com/gin-gonic/gin"
)

// HealthHandler 提供健康检查与就绪检查接口。
type HealthHandler struct {
	cfg       config.Config
	startedAt time.Time
}

// NewHealthHandler 创建健康检查处理器。
func NewHealthHandler(cfg config.Config) *HealthHandler {
	return &HealthHandler{
		cfg:       cfg,
		startedAt: time.Now(),
	}
}

// Get 返回服务基础状态，便于上游探活与排障。
func (h *HealthHandler) Get(c *gin.Context) {
	writeSuccess(c, 200, "服务健康状态正常", gin.H{
		"service":   h.cfg.AppName,
		"env":       h.cfg.Env,
		"version":   h.cfg.AppVersion,
		"startedAt": h.startedAt.Format(time.RFC3339),
		"uptimeSec": int(time.Since(h.startedAt).Seconds()),
	})
}

// Ready 用于 Kubernetes readiness probe。
func (h *HealthHandler) Ready(c *gin.Context) {
	writeSuccess(c, 200, "服务已就绪", gin.H{
		"service": h.cfg.AppName,
		"ready":   true,
	})
}
