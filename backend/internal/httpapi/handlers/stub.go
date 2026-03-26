package handlers

import "github.com/gin-gonic/gin"

// StubHandler 用于保留 README 中已规划但尚未实现的接口。
type StubHandler struct{}

func NewStubHandler() *StubHandler {
	return &StubHandler{}
}

func (h *StubHandler) Handle(module, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(501, gin.H{
			"code":      "NOT_IMPLEMENTED",
			"message":   "该接口已预留，但当前阶段尚未实现",
			"requestId": requestIDFromContext(c),
			"data": gin.H{
				"module": module,
				"action": action,
				"method": c.Request.Method,
				"path":   c.FullPath(),
			},
		})
	}
}
