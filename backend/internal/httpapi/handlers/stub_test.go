package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"kubeclaw/backend/internal/httpapi/middleware"

	"github.com/gin-gonic/gin"
)

func TestStubHandlerReturnsConsistentNotImplementedPayload(t *testing.T) {
	t.Parallel()

	handler := NewStubHandler()
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(middleware.RequestIDKey, "req-test")
		c.Next()
	})
	router.POST("/skills/:id/execute", handler.Handle("skill", "execute"))

	req := httptest.NewRequest(http.MethodPost, "/skills/99/execute", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("expected status 501, got %d", rec.Code)
	}

	var payload struct {
		Code      string `json:"code"`
		RequestID string `json:"requestId"`
		Data      struct {
			Module string `json:"module"`
			Action string `json:"action"`
			Method string `json:"method"`
			Path   string `json:"path"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != "NOT_IMPLEMENTED" {
		t.Fatalf("expected NOT_IMPLEMENTED, got %s", payload.Code)
	}
	if payload.RequestID != "req-test" {
		t.Fatalf("expected request id req-test, got %s", payload.RequestID)
	}
	if payload.Data.Module != "skill" || payload.Data.Action != "execute" {
		t.Fatalf("unexpected data payload: %+v", payload.Data)
	}
}
