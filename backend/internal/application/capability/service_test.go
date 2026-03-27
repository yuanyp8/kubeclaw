package capability

import (
	"context"
	"encoding/json"
	"testing"

	applicationmcp "kubeclaw/backend/internal/application/mcp"
	appskill "kubeclaw/backend/internal/application/skill"
)

func TestListForAudienceExcludesWorkflowMCPFromHTTPSurface(t *testing.T) {
	service := NewService(
		stubSkillProvider{},
		stubMCPProvider{items: []applicationmcp.Record{
			{
				ID:          7,
				Name:        "k8s-ops",
				Description: "Scale deployments and inspect rollout status",
				Transport:   "http",
				IsEnabled:   true,
			},
		}},
	)

	items := service.ListForAudience(context.Background(), AudienceHTTP)
	for _, item := range items {
		if item.CapabilityType == "mcp" {
			t.Fatalf("did not expect mcp workflow capability on HTTP audience: %+v", item)
		}
	}
}

func TestSkillCapabilitiesDefaultToWorkflowAgentAudience(t *testing.T) {
	service := NewService(
		stubSkillProvider{items: []appskill.Record{
			{
				ID:          11,
				Name:        "deploy-guard",
				Description: "Review deployment rollout before scaling",
				Type:        "builtin",
				Status:      "active",
				Definition:  json.RawMessage(`{"executor":"builtin","kind":"scale_deployment","payload":{"replicas":3}}`),
			},
		}},
		stubMCPProvider{},
	)

	items := service.ListForAudience(context.Background(), AudienceAgent)
	var found *Descriptor
	for idx := range items {
		if items[idx].CapabilityType == "skill" {
			found = &items[idx]
			break
		}
	}
	if found == nil {
		t.Fatal("expected to find skill capability")
	}
	if found.Level != LevelWorkflow {
		t.Fatalf("expected workflow level, got %s", found.Level)
	}
	if len(found.Audiences) != 1 || found.Audiences[0] != AudienceAgent {
		t.Fatalf("expected agent-only audience, got %#v", found.Audiences)
	}
}

type stubSkillProvider struct {
	items []appskill.Record
}

func (s stubSkillProvider) List(context.Context) ([]appskill.Record, error) {
	return s.items, nil
}

type stubMCPProvider struct {
	items []applicationmcp.Record
}

func (s stubMCPProvider) List(context.Context) ([]applicationmcp.Record, error) {
	return s.items, nil
}
