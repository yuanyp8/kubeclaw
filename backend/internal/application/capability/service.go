package capability

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	applicationmcp "kubeclaw/backend/internal/application/mcp"
	appskill "kubeclaw/backend/internal/application/skill"
)

type Audience string

const (
	AudienceAgent Audience = "agent"
	AudienceHTTP  Audience = "http"
	AudienceMCP   Audience = "mcp"
)

type Level string

const (
	LevelPrimitive Level = "primitive"
	LevelWorkflow  Level = "workflow"
	LevelCatalog   Level = "catalog"
)

type Descriptor struct {
	Reference        string
	CapabilityType   string
	ID               int64
	Name             string
	Tool             string
	Role             string
	Summary          string
	Actions          []string
	ResourceTypes    []string
	Keywords         []string
	PreferredWeight  int
	Executor         string
	Transport        string
	DefaultPayload   map[string]any
	TargetAction     string
	TargetMCPID      int64
	TargetMCPName    string
	SystemPrompt     string
	InputTemplate    string
	Level            Level
	Audiences        []Audience
	Mutation         bool
	RequiresApproval bool
	RequestMode      string
	MCP              *applicationmcp.Record
	Skill            *appskill.Record
}

type SkillProvider interface {
	List(ctx context.Context) ([]appskill.Record, error)
}

type MCPProvider interface {
	List(ctx context.Context) ([]applicationmcp.Record, error)
}

type Service struct {
	skills   SkillProvider
	mcp      MCPProvider
	models   ModelService
	clusters ClusterService
	llm      LLMClient
}

func NewService(skills SkillProvider, mcp MCPProvider) *Service {
	return &Service{
		skills: skills,
		mcp:    mcp,
	}
}

func (s *Service) List(ctx context.Context) []Descriptor {
	items := builtinCapabilities()

	if s != nil && s.skills != nil {
		if records, err := s.skills.List(ctx); err == nil {
			for _, record := range records {
				if item, ok := buildSkillCapability(record); ok {
					items = append(items, item)
				}
			}
		}
	}

	if s != nil && s.mcp != nil {
		if records, err := s.mcp.List(ctx); err == nil {
			for _, record := range records {
				if item, ok := buildMCPCapability(record); ok {
					items = append(items, item)
				}
			}
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].CapabilityType == items[j].CapabilityType {
			return items[i].Name < items[j].Name
		}
		return items[i].CapabilityType < items[j].CapabilityType
	})

	return items
}

func (s *Service) ListForAudience(ctx context.Context, audience Audience) []Descriptor {
	items := s.List(ctx)
	filtered := make([]Descriptor, 0, len(items))
	for _, item := range items {
		if supportsAudience(item, audience) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func PlannerCatalog(items []Descriptor) []map[string]any {
	catalog := make([]map[string]any, 0, len(items))
	for _, item := range items {
		catalog = append(catalog, map[string]any{
			"type":             item.CapabilityType,
			"id":               item.ID,
			"reference":        item.Reference,
			"name":             item.Name,
			"tool":             item.Tool,
			"summary":          item.Summary,
			"actions":          item.Actions,
			"resourceTypes":    item.ResourceTypes,
			"level":            item.Level,
			"audiences":        item.Audiences,
			"mutation":         item.Mutation,
			"requiresApproval": item.RequiresApproval,
			"requestMode":      item.RequestMode,
		})
	}
	return catalog
}

func builtinCapabilities() []Descriptor {
	return []Descriptor{
		{
			Reference:       "builtin.cluster.namespaces",
			CapabilityType:  "builtin",
			Name:            "builtin.cluster.namespaces",
			Tool:            "cluster.list_namespaces",
			Role:            "k8s_expert",
			Summary:         "Primitive Kubernetes namespace reader.",
			Actions:         []string{"list_namespaces"},
			Keywords:        []string{"namespace", "namespaces"},
			PreferredWeight: 30,
			Executor:        "builtin",
			Level:           LevelPrimitive,
			Audiences:       []Audience{AudienceAgent, AudienceHTTP},
		},
		{
			Reference:       "builtin.cluster.resources",
			CapabilityType:  "builtin",
			Name:            "builtin.cluster.resources",
			Tool:            "cluster.list_resources",
			Role:            "k8s_expert",
			Summary:         "Primitive Kubernetes resource reader for pods, deployments, services, configmaps, and secrets.",
			Actions:         []string{"list_resources"},
			ResourceTypes:   []string{"pods", "deployments", "services", "configmaps", "secrets"},
			Keywords:        []string{"k8s", "kubernetes", "cluster", "pod", "pods", "deployment", "deployments", "service", "services", "configmap", "secret"},
			PreferredWeight: 30,
			Executor:        "builtin",
			Level:           LevelPrimitive,
			Audiences:       []Audience{AudienceAgent, AudienceHTTP},
		},
		{
			Reference:       "builtin.cluster.events",
			CapabilityType:  "builtin",
			Name:            "builtin.cluster.events",
			Tool:            "cluster.list_events",
			Role:            "k8s_expert",
			Summary:         "Primitive Kubernetes event reader.",
			Actions:         []string{"list_events"},
			Keywords:        []string{"event", "events"},
			PreferredWeight: 30,
			Executor:        "builtin",
			Level:           LevelPrimitive,
			Audiences:       []Audience{AudienceAgent, AudienceHTTP},
		},
		{
			Reference:        "builtin.cluster.delete",
			CapabilityType:   "builtin",
			Name:             "builtin.cluster.delete",
			Tool:             "cluster.delete_resource",
			Role:             "k8s_expert",
			Summary:          "Primitive Kubernetes resource deletion capability.",
			Actions:          []string{"delete_resource"},
			ResourceTypes:    []string{"pods", "deployments", "services", "configmaps", "secrets"},
			Keywords:         []string{"delete", "remove"},
			PreferredWeight:  30,
			Executor:         "builtin",
			Level:            LevelPrimitive,
			Audiences:        []Audience{AudienceAgent},
			Mutation:         true,
			RequiresApproval: true,
			RequestMode:      "agent_approval",
		},
		{
			Reference:        "builtin.cluster.scale",
			CapabilityType:   "builtin",
			Name:             "builtin.cluster.scale",
			Tool:             "cluster.scale_deployment",
			Role:             "k8s_expert",
			Summary:          "Primitive deployment scaling capability.",
			Actions:          []string{"scale_deployment"},
			ResourceTypes:    []string{"deployments"},
			Keywords:         []string{"scale", "replicas"},
			PreferredWeight:  30,
			Executor:         "builtin",
			Level:            LevelPrimitive,
			Audiences:        []Audience{AudienceAgent},
			Mutation:         true,
			RequiresApproval: true,
			RequestMode:      "agent_approval",
		},
		{
			Reference:        "builtin.cluster.restart",
			CapabilityType:   "builtin",
			Name:             "builtin.cluster.restart",
			Tool:             "cluster.restart_deployment",
			Role:             "k8s_expert",
			Summary:          "Primitive deployment restart capability.",
			Actions:          []string{"restart_deployment"},
			ResourceTypes:    []string{"deployments"},
			Keywords:         []string{"restart", "rollout"},
			PreferredWeight:  30,
			Executor:         "builtin",
			Level:            LevelPrimitive,
			Audiences:        []Audience{AudienceAgent},
			Mutation:         true,
			RequiresApproval: true,
			RequestMode:      "agent_approval",
		},
		{
			Reference:        "builtin.cluster.apply",
			CapabilityType:   "builtin",
			Name:             "builtin.cluster.apply",
			Tool:             "cluster.apply_yaml",
			Role:             "k8s_expert",
			Summary:          "Primitive YAML apply capability.",
			Actions:          []string{"apply_yaml"},
			Keywords:         []string{"apply", "manifest", "yaml", "kubectl"},
			PreferredWeight:  30,
			Executor:         "builtin",
			Level:            LevelPrimitive,
			Audiences:        []Audience{AudienceAgent},
			Mutation:         true,
			RequiresApproval: true,
			RequestMode:      "agent_approval",
		},
		{
			Reference:       "builtin.models.list",
			CapabilityType:  "builtin",
			Name:            "builtin.models.list",
			Tool:            "model.list",
			Role:            "skill_mcp_expert",
			Summary:         "Catalog capability for model inventory.",
			Actions:         []string{"list_models"},
			Keywords:        []string{"model", "models"},
			PreferredWeight: 20,
			Executor:        "builtin",
			Level:           LevelCatalog,
			Audiences:       []Audience{AudienceAgent, AudienceHTTP},
		},
		{
			Reference:       "builtin.skills.list",
			CapabilityType:  "builtin",
			Name:            "builtin.skills.list",
			Tool:            "skill.list",
			Role:            "skill_mcp_expert",
			Summary:         "Catalog capability for skill inventory.",
			Actions:         []string{"list_skills"},
			Keywords:        []string{"skill", "skills"},
			PreferredWeight: 20,
			Executor:        "builtin",
			Level:           LevelCatalog,
			Audiences:       []Audience{AudienceAgent, AudienceHTTP},
		},
		{
			Reference:       "builtin.mcp.list",
			CapabilityType:  "builtin",
			Name:            "builtin.mcp.list",
			Tool:            "mcp.list",
			Role:            "skill_mcp_expert",
			Summary:         "Catalog capability for configured MCP inventory.",
			Actions:         []string{"list_mcp"},
			Keywords:        []string{"mcp"},
			PreferredWeight: 20,
			Executor:        "builtin",
			Level:           LevelCatalog,
			Audiences:       []Audience{AudienceAgent, AudienceHTTP},
		},
	}
}

func buildSkillCapability(record appskill.Record) (Descriptor, bool) {
	if !isSkillRunnable(record.Status) {
		return Descriptor{}, false
	}

	definition := parseJSONMap(record.Definition)
	executor := strings.ToLower(firstNonEmpty(
		stringFromPath(definition, "executor"),
		stringFromPath(definition, "runtime.executor"),
		stringFromPath(definition, "runtime.type"),
		record.Type,
	))

	defaultPayload := firstMap(
		mapFromPath(definition, "payload"),
		mapFromPath(definition, "defaults"),
		mapFromPath(definition, "routing.payload"),
	)
	targetAction := normalizeAction(firstNonEmpty(
		stringFromPath(definition, "targetAction"),
		stringFromPath(definition, "builtinKind"),
		stringFromPath(definition, "action"),
		stringFromPath(definition, "kind"),
		stringFromPath(definition, "routing.action"),
	))
	actions := normalizeActionList(append(
		stringListFromPath(definition, "actions"),
		stringListFromPath(definition, "kinds")...,
	))
	actions = normalizeActionList(append(actions, stringListFromPath(definition, "routing.actions")...))
	actions = normalizeActionList(append(actions, targetAction))

	resourceTypes := normalizeResourceTypes(append(
		stringListFromPath(definition, "resourceTypes"),
		stringListFromPath(definition, "routing.resourceTypes")...,
	))
	keywords := uniqueLower(append(
		tokenizeCapabilityText(record.Name+" "+record.Description),
		stringListFromPath(definition, "keywords")...,
	))
	keywords = uniqueLower(append(keywords, stringListFromPath(definition, "routing.keywords")...))

	targetMCPID := int64(numberField(valueFromPath(definition, "mcpId")))
	if targetMCPID == 0 {
		targetMCPID = int64(numberField(valueFromPath(definition, "target.mcpId")))
	}
	targetMCPName := strings.TrimSpace(firstNonEmpty(
		stringFromPath(definition, "mcpName"),
		stringFromPath(definition, "target.mcpName"),
	))
	systemPrompt := strings.TrimSpace(firstNonEmpty(
		stringFromPath(definition, "systemPrompt"),
		stringFromPath(definition, "prompt"),
		stringFromPath(definition, "instructions"),
		record.Description,
	))
	inputTemplate := strings.TrimSpace(firstNonEmpty(
		stringFromPath(definition, "inputTemplate"),
		stringFromPath(definition, "promptTemplate"),
		stringFromPath(definition, "template"),
	))

	if executor == "" {
		switch {
		case targetMCPID > 0 || targetMCPName != "" || strings.Contains(strings.ToLower(record.Type), "mcp"):
			executor = "mcp"
		case systemPrompt != "" || strings.Contains(strings.ToLower(record.Type), "prompt") || strings.Contains(strings.ToLower(record.Type), "llm"):
			executor = "llm"
		default:
			executor = "builtin"
		}
	}

	if len(actions) == 0 {
		actions = inferActionKinds(record.Name + " " + record.Description)
	}
	if len(actions) == 0 && executor == "llm" {
		actions = []string{"llm"}
	}
	if len(resourceTypes) == 0 {
		resourceTypes = inferResourceTypes(record.Name + " " + record.Description)
	}
	if len(keywords) == 0 {
		keywords = tokenizeCapabilityText(record.Name + " " + record.Description)
	}

	toolName := fmt.Sprintf("skill.%s", sanitizeToolSegment(record.Name))
	if targetAction != "" && targetAction != "llm" {
		toolName = fmt.Sprintf("%s.%s", toolName, targetAction)
	}

	return Descriptor{
		Reference:       fmt.Sprintf("skill:%d", record.ID),
		CapabilityType:  "skill",
		ID:              record.ID,
		Name:            record.Name,
		Tool:            toolName,
		Role:            "skill_worker",
		Summary:         buildCapabilitySummary("Skill", record.Name, record.Description, actions, resourceTypes),
		Actions:         actions,
		ResourceTypes:   resourceTypes,
		Keywords:        keywords,
		PreferredWeight: 60,
		Executor:        executor,
		DefaultPayload:  defaultPayload,
		TargetAction:    targetAction,
		TargetMCPID:     targetMCPID,
		TargetMCPName:   targetMCPName,
		SystemPrompt:    systemPrompt,
		InputTemplate:   inputTemplate,
		Level:           LevelWorkflow,
		Audiences:       []Audience{AudienceAgent},
		Skill:           &record,
	}, true
}

func buildMCPCapability(record applicationmcp.Record) (Descriptor, bool) {
	if !record.IsEnabled {
		return Descriptor{}, false
	}

	fullText := strings.TrimSpace(strings.Join([]string{record.Name, record.Type, record.Description, record.Endpoint, record.Command}, " "))
	actions := inferActionKinds(fullText)
	resourceTypes := inferResourceTypes(fullText)
	keywords := uniqueLower(append(tokenizeCapabilityText(fullText), "mcp"))

	summary := buildCapabilitySummary(
		"MCP workflow",
		record.Name,
		firstNonEmpty(record.Description, "Higher-level orchestration capability backed by an external MCP service. Not a one-to-one mirror of every HTTP API."),
		actions,
		resourceTypes,
	)

	return Descriptor{
		Reference:       fmt.Sprintf("mcp:%d", record.ID),
		CapabilityType:  "mcp",
		ID:              record.ID,
		Name:            record.Name,
		Tool:            fmt.Sprintf("mcp.%s", sanitizeToolSegment(record.Name)),
		Role:            "mcp_worker",
		Summary:         summary,
		Actions:         actions,
		ResourceTypes:   resourceTypes,
		Keywords:        keywords,
		PreferredWeight: 50,
		Executor:        "mcp",
		Transport:       strings.ToLower(strings.TrimSpace(record.Transport)),
		Level:           LevelWorkflow,
		Audiences:       []Audience{AudienceAgent},
		MCP:             &record,
	}, true
}

func supportsAudience(item Descriptor, audience Audience) bool {
	for _, candidate := range item.Audiences {
		if candidate == audience {
			return true
		}
	}
	return false
}

func isSkillRunnable(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "", "draft", "active", "published", "ready":
		return true
	case "disabled", "archived":
		return false
	default:
		return true
	}
}

func buildCapabilitySummary(prefix string, name string, description string, actions []string, resourceTypes []string) string {
	parts := []string{fmt.Sprintf("%s %s", prefix, name)}
	if strings.TrimSpace(description) != "" {
		parts = append(parts, strings.TrimSpace(description))
	}
	if len(actions) > 0 {
		parts = append(parts, fmt.Sprintf("actions=%s", strings.Join(actions, ",")))
	}
	if len(resourceTypes) > 0 {
		parts = append(parts, fmt.Sprintf("resourceTypes=%s", strings.Join(resourceTypes, ",")))
	}
	return strings.Join(parts, ". ")
}

func inferActionKinds(text string) []string {
	lower := strings.ToLower(text)
	actions := make([]string, 0, 6)

	if strings.Contains(lower, "namespace") {
		actions = append(actions, "list_namespaces")
	}
	if strings.Contains(lower, "event") {
		actions = append(actions, "list_events")
	}
	if matchesAny(lower, []string{"pod", "pods", "deployment", "deployments", "service", "services", "configmap", "secret", "k8s", "kubernetes", "cluster"}) {
		actions = append(actions, "list_resources")
	}
	if strings.Contains(lower, "delete") || strings.Contains(lower, "remove") {
		actions = append(actions, "delete_resource")
	}
	if strings.Contains(lower, "scale") || strings.Contains(lower, "replica") {
		actions = append(actions, "scale_deployment")
	}
	if strings.Contains(lower, "restart") || strings.Contains(lower, "rollout") {
		actions = append(actions, "restart_deployment")
	}
	if strings.Contains(lower, "apply") || strings.Contains(lower, "manifest") || strings.Contains(lower, "yaml") {
		actions = append(actions, "apply_yaml")
	}
	return normalizeActionList(actions)
}

func inferResourceTypes(text string) []string {
	lower := strings.ToLower(text)
	resourceTypes := make([]string, 0, 5)
	if strings.Contains(lower, "pod") {
		resourceTypes = append(resourceTypes, "pods")
	}
	if strings.Contains(lower, "deployment") {
		resourceTypes = append(resourceTypes, "deployments")
	}
	if strings.Contains(lower, "service") {
		resourceTypes = append(resourceTypes, "services")
	}
	if strings.Contains(lower, "configmap") {
		resourceTypes = append(resourceTypes, "configmaps")
	}
	if strings.Contains(lower, "secret") {
		resourceTypes = append(resourceTypes, "secrets")
	}
	return normalizeResourceTypes(resourceTypes)
}

func normalizeActionList(values []string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		normalized := normalizeAction(value)
		if normalized != "" {
			items = append(items, normalized)
		}
	}
	return uniqueStrings(items)
}

func normalizeResourceTypes(values []string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		normalized := normalizeResourceType(value)
		if normalized != "" {
			items = append(items, normalized)
		}
	}
	return uniqueStrings(items)
}

func normalizeAction(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "list_namespaces", "namespace", "namespaces":
		return "list_namespaces"
	case "list_resources", "list_resource", "resource", "resources":
		return "list_resources"
	case "list_events", "events", "event":
		return "list_events"
	case "list_models", "models", "model":
		return "list_models"
	case "list_skills", "skills", "skill":
		return "list_skills"
	case "list_mcp", "mcp":
		return "list_mcp"
	case "delete_resource", "delete":
		return "delete_resource"
	case "scale_deployment", "scale":
		return "scale_deployment"
	case "restart_deployment", "restart":
		return "restart_deployment"
	case "apply_yaml", "apply":
		return "apply_yaml"
	case "llm", "chat", "answer":
		return "llm"
	default:
		return ""
	}
}

func normalizeResourceType(resourceType string) string {
	switch strings.ToLower(strings.TrimSpace(resourceType)) {
	case "pod", "pods":
		return "pods"
	case "deployment", "deployments":
		return "deployments"
	case "service", "services":
		return "services"
	case "configmap", "configmaps":
		return "configmaps"
	case "secret", "secrets":
		return "secrets"
	default:
		return ""
	}
}

func parseJSONMap(raw json.RawMessage) map[string]any {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil
	}
	return parsed
}

func valueFromPath(data map[string]any, path string) any {
	if len(data) == 0 {
		return nil
	}
	current := any(data)
	for _, segment := range strings.Split(path, ".") {
		typed, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		current, ok = typed[segment]
		if !ok {
			return nil
		}
	}
	return current
}

func stringFromPath(data map[string]any, path string) string {
	return stringField(valueFromPath(data, path))
}

func mapFromPath(data map[string]any, path string) map[string]any {
	typed, _ := valueFromPath(data, path).(map[string]any)
	return typed
}

func stringListFromPath(data map[string]any, path string) []string {
	value := valueFromPath(data, path)
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			text := strings.TrimSpace(stringField(item))
			if text != "" {
				result = append(result, text)
			}
		}
		return result
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil
		}
		return []string{typed}
	default:
		return nil
	}
}

func firstMap(values ...map[string]any) map[string]any {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}
	return nil
}

func mergePayloads(base map[string]any, override map[string]any) map[string]any {
	result := make(map[string]any, len(base)+len(override))
	for key, value := range base {
		result[key] = value
	}
	for key, value := range override {
		result[key] = value
	}
	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func uniqueLower(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value != "" {
			normalized = append(normalized, value)
		}
	}
	return uniqueStrings(normalized)
}

func tokenizeCapabilityText(text string) []string {
	lower := strings.ToLower(text)
	re := regexp.MustCompile(`[a-z0-9_\-]+`)
	return re.FindAllString(lower, -1)
}

func matchesAny(text string, patterns []string) bool {
	for _, pattern := range patterns {
		if strings.Contains(text, pattern) {
			return true
		}
	}
	return false
}

func sanitizeToolSegment(value string) string {
	lower := strings.ToLower(strings.TrimSpace(value))
	if lower == "" {
		return "unknown"
	}
	re := regexp.MustCompile(`[^a-z0-9]+`)
	segment := strings.Trim(re.ReplaceAllString(lower, "_"), "_")
	if segment == "" {
		return "unknown"
	}
	return segment
}

func stringField(value any) string {
	text, _ := value.(string)
	return text
}

func numberField(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}
