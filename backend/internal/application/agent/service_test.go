package agent

import (
	"context"
	"encoding/json"
	"testing"

	applicationchat "kubeclaw/backend/internal/application/chat"
	applicationcluster "kubeclaw/backend/internal/application/cluster"
	applicationmcp "kubeclaw/backend/internal/application/mcp"
	applicationmodel "kubeclaw/backend/internal/application/model"
	applicationsecurity "kubeclaw/backend/internal/application/security"
	appskill "kubeclaw/backend/internal/application/skill"
)

func TestAnalyzeIntentPrefersConfiguredK8sMCP(t *testing.T) {
	clusterID := int64(9)
	service := &Service{
		models:   stubModelService{resolveErr: applicationmodel.ErrNotFound, resolveDefaultErr: applicationmodel.ErrNotFound},
		mcp:      stubMCPService{items: []applicationmcp.Record{{ID: 7, Name: "k8s-mcp", Description: "Kubernetes pods deployments services", IsEnabled: true, Transport: "http"}}},
		skills:   stubSkillService{},
		security: stubSecurityService{},
	}

	selected := service.analyzeIntent(context.Background(), applicationchat.Session{
		Context: applicationchat.SessionContext{ClusterID: &clusterID, Namespace: "default"},
	}, "use k8s-mcp to list pods in namespace default")

	if selected.Kind != "list_resources" {
		t.Fatalf("expected list_resources, got %s", selected.Kind)
	}
	if selected.CapabilityType != "mcp" {
		t.Fatalf("expected mcp capability, got %s", selected.CapabilityType)
	}
	if selected.CapabilityName != "k8s-mcp" {
		t.Fatalf("expected k8s-mcp capability, got %s", selected.CapabilityName)
	}
	if got := stringField(selected.Payload["type"]); got != "pods" {
		t.Fatalf("expected pods resource type, got %s", got)
	}
}

func TestRunSkillBuiltinIntentUsesBuiltinExecutor(t *testing.T) {
	clusterID := int64(3)
	service := &Service{
		skills: stubSkillService{items: []appskill.Record{
			{
				ID:          11,
				Name:        "pod-skill",
				Type:        "builtin",
				Status:      "draft",
				Definition:  json.RawMessage(`{"executor":"builtin","kind":"list_resources","payload":{"type":"pods"}}`),
				Description: "List Kubernetes pods",
			},
		}},
		mcp: stubMCPService{},
		clusters: stubClusterService{
			listResourcesResult: []applicationcluster.ResourceRecord{
				{Name: "pod-a", Namespace: "default", Type: "pods"},
			},
		},
	}

	result, err := service.runCapabilityIntent(context.Background(), applicationchat.Session{
		Context: applicationchat.SessionContext{ClusterID: &clusterID, Namespace: "default"},
	}, applicationchat.Message{Content: "show pods"}, intent{
		Kind:           "list_resources",
		Tool:           "skill.pod-skill.list_resources",
		CapabilityType: "skill",
		CapabilityID:   11,
		CapabilityName: "pod-skill",
		Payload:        map[string]any{},
	})
	if err != nil {
		t.Fatalf("expected skill execution to succeed: %v", err)
	}
	if result == "" {
		t.Fatal("expected a rendered result")
	}
}

type stubModelService struct {
	resolveErr        error
	resolveDefaultErr error
}

func (s stubModelService) List(context.Context) ([]applicationmodel.Record, error) {
	return nil, nil
}

func (s stubModelService) Resolve(context.Context, int64) (*applicationmodel.ResolvedRecord, error) {
	return nil, s.resolveErr
}

func (s stubModelService) ResolveDefault(context.Context) (*applicationmodel.ResolvedRecord, error) {
	return nil, s.resolveDefaultErr
}

type stubSkillService struct {
	items []appskill.Record
}

func (s stubSkillService) List(context.Context) ([]appskill.Record, error) {
	return s.items, nil
}

type stubMCPService struct {
	items []applicationmcp.Record
}

func (s stubMCPService) List(context.Context) ([]applicationmcp.Record, error) {
	return s.items, nil
}

type stubSecurityService struct {
	items []applicationsecurity.SensitiveWordRecord
}

func (s stubSecurityService) ListSensitiveWords(context.Context) ([]applicationsecurity.SensitiveWordRecord, error) {
	return s.items, nil
}

type stubClusterService struct {
	listResourcesResult []applicationcluster.ResourceRecord
}

func (s stubClusterService) ListNamespaces(context.Context, int64) ([]applicationcluster.NamespaceRecord, error) {
	return nil, nil
}

func (s stubClusterService) ListResources(context.Context, int64, applicationcluster.ResourceQuery) ([]applicationcluster.ResourceRecord, error) {
	return s.listResourcesResult, nil
}

func (s stubClusterService) ListEvents(context.Context, int64, string) ([]applicationcluster.EventRecord, error) {
	return nil, nil
}

func (s stubClusterService) DeleteResource(context.Context, int64, applicationcluster.ResourceQuery, string) error {
	return nil
}

func (s stubClusterService) ScaleDeployment(context.Context, int64, string, string, int32) error {
	return nil
}

func (s stubClusterService) RestartDeployment(context.Context, int64, string, string) error {
	return nil
}

func (s stubClusterService) ApplyYAML(context.Context, int64, string) (*applicationcluster.ApplyResult, error) {
	return &applicationcluster.ApplyResult{Summary: "ok"}, nil
}
