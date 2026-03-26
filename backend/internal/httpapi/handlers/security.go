package handlers

import (
	"errors"
	"net/http"

	appsecurity "kubeclaw/backend/internal/application/security"

	"github.com/gin-gonic/gin"
)

type SecurityHandler struct{ service *appsecurity.Service }

func NewSecurityHandler(service *appsecurity.Service) *SecurityHandler {
	return &SecurityHandler{service: service}
}

func (h *SecurityHandler) ListIPWhitelists(c *gin.Context) {
	items, err := h.service.ListIPWhitelists(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取白名单列表失败")
		return
	}
	writeSuccess(c, http.StatusOK, "获取白名单列表成功", items)
}

func (h *SecurityHandler) GetIPWhitelist(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	item, err := h.service.GetIPWhitelist(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, appsecurity.ErrNotFound) {
			writeError(c, http.StatusNotFound, "IP_WHITELIST_NOT_FOUND", "白名单规则不存在")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取白名单详情失败")
		return
	}
	writeSuccess(c, http.StatusOK, "获取白名单详情成功", item)
}

func (h *SecurityHandler) CreateIPWhitelist(c *gin.Context) {
	var req appsecurity.IPWhitelistInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "创建白名单参数不完整")
		return
	}
	req.Scope = defaultString(req.Scope, "global")
	item, err := h.service.CreateIPWhitelist(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "创建白名单失败")
		return
	}
	writeSuccess(c, http.StatusCreated, "创建白名单成功", item)
}

func (h *SecurityHandler) UpdateIPWhitelist(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req appsecurity.IPWhitelistInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "更新白名单参数不完整")
		return
	}
	item, err := h.service.UpdateIPWhitelist(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, appsecurity.ErrNotFound) {
			writeError(c, http.StatusNotFound, "IP_WHITELIST_NOT_FOUND", "白名单规则不存在")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "更新白名单失败")
		return
	}
	writeSuccess(c, http.StatusOK, "更新白名单成功", item)
}

func (h *SecurityHandler) DeleteIPWhitelist(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteIPWhitelist(c.Request.Context(), id); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "删除白名单失败")
		return
	}
	writeSuccess(c, http.StatusOK, "删除白名单成功", gin.H{"id": id})
}

func (h *SecurityHandler) ListSensitiveWords(c *gin.Context) {
	items, err := h.service.ListSensitiveWords(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取敏感词列表失败")
		return
	}
	writeSuccess(c, http.StatusOK, "获取敏感词列表成功", items)
}

func (h *SecurityHandler) GetSensitiveWord(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	item, err := h.service.GetSensitiveWord(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, appsecurity.ErrNotFound) {
			writeError(c, http.StatusNotFound, "SENSITIVE_WORD_NOT_FOUND", "敏感词规则不存在")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取敏感词详情失败")
		return
	}
	writeSuccess(c, http.StatusOK, "获取敏感词详情成功", item)
}

func (h *SecurityHandler) CreateSensitiveWord(c *gin.Context) {
	var req appsecurity.SensitiveWordInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "创建敏感词参数不完整")
		return
	}
	req.Category = defaultString(req.Category, "command")
	req.Level = defaultString(req.Level, "medium")
	req.Action = defaultString(req.Action, "review")
	item, err := h.service.CreateSensitiveWord(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "创建敏感词失败")
		return
	}
	writeSuccess(c, http.StatusCreated, "创建敏感词成功", item)
}

func (h *SecurityHandler) UpdateSensitiveWord(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req appsecurity.SensitiveWordInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "更新敏感词参数不完整")
		return
	}
	item, err := h.service.UpdateSensitiveWord(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, appsecurity.ErrNotFound) {
			writeError(c, http.StatusNotFound, "SENSITIVE_WORD_NOT_FOUND", "敏感词规则不存在")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "更新敏感词失败")
		return
	}
	writeSuccess(c, http.StatusOK, "更新敏感词成功", item)
}

func (h *SecurityHandler) DeleteSensitiveWord(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteSensitiveWord(c.Request.Context(), id); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "删除敏感词失败")
		return
	}
	writeSuccess(c, http.StatusOK, "删除敏感词成功", gin.H{"id": id})
}

func (h *SecurityHandler) ListSensitiveFieldRules(c *gin.Context) {
	items, err := h.service.ListSensitiveFieldRules(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取敏感字段规则列表失败")
		return
	}
	writeSuccess(c, http.StatusOK, "获取敏感字段规则列表成功", items)
}

func (h *SecurityHandler) GetSensitiveFieldRule(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	item, err := h.service.GetSensitiveFieldRule(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, appsecurity.ErrNotFound) {
			writeError(c, http.StatusNotFound, "SENSITIVE_FIELD_RULE_NOT_FOUND", "敏感字段规则不存在")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取敏感字段规则详情失败")
		return
	}
	writeSuccess(c, http.StatusOK, "获取敏感字段规则详情成功", item)
}

func (h *SecurityHandler) CreateSensitiveFieldRule(c *gin.Context) {
	var req appsecurity.SensitiveFieldRuleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "创建敏感字段规则参数不完整")
		return
	}
	req.Action = defaultString(req.Action, "mask")
	item, err := h.service.CreateSensitiveFieldRule(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "创建敏感字段规则失败")
		return
	}
	writeSuccess(c, http.StatusCreated, "创建敏感字段规则成功", item)
}

func (h *SecurityHandler) UpdateSensitiveFieldRule(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req appsecurity.SensitiveFieldRuleInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "更新敏感字段规则参数不完整")
		return
	}
	item, err := h.service.UpdateSensitiveFieldRule(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, appsecurity.ErrNotFound) {
			writeError(c, http.StatusNotFound, "SENSITIVE_FIELD_RULE_NOT_FOUND", "敏感字段规则不存在")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "更新敏感字段规则失败")
		return
	}
	writeSuccess(c, http.StatusOK, "更新敏感字段规则成功", item)
}

func (h *SecurityHandler) DeleteSensitiveFieldRule(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.DeleteSensitiveFieldRule(c.Request.Context(), id); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "删除敏感字段规则失败")
		return
	}
	writeSuccess(c, http.StatusOK, "删除敏感字段规则成功", gin.H{"id": id})
}
