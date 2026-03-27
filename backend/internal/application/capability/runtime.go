package capability

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	applicationcluster "kubeclaw/backend/internal/application/cluster"
	applicationmcp "kubeclaw/backend/internal/application/mcp"
	applicationmodel "kubeclaw/backend/internal/application/model"
	"kubeclaw/backend/internal/infrastructure/llm"
)

var thinkBlockPattern = regexp.MustCompile(`(?is)<think>.*?</think>`)

type ModelService interface {
	List(ctx context.Context) ([]applicationmodel.Record, error)
	Resolve(ctx context.Context, id int64) (*applicationmodel.ResolvedRecord, error)
	ResolveDefault(ctx context.Context) (*applicationmodel.ResolvedRecord, error)
}

type ClusterService interface {
	ListNamespaces(ctx context.Context, clusterID int64) ([]applicationcluster.NamespaceRecord, error)
	ListResources(ctx context.Context, clusterID int64, query applicationcluster.ResourceQuery) ([]applicationcluster.ResourceRecord, error)
	ListEvents(ctx context.Context, clusterID int64, namespace string) ([]applicationcluster.EventRecord, error)
	DeleteResource(ctx context.Context, clusterID int64, query applicationcluster.ResourceQuery, name string) error
	ScaleDeployment(ctx context.Context, clusterID int64, namespace string, name string, replicas int32) error
	RestartDeployment(ctx context.Context, clusterID int64, namespace string, name string) error
	ApplyYAML(ctx context.Context, clusterID int64, manifest string) (*applicationcluster.ApplyResult, error)
}

type LLMClient interface {
	Chat(ctx context.Context, input llm.ChatInput) (*llm.ChatResult, error)
}

type InvokeContext struct {
	ClusterID *int64 `json:"clusterId"`
	Namespace string `json:"namespace"`
	ModelID   *int64 `json:"modelId"`
}

type Selection struct {
	Action         string         `json:"action"`
	Tool           string         `json:"tool"`
	CapabilityType string         `json:"capabilityType"`
	CapabilityID   int64          `json:"capabilityId"`
	CapabilityName string         `json:"capabilityName"`
	Payload        map[string]any `json:"payload"`
}

type InvokeInput struct {
	Audience  Audience
	Reference string
	Selection Selection
	Context   InvokeContext
	UserInput string
}

func (s *Service) WithRuntime(models ModelService, clusters ClusterService, llmClient LLMClient) *Service {
	s.models = models
	s.clusters = clusters
	s.llm = llmClient
	return s
}

func (s *Service) ResolveReference(ctx context.Context, reference string, audience Audience) (*Descriptor, error) {
	items := s.listForResolution(ctx, audience)
	needle := strings.TrimSpace(reference)
	for idx := range items {
		item := &items[idx]
		if strings.EqualFold(strings.TrimSpace(item.Reference), needle) ||
			strings.EqualFold(strings.TrimSpace(item.Name), needle) ||
			strings.EqualFold(strings.TrimSpace(item.Tool), needle) {
			return item, nil
		}
	}
	return nil, fmt.Errorf("capability %s not found", reference)
}

func (s *Service) ResolveSelection(ctx context.Context, selection Selection, audience Audience) (*Descriptor, error) {
	items := s.listForResolution(ctx, audience)
	for idx := range items {
		item := &items[idx]
		if selection.CapabilityType != "" && selection.CapabilityType != item.CapabilityType {
			continue
		}
		if selection.CapabilityID > 0 && selection.CapabilityID == item.ID {
			return item, nil
		}
		if selection.CapabilityName != "" && strings.EqualFold(strings.TrimSpace(selection.CapabilityName), strings.TrimSpace(item.Name)) {
			return item, nil
		}
	}

	for idx := range items {
		item := &items[idx]
		if item.Tool != "" && item.Tool == selection.Tool {
			return item, nil
		}
	}

	return nil, fmt.Errorf("capability not found for %s", firstNonEmpty(selection.CapabilityName, selection.Tool))
}

func (s *Service) Invoke(ctx context.Context, input InvokeInput) (string, error) {
	if s == nil {
		return "", fmt.Errorf("capability service is unavailable")
	}

	selection := input.Selection
	if strings.TrimSpace(input.Reference) != "" && selection.CapabilityName == "" && selection.CapabilityID == 0 && selection.Tool == "" {
		item, err := s.ResolveReference(ctx, input.Reference, input.Audience)
		if err != nil {
			return "", err
		}
		selection.CapabilityType = item.CapabilityType
		selection.CapabilityID = item.ID
		selection.CapabilityName = item.Name
		selection.Tool = item.Tool
	}

	item, err := s.ResolveSelection(ctx, selection, input.Audience)
	if err != nil {
		return "", err
	}

	action := normalizeAction(firstNonEmpty(selection.Action, item.TargetAction))
	if action == "" && len(item.Actions) == 1 {
		action = item.Actions[0]
	}
	if action == "" {
		return "", fmt.Errorf("capability %s requires an explicit action", item.Name)
	}

	payload := mergePayloads(item.DefaultPayload, selection.Payload)

	switch item.CapabilityType {
	case "skill":
		return s.invokeSkill(ctx, *item, action, payload, input)
	case "mcp":
		return s.invokeMCP(ctx, *item, action, payload, input)
	default:
		return s.invokeBuiltin(ctx, action, payload, input.Context)
	}
}

func (s *Service) listForResolution(ctx context.Context, audience Audience) []Descriptor {
	if audience == "" {
		return s.List(ctx)
	}
	return s.ListForAudience(ctx, audience)
}

func (s *Service) invokeBuiltin(ctx context.Context, action string, payload map[string]any, invokeContext InvokeContext) (string, error) {
	if s.clusters == nil && (action == "list_namespaces" || action == "list_resources" || action == "list_events" || action == "delete_resource" || action == "scale_deployment" || action == "restart_deployment" || action == "apply_yaml") {
		return "", fmt.Errorf("cluster runtime is unavailable")
	}
	if s.models == nil && action == "list_models" {
		return "", fmt.Errorf("model runtime is unavailable")
	}

	switch action {
	case "list_namespaces":
		if invokeContext.ClusterID == nil {
			return "", fmt.Errorf("clusterId is required")
		}
		items, err := s.clusters.ListNamespaces(ctx, *invokeContext.ClusterID)
		if err != nil {
			return "", err
		}
		return renderJSON(items)
	case "list_resources":
		if invokeContext.ClusterID == nil {
			return "", fmt.Errorf("clusterId is required")
		}
		items, err := s.clusters.ListResources(ctx, *invokeContext.ClusterID, applicationcluster.ResourceQuery{
			Type:      stringField(payload["type"]),
			Namespace: namespaceFromInvokeContext(invokeContext, payload),
		})
		if err != nil {
			return "", err
		}
		return renderJSON(items)
	case "list_events":
		if invokeContext.ClusterID == nil {
			return "", fmt.Errorf("clusterId is required")
		}
		items, err := s.clusters.ListEvents(ctx, *invokeContext.ClusterID, namespaceFromInvokeContext(invokeContext, payload))
		if err != nil {
			return "", err
		}
		return renderJSON(items)
	case "list_models":
		items, err := s.models.List(ctx)
		if err != nil {
			return "", err
		}
		return renderJSON(items)
	case "list_skills":
		items, err := s.skills.List(ctx)
		if err != nil {
			return "", err
		}
		return renderJSON(items)
	case "list_mcp":
		items, err := s.mcp.List(ctx)
		if err != nil {
			return "", err
		}
		return renderJSON(items)
	case "delete_resource":
		if invokeContext.ClusterID == nil {
			return "", fmt.Errorf("clusterId is required")
		}
		resourceType := normalizeResourceType(stringField(payload["type"]))
		name := strings.TrimSpace(stringField(payload["name"]))
		namespace := defaultString(namespaceFromInvokeContext(invokeContext, payload), "default")
		if name == "" {
			return "", fmt.Errorf("resource name is required")
		}
		if resourceType == "" {
			resourceType = inferResourceType(name)
		}
		if err := s.clusters.DeleteResource(ctx, *invokeContext.ClusterID, applicationcluster.ResourceQuery{
			Type:      resourceType,
			Namespace: namespace,
		}, name); err != nil {
			return "", err
		}
		return fmt.Sprintf("Deleted %s %s in namespace %s.", resourceType, name, namespace), nil
	case "scale_deployment":
		if invokeContext.ClusterID == nil {
			return "", fmt.Errorf("clusterId is required")
		}
		name := strings.TrimSpace(stringField(payload["name"]))
		namespace := defaultString(namespaceFromInvokeContext(invokeContext, payload), "default")
		replicas := int32(numberField(payload["replicas"]))
		if name == "" {
			return "", fmt.Errorf("deployment name is required")
		}
		if replicas <= 0 {
			return "", fmt.Errorf("replicas must be greater than zero")
		}
		if err := s.clusters.ScaleDeployment(ctx, *invokeContext.ClusterID, namespace, name, replicas); err != nil {
			return "", err
		}
		return fmt.Sprintf("Scaled deployment %s in namespace %s to %d replicas.", name, namespace, replicas), nil
	case "restart_deployment":
		if invokeContext.ClusterID == nil {
			return "", fmt.Errorf("clusterId is required")
		}
		name := strings.TrimSpace(stringField(payload["name"]))
		namespace := defaultString(namespaceFromInvokeContext(invokeContext, payload), "default")
		if name == "" {
			return "", fmt.Errorf("deployment name is required")
		}
		if err := s.clusters.RestartDeployment(ctx, *invokeContext.ClusterID, namespace, name); err != nil {
			return "", err
		}
		return fmt.Sprintf("Restarted deployment %s in namespace %s.", name, namespace), nil
	case "apply_yaml":
		if invokeContext.ClusterID == nil {
			return "", fmt.Errorf("clusterId is required")
		}
		manifest := strings.TrimSpace(stringField(payload["manifest"]))
		if manifest == "" {
			return "", fmt.Errorf("manifest is required")
		}
		result, err := s.clusters.ApplyYAML(ctx, *invokeContext.ClusterID, manifest)
		if err != nil {
			return "", err
		}
		if result == nil {
			return "Manifest applied.", nil
		}
		return defaultString(result.Summary, "Manifest applied."), nil
	default:
		return "", fmt.Errorf("unsupported capability action: %s", action)
	}
}

func (s *Service) invokeSkill(ctx context.Context, item Descriptor, action string, payload map[string]any, input InvokeInput) (string, error) {
	switch item.Executor {
	case "mcp":
		if item.TargetMCPID == 0 && strings.TrimSpace(item.TargetMCPName) == "" {
			return "", fmt.Errorf("skill %s is configured for mcp execution but does not declare a target mcp", item.Name)
		}
		nested := item
		nested.CapabilityType = "mcp"
		nested.ID = item.TargetMCPID
		nested.Name = item.TargetMCPName
		nested.Tool = fmt.Sprintf("mcp.%s.%s", sanitizeToolSegment(defaultString(item.TargetMCPName, item.Name)), action)
		nested.Executor = "mcp"
		return s.invokeMCP(ctx, nested, action, payload, input)
	case "llm":
		return s.invokeSkillLLM(ctx, item, payload, input)
	default:
		return s.invokeBuiltin(ctx, defaultString(item.TargetAction, action), payload, input.Context)
	}
}

func (s *Service) invokeSkillLLM(ctx context.Context, item Descriptor, payload map[string]any, input InvokeInput) (string, error) {
	if s.models == nil || s.llm == nil {
		return "", fmt.Errorf("llm runtime is unavailable")
	}

	resolvedModel, err := s.resolveModel(ctx, input.Context.ModelID)
	if err != nil {
		return "", err
	}

	systemPrompt := strings.TrimSpace(item.SystemPrompt)
	if systemPrompt == "" {
		systemPrompt = fmt.Sprintf("You are the %s skill in KubeClaw. Answer in the same language as the user and stay concise.", item.Name)
	}

	userContent := applyCapabilityTemplate(item.InputTemplate, input.UserInput, input.Context, payload)
	result, err := s.llm.Chat(ctx, llm.ChatInput{
		Model: *resolvedModel,
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
	})
	if err != nil {
		return "", fmt.Errorf("skill %s execution failed: %w", item.Name, err)
	}

	content, strippedThink := sanitizeLLMOutput(result.Content)
	if content == "" && strippedThink {
		return "", fmt.Errorf("skill %s returned reasoning only without a final answer", item.Name)
	}
	if content == "" {
		return "", fmt.Errorf("skill %s returned an empty answer", item.Name)
	}
	return content, nil
}

func (s *Service) invokeMCP(ctx context.Context, item Descriptor, action string, payload map[string]any, input InvokeInput) (string, error) {
	if item.MCP == nil {
		resolved, err := s.ResolveSelection(ctx, Selection{
			CapabilityType: "mcp",
			CapabilityID:   item.ID,
			CapabilityName: item.Name,
			Tool:           item.Tool,
		}, AudienceAgent)
		if err != nil {
			return "", err
		}
		item = *resolved
	}
	if item.MCP == nil {
		return "", fmt.Errorf("mcp capability %s is missing server configuration", item.Name)
	}

	requestBody := map[string]any{
		"action": action,
		"tool":   defaultString(item.Tool, defaultToolForAction(action)),
		"input":  payload,
		"session": map[string]any{
			"clusterId": input.Context.ClusterID,
			"namespace": input.Context.Namespace,
			"modelId":   input.Context.ModelID,
		},
		"userMessage": input.UserInput,
		"capability": map[string]any{
			"id":        item.ID,
			"name":      item.Name,
			"type":      item.CapabilityType,
			"reference": item.Reference,
			"level":     item.Level,
		},
	}

	switch strings.ToLower(defaultString(item.Transport, item.MCP.Transport)) {
	case "http", "https":
		return invokeHTTPMCP(ctx, *item.MCP, requestBody)
	case "command", "stdio", "process":
		return invokeCommandMCP(ctx, *item.MCP, requestBody)
	default:
		return "", fmt.Errorf("unsupported mcp transport: %s", defaultString(item.Transport, item.MCP.Transport))
	}
}

func (s *Service) resolveModel(ctx context.Context, modelID *int64) (*applicationmodel.ResolvedRecord, error) {
	if modelID != nil {
		return s.models.Resolve(ctx, *modelID)
	}
	return s.models.ResolveDefault(ctx)
}

func invokeHTTPMCP(ctx context.Context, record applicationmcp.Record, requestBody map[string]any) (string, error) {
	endpoint := strings.TrimSpace(record.Endpoint)
	if endpoint == "" {
		return "", fmt.Errorf("mcp %s is missing an endpoint", record.Name)
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal mcp request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create mcp request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for key, value := range record.Headers {
		if strings.TrimSpace(key) != "" {
			req.Header.Set(key, value)
		}
	}

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request mcp endpoint %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read mcp response: %w", err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("mcp endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return formatMCPResponse(body)
}

func invokeCommandMCP(ctx context.Context, record applicationmcp.Record, requestBody map[string]any) (string, error) {
	command := strings.TrimSpace(record.Command)
	if command == "" {
		return "", fmt.Errorf("mcp %s is missing a command", record.Name)
	}

	payload, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("marshal mcp command request: %w", err)
	}

	cmd := exec.CommandContext(ctx, command, record.Args...)
	cmd.Stdin = bytes.NewReader(payload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("run mcp command %s: %w: %s", command, err, strings.TrimSpace(string(output)))
	}

	return formatMCPResponse(output)
}

func formatMCPResponse(body []byte) (string, error) {
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		return "", fmt.Errorf("mcp returned an empty response")
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return trimmed, nil
	}

	switch typed := payload.(type) {
	case map[string]any:
		for _, key := range []string{"output", "result", "message"} {
			if value := strings.TrimSpace(stringField(typed[key])); value != "" {
				return value, nil
			}
		}
		if value, ok := typed["data"]; ok {
			return renderJSON(value)
		}
		return renderJSON(typed)
	default:
		return renderJSON(payload)
	}
}

func renderJSON(value any) (string, error) {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(payload), nil
}

func namespaceFromInvokeContext(invokeContext InvokeContext, payload map[string]any) string {
	if namespace := stringField(payload["namespace"]); namespace != "" {
		return namespace
	}
	return invokeContext.Namespace
}

func sanitizeLLMOutput(content string) (string, bool) {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return "", false
	}
	lower := strings.ToLower(trimmed)
	if !strings.Contains(lower, "<think>") {
		return trimmed, false
	}
	if !strings.Contains(lower, "</think>") {
		return "", true
	}
	trimmed = strings.TrimSpace(thinkBlockPattern.ReplaceAllString(trimmed, " "))
	return trimmed, true
}

func defaultToolForAction(action string) string {
	switch action {
	case "list_namespaces":
		return "cluster.list_namespaces"
	case "list_resources":
		return "cluster.list_resources"
	case "list_events":
		return "cluster.list_events"
	case "list_models":
		return "model.list"
	case "list_skills":
		return "skill.list"
	case "list_mcp":
		return "mcp.list"
	case "delete_resource":
		return "cluster.delete_resource"
	case "scale_deployment":
		return "cluster.scale_deployment"
	case "restart_deployment":
		return "cluster.restart_deployment"
	case "apply_yaml":
		return "cluster.apply_yaml"
	default:
		return "llm.chat"
	}
}

func applyCapabilityTemplate(template string, userInput string, invokeContext InvokeContext, payload map[string]any) string {
	if strings.TrimSpace(template) == "" {
		return userInput
	}

	payloadJSON, _ := json.Marshal(payload)
	clusterID := ""
	if invokeContext.ClusterID != nil {
		clusterID = fmt.Sprintf("%d", *invokeContext.ClusterID)
	}

	replacer := strings.NewReplacer(
		"{input}", userInput,
		"{userInput}", userInput,
		"{namespace}", invokeContext.Namespace,
		"{clusterId}", clusterID,
		"{payload}", string(payloadJSON),
	)
	return replacer.Replace(template)
}

func inferResourceType(text string) string {
	lower := strings.ToLower(text)
	switch {
	case strings.Contains(lower, "pod"):
		return "pods"
	case strings.Contains(lower, "service"):
		return "services"
	case strings.Contains(lower, "configmap"):
		return "configmaps"
	case strings.Contains(lower, "secret"):
		return "secrets"
	default:
		return "deployments"
	}
}
