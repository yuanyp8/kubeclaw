package handlers

import (
	"errors"
	"fmt"
	"net/http"

	applicationmodel "kubeclaw/backend/internal/application/model"
	"kubeclaw/backend/internal/httpapi/middleware"

	"github.com/gin-gonic/gin"
)

type ModelHandler struct{ service *applicationmodel.Service }

type modelRequest struct {
	TenantID       *int64   `json:"tenantId"`
	Name           string   `json:"name"`
	Provider       string   `json:"provider"`
	Model          string   `json:"model"`
	BaseURL        string   `json:"baseUrl"`
	APIKey         string   `json:"apiKey"`
	Description    string   `json:"description"`
	Capabilities   []string `json:"capabilities"`
	IsDefault      bool     `json:"isDefault"`
	IsEnabled      bool     `json:"isEnabled"`
	MaxTokens      int      `json:"maxTokens"`
	Temperature    float64  `json:"temperature"`
	TopP           float64  `json:"topP"`
	TestBeforeSave bool     `json:"testBeforeSave"`
}

func NewModelHandler(service *applicationmodel.Service) *ModelHandler {
	return &ModelHandler{service: service}
}

func (h *ModelHandler) List(c *gin.Context) {
	items, err := h.service.List(c.Request.Context())
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load models failed")
		return
	}
	writeSuccess(c, http.StatusOK, "models loaded", items)
}

func (h *ModelHandler) Get(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}
	item, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, applicationmodel.ErrNotFound) {
			writeError(c, http.StatusNotFound, "MODEL_NOT_FOUND", "model config was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load model failed")
		return
	}
	writeSuccess(c, http.StatusOK, "model loaded", item)
}

func (h *ModelHandler) Create(c *gin.Context) {
	var req modelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid model payload")
		return
	}

	input := applicationmodel.CreateInput{
		TenantID:     req.TenantID,
		Name:         req.Name,
		Provider:     req.Provider,
		Model:        req.Model,
		BaseURL:      req.BaseURL,
		APIKey:       req.APIKey,
		Description:  req.Description,
		Capabilities: req.Capabilities,
		IsDefault:    req.IsDefault,
		IsEnabled:    req.IsEnabled,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		TopP:         req.TopP,
	}

	if req.TestBeforeSave {
		if _, err := h.service.TestDraft(c.Request.Context(), input); err != nil {
			writeError(c, http.StatusBadGateway, "MODEL_TEST_FAILED", err.Error())
			return
		}
	}

	item, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "create model failed")
		return
	}

	middleware.AppendAuditEntry(c, middleware.AuditEntry{
		Action: "model.create",
		Target: fmt.Sprintf("model:%d", item.ID),
		Details: map[string]any{
			"resourceId": item.ID,
			"after": map[string]any{
				"name":      item.Name,
				"provider":  item.Provider,
				"model":     item.Model,
				"baseUrl":   item.BaseURL,
				"isDefault": item.IsDefault,
				"isEnabled": item.IsEnabled,
			},
			"tested": req.TestBeforeSave,
		},
	})

	writeSuccess(c, http.StatusCreated, "model created", item)
}

func (h *ModelHandler) Update(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	before, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, applicationmodel.ErrNotFound) {
			writeError(c, http.StatusNotFound, "MODEL_NOT_FOUND", "model config was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load model before update failed")
		return
	}

	var req modelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_ARGUMENT", "invalid model payload")
		return
	}

	input := applicationmodel.UpdateInput{
		TenantID:     req.TenantID,
		Name:         req.Name,
		Provider:     req.Provider,
		Model:        req.Model,
		BaseURL:      req.BaseURL,
		APIKey:       req.APIKey,
		Description:  req.Description,
		Capabilities: req.Capabilities,
		IsDefault:    req.IsDefault,
		IsEnabled:    req.IsEnabled,
		MaxTokens:    req.MaxTokens,
		Temperature:  req.Temperature,
		TopP:         req.TopP,
	}

	if req.TestBeforeSave {
		if _, err := h.service.TestUpdatedDraft(c.Request.Context(), id, input); err != nil {
			writeError(c, http.StatusBadGateway, "MODEL_TEST_FAILED", err.Error())
			return
		}
	}

	item, err := h.service.Update(c.Request.Context(), id, input)
	if err != nil {
		if errors.Is(err, applicationmodel.ErrNotFound) {
			writeError(c, http.StatusNotFound, "MODEL_NOT_FOUND", "model config was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "update model failed")
		return
	}

	middleware.AppendAuditEntry(c, middleware.AuditEntry{
		Action: "model.update",
		Target: fmt.Sprintf("model:%d", item.ID),
		Details: map[string]any{
			"resourceId": item.ID,
			"before": map[string]any{
				"name":      before.Name,
				"provider":  before.Provider,
				"model":     before.Model,
				"baseUrl":   before.BaseURL,
				"isDefault": before.IsDefault,
				"isEnabled": before.IsEnabled,
			},
			"after": map[string]any{
				"name":      item.Name,
				"provider":  item.Provider,
				"model":     item.Model,
				"baseUrl":   item.BaseURL,
				"isDefault": item.IsDefault,
				"isEnabled": item.IsEnabled,
			},
			"tested": req.TestBeforeSave,
		},
	})

	writeSuccess(c, http.StatusOK, "model updated", item)
}

func (h *ModelHandler) Delete(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	before, err := h.service.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, applicationmodel.ErrNotFound) {
			writeError(c, http.StatusNotFound, "MODEL_NOT_FOUND", "model config was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "load model before delete failed")
		return
	}

	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "delete model failed")
		return
	}

	middleware.AppendAuditEntry(c, middleware.AuditEntry{
		Action: "model.delete",
		Target: fmt.Sprintf("model:%d", id),
		Details: map[string]any{
			"resourceId": id,
			"before": map[string]any{
				"name":      before.Name,
				"provider":  before.Provider,
				"model":     before.Model,
				"baseUrl":   before.BaseURL,
				"isDefault": before.IsDefault,
				"isEnabled": before.IsEnabled,
			},
		},
	})

	writeSuccess(c, http.StatusOK, "model deleted", gin.H{"id": id})
}

func (h *ModelHandler) Test(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	result, err := h.service.TestConnection(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, applicationmodel.ErrNotFound) {
			writeError(c, http.StatusNotFound, "MODEL_NOT_FOUND", "model config was not found")
			return
		}
		writeError(c, http.StatusBadGateway, "MODEL_TEST_FAILED", err.Error())
		return
	}

	writeSuccess(c, http.StatusOK, "model connectivity verified", result)
}

func (h *ModelHandler) SetDefault(c *gin.Context) {
	id, ok := parseIDParam(c, "id")
	if !ok {
		return
	}

	item, err := h.service.SetDefault(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, applicationmodel.ErrNotFound) {
			writeError(c, http.StatusNotFound, "MODEL_NOT_FOUND", "model config was not found")
			return
		}
		writeError(c, http.StatusInternalServerError, "INTERNAL_ERROR", "set default model failed")
		return
	}

	middleware.AppendAuditEntry(c, middleware.AuditEntry{
		Action: "model.set_default",
		Target: fmt.Sprintf("model:%d", item.ID),
		Details: map[string]any{
			"resourceId": item.ID,
			"name":       item.Name,
			"provider":   item.Provider,
			"model":      item.Model,
		},
	})

	writeSuccess(c, http.StatusOK, "default model updated", item)
}
