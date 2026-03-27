package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	applicationagent "kubeclaw/backend/internal/application/agent"
	applicationcapability "kubeclaw/backend/internal/application/capability"
	applicationcluster "kubeclaw/backend/internal/application/cluster"
	applicationmcp "kubeclaw/backend/internal/application/mcp"
	applicationmodel "kubeclaw/backend/internal/application/model"
	domainuser "kubeclaw/backend/internal/domain/user"
	"kubeclaw/backend/internal/infrastructure/llm"

	"github.com/gin-gonic/gin"
)

func TestCapabilityHandlerListSupportsAudienceFiltering(t *testing.T) {
	t.Parallel()

	handler := NewCapabilityHandler(applicationcapability.NewService(
		nil,
		stubCapabilityMCPProvider{items: []applicationmcp.Record{
			{ID: 9, Name: "k8s-ops", Description: "Scale deployments", IsEnabled: true, Transport: "http"},
		}},
	), nil)

	router := gin.New()
	router.GET("/capabilities", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/capabilities?audience=http", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	var payload struct {
		Data []applicationcapability.Descriptor `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Data) == 0 {
		t.Fatal("expected capabilities in response")
	}
	for _, item := range payload.Data {
		if item.CapabilityType == "mcp" {
			t.Fatalf("expected http capability list to exclude workflow mcp capabilities, got %+v", item)
		}
	}
	foundBuiltin := false
	for _, item := range payload.Data {
		if item.CapabilityType == "builtin" {
			foundBuiltin = true
			break
		}
	}
	if !foundBuiltin {
		t.Fatalf("expected builtin capabilities in response, got %+v", payload.Data)
	}
}

func TestCapabilityHandlerRejectsUnknownAudience(t *testing.T) {
	t.Parallel()

	handler := NewCapabilityHandler(applicationcapability.NewService(nil, nil), nil)
	router := gin.New()
	router.GET("/capabilities", handler.List)

	req := httptest.NewRequest(http.MethodGet, "/capabilities?audience=unknown", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rec.Code)
	}
}

func TestCapabilityHandlerInvokeBuiltinHTTPCapability(t *testing.T) {
	t.Parallel()

	service := applicationcapability.NewService(nil, nil).WithRuntime(
		stubCapabilityModelService{},
		stubCapabilityClusterService{
			listResourcesResult: []applicationcluster.ResourceRecord{
				{Name: "pod-a", Namespace: "default", Type: "pods"},
			},
		},
		stubCapabilityLLM{},
	)
	handler := NewCapabilityHandler(service, nil)
	router := gin.New()
	router.POST("/capabilities/:ref/invoke", handler.Invoke)

	body := `{"clusterId":1,"namespace":"default","payload":{"type":"pods"}}`
	req := httptest.NewRequest(http.MethodPost, "/capabilities/builtin.cluster.resources/invoke", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "pod-a") {
		t.Fatalf("expected pod result in response, got %s", rec.Body.String())
	}
}

func TestCapabilityHandlerInvokeRejectsAgentOnlyCapabilityOverHTTP(t *testing.T) {
	t.Parallel()

	service := applicationcapability.NewService(nil, nil).WithRuntime(
		stubCapabilityModelService{},
		stubCapabilityClusterService{},
		stubCapabilityLLM{},
	)
	handler := NewCapabilityHandler(service, nil)
	router := gin.New()
	router.POST("/capabilities/:ref/invoke", handler.Invoke)

	body := `{"clusterId":1,"namespace":"default","payload":{"name":"demo","replicas":2}}`
	req := httptest.NewRequest(http.MethodPost, "/capabilities/builtin.cluster.scale/invoke", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rec.Code, rec.Body.String())
	}
}

func TestCapabilityHandlerRequestRoutesMutationToAgentApprovalFlow(t *testing.T) {
	t.Parallel()

	requester := &stubCapabilityRequester{
		result: &applicationagent.SendMessageResult{
			SessionID:     21,
			UserMessageID: 22,
			RunID:         23,
			Status:        "queued",
		},
	}
	service := applicationcapability.NewService(nil, nil).WithRuntime(
		stubCapabilityModelService{},
		stubCapabilityClusterService{},
		stubCapabilityLLM{},
	)
	handler := NewCapabilityHandler(service, requester)
	router := gin.New()
	router.POST("/capabilities/:ref/request", withCurrentUser(&domainuser.User{ID: 88, Role: domainuser.RoleClusterAdmin}), handler.Request)

	body := `{"clusterId":1,"namespace":"default","payload":{"name":"payments","replicas":5}}`
	req := httptest.NewRequest(http.MethodPost, "/capabilities/builtin.cluster.scale/request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status 202, got %d body=%s", rec.Code, rec.Body.String())
	}
	if requester.lastInput.Action != "scale_deployment" {
		t.Fatalf("expected scale_deployment action, got %s", requester.lastInput.Action)
	}
	if requester.lastInput.Replicas != 5 {
		t.Fatalf("expected replicas 5, got %d", requester.lastInput.Replicas)
	}
	if requester.lastInput.ResourceName != "payments" {
		t.Fatalf("expected resource name payments, got %s", requester.lastInput.ResourceName)
	}
}

type stubCapabilityMCPProvider struct {
	items []applicationmcp.Record
}

func (s stubCapabilityMCPProvider) List(context.Context) ([]applicationmcp.Record, error) {
	return s.items, nil
}

type stubCapabilityRequester struct {
	lastInput applicationagent.ClusterActionRequestInput
	result    *applicationagent.SendMessageResult
	err       error
}

func (s *stubCapabilityRequester) RequestClusterAction(_ context.Context, input applicationagent.ClusterActionRequestInput) (*applicationagent.SendMessageResult, error) {
	s.lastInput = input
	return s.result, s.err
}

type stubCapabilityClusterService struct {
	listResourcesResult []applicationcluster.ResourceRecord
}

func (s stubCapabilityClusterService) ListNamespaces(context.Context, int64) ([]applicationcluster.NamespaceRecord, error) {
	return nil, nil
}

func (s stubCapabilityClusterService) ListResources(context.Context, int64, applicationcluster.ResourceQuery) ([]applicationcluster.ResourceRecord, error) {
	return s.listResourcesResult, nil
}

func (s stubCapabilityClusterService) ListEvents(context.Context, int64, string) ([]applicationcluster.EventRecord, error) {
	return nil, nil
}

func (s stubCapabilityClusterService) DeleteResource(context.Context, int64, applicationcluster.ResourceQuery, string) error {
	return nil
}

func (s stubCapabilityClusterService) ScaleDeployment(context.Context, int64, string, string, int32) error {
	return nil
}

func (s stubCapabilityClusterService) RestartDeployment(context.Context, int64, string, string) error {
	return nil
}

func (s stubCapabilityClusterService) ApplyYAML(context.Context, int64, string) (*applicationcluster.ApplyResult, error) {
	return &applicationcluster.ApplyResult{Summary: "ok"}, nil
}

type stubCapabilityModelService struct{}

func (stubCapabilityModelService) List(context.Context) ([]applicationmodel.Record, error) {
	return nil, nil
}

func (stubCapabilityModelService) Resolve(context.Context, int64) (*applicationmodel.ResolvedRecord, error) {
	return &applicationmodel.ResolvedRecord{}, nil
}

func (stubCapabilityModelService) ResolveDefault(context.Context) (*applicationmodel.ResolvedRecord, error) {
	return &applicationmodel.ResolvedRecord{}, nil
}

type stubCapabilityLLM struct{}

func (stubCapabilityLLM) Chat(context.Context, llm.ChatInput) (*llm.ChatResult, error) {
	return &llm.ChatResult{Content: "ok"}, nil
}
