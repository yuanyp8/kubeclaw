package handlers

import (
	"errors"
	"net/http"

	applicationmcp "kubeclaw/backend/internal/application/mcp"

	"github.com/gin-gonic/gin"
)

type MCPHandler struct{ service *applicationmcp.Service }

func NewMCPHandler(service *applicationmcp.Service) *MCPHandler { return &MCPHandler{service: service} }

func (h *MCPHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取 MCP 列表失败")
		return
	}
	writeSuccess(c, http.StatusOK, "获取 MCP 列表成功", items)
}

func (h *MCPHandler) Get(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, applicationmcp.ErrNotFound) {
			writeError(c, http.StatusNotFound, "MCP_NOT_FOUND", "MCP 服务不存在")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取 MCP 详情失败")
		return
	}
	writeSuccess(c, http.StatusOK, "获取 MCP 详情成功", item)
}

func (h *MCPHandler) Create(c *gin.Context) {
	var req applicationmcp.CreateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "创建 MCP 参数不完整")
		return
	}
	req.Type = defaultString(req.Type, "custom")
	req.Transport = defaultString(req.Transport, "http")
	req.AuthType = defaultString(req.AuthType, "none")
	req.HealthStatus = defaultString(req.HealthStatus, "unknown")
	item, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "创建 MCP 失败")
		return
	}
	writeSuccess(c, http.StatusCreated, "创建 MCP 成功", item)
}

func (h *MCPHandler) Update(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req applicationmcp.UpdateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "更新 MCP 参数不完整")
		return
	}
	item, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, applicationmcp.ErrNotFound) {
			writeError(c, http.StatusNotFound, "MCP_NOT_FOUND", "MCP 服务不存在")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "更新 MCP 失败")
		return
	}
	writeSuccess(c, http.StatusOK, "更新 MCP 成功", item)
}

func (h *MCPHandler) Delete(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "删除 MCP 失败")
		return
	}
	writeSuccess(c, http.StatusOK, "删除 MCP 成功", gin.H{"id": id})
}
