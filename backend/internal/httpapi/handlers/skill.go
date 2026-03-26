package handlers

import (
	"errors"
	"net/http"

	appskill "kubeclaw/backend/internal/application/skill"

	"github.com/gin-gonic/gin"
)

type SkillHandler struct{ service *appskill.Service }

func NewSkillHandler(service *appskill.Service) *SkillHandler { return &SkillHandler{service: service} }

func (h *SkillHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取技能列表失败")
		return
	}
	writeSuccess(c, http.StatusOK, "获取技能列表成功", items)
}

func (h *SkillHandler) Get(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, appskill.ErrNotFound) {
			writeError(c, http.StatusNotFound, "SKILL_NOT_FOUND", "技能不存在")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "获取技能详情失败")
		return
	}
	writeSuccess(c, http.StatusOK, "获取技能详情成功", item)
}

func (h *SkillHandler) Create(c *gin.Context) {
	var req appskill.CreateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "创建技能参数不完整")
		return
	}
	if req.Version == 0 {
		req.Version = 1
	}
	req.Status = defaultString(req.Status, "draft")
	item, err := h.service.Create(c.Request.Context(), req)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "创建技能失败")
		return
	}
	writeSuccess(c, http.StatusCreated, "创建技能成功", item)
}

func (h *SkillHandler) Update(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	var req appskill.UpdateInput
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "更新技能参数不完整")
		return
	}
	item, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		if errors.Is(err, appskill.ErrNotFound) {
			writeError(c, http.StatusNotFound, "SKILL_NOT_FOUND", "技能不存在")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "更新技能失败")
		return
	}
	writeSuccess(c, http.StatusOK, "更新技能成功", item)
}

func (h *SkillHandler) Delete(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "删除技能失败")
		return
	}
	writeSuccess(c, http.StatusOK, "删除技能成功", gin.H{"id": id})
}
