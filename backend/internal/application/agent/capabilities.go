package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	applicationcapability "kubeclaw/backend/internal/application/capability"
	applicationchat "kubeclaw/backend/internal/application/chat"
	applicationcluster "kubeclaw/backend/internal/application/cluster"
	applicationmcp "kubeclaw/backend/internal/application/mcp"
	appskill "kubeclaw/backend/internal/application/skill"
	"kubeclaw/backend/internal/infrastructure/llm"
)

type capability struct {
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
	Level            string
	Audiences        []string
	Mutation         bool
	RequiresApproval bool
	RequestMode      string
	MCP              *applicationmcp.Record
	Skill            *appskill.Record
}

func (s *Service) loadCapabilities(ctx context.Context) []capability {
	service := s.caps
	if service == nil {
		service = applicationcapability.NewService(s.skills, s.mcp)
	}

	descriptors := service.ListForAudience(ctx, applicationcapability.AudienceAgent)
	capabilities := make([]capability, 0, len(descriptors))
	for _, item := range descriptors {
		capabilities = append(capabilities, capability{
			CapabilityType:   item.CapabilityType,
			ID:               item.ID,
			Name:             item.Name,
			Tool:             item.Tool,
			Role:             item.Role,
			Summary:          item.Summary,
			Actions:          append([]string(nil), item.Actions...),
			ResourceTypes:    append([]string(nil), item.ResourceTypes...),
			Keywords:         append([]string(nil), item.Keywords...),
			PreferredWeight:  item.PreferredWeight,
			Executor:         item.Executor,
			Transport:        item.Transport,
			DefaultPayload:   item.DefaultPayload,
			TargetAction:     item.TargetAction,
			TargetMCPID:      item.TargetMCPID,
			TargetMCPName:    item.TargetMCPName,
			SystemPrompt:     item.SystemPrompt,
			InputTemplate:    item.InputTemplate,
			Level:            string(item.Level),
			Audiences:        capabilityAudiences(item.Audiences),
			Mutation:         item.Mutation,
			RequiresApproval: item.RequiresApproval,
			RequestMode:      item.RequestMode,
			MCP:              item.MCP,
			Skill:            item.Skill,
		})
	}
	return capabilities
}

func capabilityAudiences(values []applicationcapability.Audience) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, string(value))
	}
	return result
}

func builtinCapabilities() []capability {
	return []capability{
		{
			CapabilityType:  "builtin",
			Name:            "builtin.cluster.namespaces",
			Tool:            "cluster.list_namespaces",
			Role:            "k8s_expert",
			Summary:         "Built-in Kubernetes namespace reader.",
			Actions:         []string{"list_namespaces"},
			Keywords:        []string{"namespace", "namespaces", "命名空间"},
			PreferredWeight: 30,
			Executor:        "builtin",
		},
		{
			CapabilityType:  "builtin",
			Name:            "builtin.cluster.resources",
			Tool:            "cluster.list_resources",
			Role:            "k8s_expert",
			Summary:         "Built-in Kubernetes resource reader for pods, deployments, services, configmaps, and secrets.",
			Actions:         []string{"list_resources"},
			ResourceTypes:   []string{"pods", "deployments", "services", "configmaps", "secrets"},
			Keywords:        []string{"k8s", "kubernetes", "cluster", "pod", "pods", "deployment", "deployments", "service", "services", "configmap", "secret", "资源", "部署", "服务", "容器组"},
			PreferredWeight: 30,
			Executor:        "builtin",
		},
		{
			CapabilityType:  "builtin",
			Name:            "builtin.cluster.events",
			Tool:            "cluster.list_events",
			Role:            "k8s_expert",
			Summary:         "Built-in Kubernetes event reader.",
			Actions:         []string{"list_events"},
			Keywords:        []string{"event", "events", "事件"},
			PreferredWeight: 30,
			Executor:        "builtin",
		},
		{
			CapabilityType:  "builtin",
			Name:            "builtin.cluster.delete",
			Tool:            "cluster.delete_resource",
			Role:            "k8s_expert",
			Summary:         "Built-in Kubernetes resource deletion tool.",
			Actions:         []string{"delete_resource"},
			ResourceTypes:   []string{"pods", "deployments", "services", "configmaps", "secrets"},
			Keywords:        []string{"delete", "remove", "删除"},
			PreferredWeight: 30,
			Executor:        "builtin",
		},
		{
			CapabilityType:  "builtin",
			Name:            "builtin.cluster.scale",
			Tool:            "cluster.scale_deployment",
			Role:            "k8s_expert",
			Summary:         "Built-in Kubernetes deployment scaler.",
			Actions:         []string{"scale_deployment"},
			ResourceTypes:   []string{"deployments"},
			Keywords:        []string{"scale", "replicas", "扩容", "缩容", "副本"},
			PreferredWeight: 30,
			Executor:        "builtin",
		},
		{
			CapabilityType:  "builtin",
			Name:            "builtin.cluster.restart",
			Tool:            "cluster.restart_deployment",
			Role:            "k8s_expert",
			Summary:         "Built-in Kubernetes deployment restart tool.",
			Actions:         []string{"restart_deployment"},
			ResourceTypes:   []string{"deployments"},
			Keywords:        []string{"restart", "rollout", "重启"},
			PreferredWeight: 30,
			Executor:        "builtin",
		},
		{
			CapabilityType:  "builtin",
			Name:            "builtin.cluster.apply",
			Tool:            "cluster.apply_yaml",
			Role:            "k8s_expert",
			Summary:         "Built-in Kubernetes YAML applier.",
			Actions:         []string{"apply_yaml"},
			Keywords:        []string{"apply", "manifest", "yaml", "kubectl", "应用yaml", "应用 yaml"},
			PreferredWeight: 30,
			Executor:        "builtin",
		},
		{
			CapabilityType:  "builtin",
			Name:            "builtin.models.list",
			Tool:            "model.list",
			Role:            "skill_mcp_expert",
			Summary:         "Built-in model catalog reader.",
			Actions:         []string{"list_models"},
			Keywords:        []string{"model", "models", "模型"},
			PreferredWeight: 20,
			Executor:        "builtin",
		},
		{
			CapabilityType:  "builtin",
			Name:            "builtin.skills.list",
			Tool:            "skill.list",
			Role:            "skill_mcp_expert",
			Summary:         "Built-in skill catalog reader.",
			Actions:         []string{"list_skills"},
			Keywords:        []string{"skill", "skills", "技能"},
			PreferredWeight: 20,
			Executor:        "builtin",
		},
		{
			CapabilityType:  "builtin",
			Name:            "builtin.mcp.list",
			Tool:            "mcp.list",
			Role:            "skill_mcp_expert",
			Summary:         "Built-in MCP catalog reader.",
			Actions:         []string{"list_mcp"},
			Keywords:        []string{"mcp"},
			PreferredWeight: 20,
			Executor:        "builtin",
		},
	}
}

func buildSkillCapability(record appskill.Record) (capability, bool) {
	if !isSkillRunnable(record.Status) {
		return capability{}, false
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
	targetAction := normalizePlannedKind(firstNonEmpty(
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

	return capability{
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
		Skill:           &record,
	}, true
}

func buildMCPCapability(record applicationmcp.Record) (capability, bool) {
	if !record.IsEnabled {
		return capability{}, false
	}

	fullText := strings.TrimSpace(strings.Join([]string{record.Name, record.Type, record.Description, record.Endpoint, record.Command}, " "))
	actions := inferActionKinds(fullText)
	resourceTypes := inferResourceTypes(fullText)
	keywords := uniqueLower(append(tokenizeCapabilityText(fullText), "mcp"))

	return capability{
		CapabilityType:  "mcp",
		ID:              record.ID,
		Name:            record.Name,
		Tool:            fmt.Sprintf("mcp.%s", sanitizeToolSegment(record.Name)),
		Role:            "mcp_worker",
		Summary:         buildCapabilitySummary("MCP", record.Name, record.Description, actions, resourceTypes),
		Actions:         actions,
		ResourceTypes:   resourceTypes,
		Keywords:        keywords,
		PreferredWeight: 50,
		Executor:        "mcp",
		Transport:       strings.ToLower(strings.TrimSpace(record.Transport)),
		MCP:             &record,
	}, true
}

func (s *Service) resolveIntentCapability(raw string, planned intent, capabilities []capability) intent {
	if planned.Kind == "" {
		return intent{Kind: "llm", Tool: "llm.chat", CapabilityType: "llm"}
	}
	if planned.Kind == "llm" {
		planned.CapabilityType = "llm"
		planned.Tool = "llm.chat"
		return planned
	}

	if cap := matchCapabilityBySelection(planned, capabilities); cap != nil && capabilitySupportsIntent(*cap, raw, planned) {
		return applyCapabilityToIntent(planned, *cap)
	}

	bestScore := -1
	var best capability
	for _, candidate := range capabilities {
		score := capabilityScore(raw, planned, candidate)
		if score > bestScore {
			bestScore = score
			best = candidate
		}
	}

	if bestScore < 0 {
		planned.CapabilityType = "builtin"
		if planned.Tool == "" {
			planned.Tool = defaultToolForKind(planned.Kind)
		}
		return planned
	}

	return applyCapabilityToIntent(planned, best)
}

func matchCapabilityBySelection(selected intent, capabilities []capability) *capability {
	for idx := range capabilities {
		item := &capabilities[idx]
		if selected.CapabilityType != "" && selected.CapabilityType != item.CapabilityType {
			continue
		}
		if selected.CapabilityID > 0 && selected.CapabilityID == item.ID {
			return item
		}
		if selected.CapabilityName != "" && strings.EqualFold(strings.TrimSpace(selected.CapabilityName), strings.TrimSpace(item.Name)) {
			return item
		}
	}
	return nil
}

func applyCapabilityToIntent(input intent, selected capability) intent {
	input.CapabilityType = selected.CapabilityType
	input.CapabilityID = selected.ID
	input.CapabilityName = selected.Name
	if input.Tool == "" {
		if selected.Tool != "" {
			input.Tool = selected.Tool
		} else {
			input.Tool = defaultToolForKind(input.Kind)
		}
	}
	if input.CapabilityRole == "" {
		input.CapabilityRole = selected.Role
	}
	return input
}

func capabilityScore(raw string, selected intent, item capability) int {
	if !capabilitySupportsIntent(item, raw, selected) {
		return -1
	}

	score := item.PreferredWeight
	normalizedQuery := normalizeCapabilityText(raw)
	normalizedName := normalizeCapabilityText(item.Name)
	explicitMention := normalizedName != "" && strings.Contains(normalizedQuery, normalizedName)
	if explicitMention {
		score += 120
	}

	lower := strings.ToLower(raw)
	for _, keyword := range item.Keywords {
		if keyword != "" && strings.Contains(lower, keyword) {
			score += 8
		}
	}

	if stringInSlice(selected.Kind, item.Actions) {
		score += 20
	}

	if selected.Kind == "list_resources" || selected.Kind == "delete_resource" {
		resourceType := normalizeResourceType(stringField(selected.Payload["type"]))
		switch {
		case resourceType == "":
		case len(item.ResourceTypes) == 0:
			score += 2
		case stringInSlice(resourceType, item.ResourceTypes):
			score += 10
		default:
			score -= 6
		}
	}

	return score
}

func capabilitySupportsIntent(item capability, raw string, selected intent) bool {
	if len(item.Actions) > 0 && !stringInSlice(selected.Kind, item.Actions) {
		return false
	}

	if len(item.Actions) == 0 && item.CapabilityType != "builtin" {
		normalizedQuery := normalizeCapabilityText(raw)
		normalizedName := normalizeCapabilityText(item.Name)
		return normalizedName != "" && strings.Contains(normalizedQuery, normalizedName)
	}

	if selected.Kind == "list_resources" || selected.Kind == "delete_resource" {
		resourceType := normalizeResourceType(stringField(selected.Payload["type"]))
		if resourceType != "" && len(item.ResourceTypes) > 0 && !stringInSlice(resourceType, item.ResourceTypes) {
			normalizedQuery := normalizeCapabilityText(raw)
			normalizedName := normalizeCapabilityText(item.Name)
			return normalizedName != "" && strings.Contains(normalizedQuery, normalizedName)
		}
	}

	return true
}

func (s *Service) executeCapabilityIntent(ctx context.Context, run Run, session applicationchat.Session, userMessage applicationchat.Message, selected intent, requestID string) (string, error) {
	clusterID := session.Context.ClusterID
	toolExecutionID, err := s.repo.CreateToolExecution(ctx, &run.ID, &run.UserID, clusterID, selected.Tool, selected.Payload)
	if err != nil {
		return "", err
	}

	startedAt := time.Now()
	role := s.intentRole(selected)
	s.publishEvent(ctx, run.ID, run.SessionID, "agent_spawn", role, StatusRunning, "specialist picked the task", map[string]any{
		"tool":             selected.Tool,
		"capabilityType":   selected.CapabilityType,
		"capabilityId":     selected.CapabilityID,
		"capabilityName":   selected.CapabilityName,
		"plannedAction":    selected.Kind,
		"requiresApproval": selected.RequiresApproval,
	}, requestID)
	s.publishEvent(ctx, run.ID, run.SessionID, "tool_start", role, StatusRunning, "tool execution started", map[string]any{
		"toolExecutionId": toolExecutionID,
		"tool":            selected.Tool,
		"capabilityType":  selected.CapabilityType,
		"capabilityName":  selected.CapabilityName,
	}, requestID)

	result, execErr := s.runCapabilityIntent(ctx, session, userMessage, selected)
	status := "succeeded"
	if execErr != nil {
		status = "failed"
	}
	_ = s.repo.CompleteToolExecution(ctx, toolExecutionID, status, result, time.Since(startedAt).Milliseconds())
	if execErr != nil {
		return "", execErr
	}

	s.publishEvent(ctx, run.ID, run.SessionID, "tool_end", role, StatusCompleted, "tool execution completed", map[string]any{
		"toolExecutionId": toolExecutionID,
		"tool":            selected.Tool,
		"result":          result,
	}, requestID)
	s.publishEvent(ctx, run.ID, run.SessionID, "agent_result", role, StatusCompleted, "specialist returned a result", map[string]any{
		"tool":           selected.Tool,
		"capabilityType": selected.CapabilityType,
		"capabilityName": selected.CapabilityName,
	}, requestID)

	return result, nil
}

func (s *Service) runCapabilityIntent(ctx context.Context, session applicationchat.Session, userMessage applicationchat.Message, selected intent) (string, error) {
	service := s.caps
	if service == nil {
		service = applicationcapability.NewService(s.skills, s.mcp).WithRuntime(s.models, s.clusters, s.llm)
	}

	return service.Invoke(ctx, applicationcapability.InvokeInput{
		Audience: applicationcapability.AudienceAgent,
		Selection: applicationcapability.Selection{
			Action:         selected.Kind,
			Tool:           selected.Tool,
			CapabilityType: selected.CapabilityType,
			CapabilityID:   selected.CapabilityID,
			CapabilityName: selected.CapabilityName,
			Payload:        selected.Payload,
		},
		Context: applicationcapability.InvokeContext{
			ClusterID: session.Context.ClusterID,
			Namespace: session.Context.Namespace,
			ModelID:   session.Context.ModelID,
		},
		UserInput: userMessage.Content,
	})
}

func (s *Service) runBuiltinIntent(ctx context.Context, session applicationchat.Session, selected intent) (string, error) {
	switch selected.Kind {
	case "list_namespaces":
		if session.Context.ClusterID == nil {
			return "", fmt.Errorf("select a cluster before asking for namespaces")
		}
		items, err := s.clusters.ListNamespaces(ctx, *session.Context.ClusterID)
		if err != nil {
			return "", err
		}
		return renderJSON(items)
	case "list_resources":
		if session.Context.ClusterID == nil {
			return "", fmt.Errorf("select a cluster before asking for kubernetes resources")
		}
		items, err := s.clusters.ListResources(ctx, *session.Context.ClusterID, applicationcluster.ResourceQuery{
			Type:      stringField(selected.Payload["type"]),
			Namespace: namespaceFromContext(session, selected),
		})
		if err != nil {
			return "", err
		}
		return renderJSON(items)
	case "list_events":
		if session.Context.ClusterID == nil {
			return "", fmt.Errorf("select a cluster before asking for kubernetes events")
		}
		items, err := s.clusters.ListEvents(ctx, *session.Context.ClusterID, namespaceFromContext(session, selected))
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
		if session.Context.ClusterID == nil {
			return "", fmt.Errorf("select a cluster before deleting kubernetes resources")
		}
		resourceType := normalizeResourceType(stringField(selected.Payload["type"]))
		name := strings.TrimSpace(stringField(selected.Payload["name"]))
		namespace := defaultString(namespaceFromContext(session, selected), "default")
		if name == "" {
			return "", fmt.Errorf("resource name is required")
		}
		if resourceType == "" {
			resourceType = inferResourceType(name)
		}
		if err := s.clusters.DeleteResource(ctx, *session.Context.ClusterID, applicationcluster.ResourceQuery{
			Type:      resourceType,
			Namespace: namespace,
		}, name); err != nil {
			return "", err
		}
		return fmt.Sprintf("Deleted %s %s in namespace %s.", resourceType, name, namespace), nil
	case "scale_deployment":
		if session.Context.ClusterID == nil {
			return "", fmt.Errorf("select a cluster before scaling a deployment")
		}
		name := strings.TrimSpace(stringField(selected.Payload["name"]))
		namespace := defaultString(namespaceFromContext(session, selected), "default")
		replicas := int32(numberField(selected.Payload["replicas"]))
		if name == "" {
			return "", fmt.Errorf("deployment name is required")
		}
		if replicas <= 0 {
			return "", fmt.Errorf("replicas must be greater than zero")
		}
		if err := s.clusters.ScaleDeployment(ctx, *session.Context.ClusterID, namespace, name, replicas); err != nil {
			return "", err
		}
		return fmt.Sprintf("Scaled deployment %s in namespace %s to %d replicas.", name, namespace, replicas), nil
	case "restart_deployment":
		if session.Context.ClusterID == nil {
			return "", fmt.Errorf("select a cluster before restarting a deployment")
		}
		name := strings.TrimSpace(stringField(selected.Payload["name"]))
		namespace := defaultString(namespaceFromContext(session, selected), "default")
		if name == "" {
			return "", fmt.Errorf("deployment name is required")
		}
		if err := s.clusters.RestartDeployment(ctx, *session.Context.ClusterID, namespace, name); err != nil {
			return "", err
		}
		return fmt.Sprintf("Restarted deployment %s in namespace %s.", name, namespace), nil
	case "apply_yaml":
		if session.Context.ClusterID == nil {
			return "", fmt.Errorf("select a cluster before applying yaml")
		}
		manifest := strings.TrimSpace(stringField(selected.Payload["manifest"]))
		if manifest == "" {
			return "", fmt.Errorf("manifest is required")
		}
		result, err := s.clusters.ApplyYAML(ctx, *session.Context.ClusterID, manifest)
		if err != nil {
			return "", err
		}
		if result == nil {
			return "Manifest applied.", nil
		}
		return defaultString(result.Summary, "Manifest applied."), nil
	default:
		return "", fmt.Errorf("unsupported intent: %s", selected.Kind)
	}
}

func (s *Service) runSkillIntent(ctx context.Context, session applicationchat.Session, userMessage applicationchat.Message, selected intent) (string, error) {
	capability, err := s.resolveCapability(ctx, selected)
	if err != nil {
		return "", err
	}

	action := selected.Kind
	if capability.TargetAction != "" && capability.TargetAction != "llm" {
		action = capability.TargetAction
	}
	payload := mergePayloads(capability.DefaultPayload, selected.Payload)

	switch capability.Executor {
	case "mcp":
		if capability.TargetMCPID == 0 && strings.TrimSpace(capability.TargetMCPName) == "" {
			return "", fmt.Errorf("skill %s is configured for mcp execution but does not declare a target mcp", capability.Name)
		}
		nested := selected
		nested.Kind = action
		nested.Payload = payload
		nested.CapabilityType = "mcp"
		nested.CapabilityRole = "mcp_worker"
		nested.CapabilityID = capability.TargetMCPID
		nested.CapabilityName = capability.TargetMCPName
		if nested.Tool == "" || strings.HasPrefix(nested.Tool, "skill.") {
			if nested.CapabilityName != "" {
				nested.Tool = fmt.Sprintf("mcp.%s.%s", sanitizeToolSegment(nested.CapabilityName), action)
			} else {
				nested.Tool = fmt.Sprintf("mcp.%d.%s", nested.CapabilityID, action)
			}
		}
		return s.runMCPIntent(ctx, session, userMessage, nested)
	case "llm":
		return s.runSkillLLMIntent(ctx, session, userMessage, *capability, payload)
	default:
		nested := selected
		nested.Kind = action
		nested.Payload = payload
		nested.CapabilityType = "builtin"
		nested.CapabilityID = 0
		nested.CapabilityName = "builtin"
		nested.CapabilityRole = "k8s_expert"
		nested.Tool = defaultToolForKind(action)
		return s.runBuiltinIntent(ctx, session, nested)
	}
}

func (s *Service) runSkillLLMIntent(ctx context.Context, session applicationchat.Session, userMessage applicationchat.Message, capability capability, payload map[string]any) (string, error) {
	resolvedModel, err := s.resolveSessionModel(ctx, session.Context.ModelID)
	if err != nil {
		return "", err
	}

	systemPrompt := strings.TrimSpace(capability.SystemPrompt)
	if systemPrompt == "" {
		systemPrompt = fmt.Sprintf("You are the %s skill in KubeClaw. Answer in the same language as the user and stay concise.", capability.Name)
	}

	userContent := applyCapabilityTemplate(capability.InputTemplate, userMessage.Content, session, payload)
	result, err := s.llm.Chat(ctx, llm.ChatInput{
		Model: *resolvedModel,
		Messages: []llm.Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userContent},
		},
	})
	if err != nil {
		return "", fmt.Errorf("skill %s execution failed: %w", capability.Name, err)
	}

	content, strippedThink := sanitizeLLMOutput(result.Content)
	if content == "" && strippedThink {
		return "", fmt.Errorf("skill %s returned reasoning only without a final answer", capability.Name)
	}
	if content == "" {
		return "", fmt.Errorf("skill %s returned an empty answer", capability.Name)
	}
	return content, nil
}

func (s *Service) runMCPIntent(ctx context.Context, session applicationchat.Session, userMessage applicationchat.Message, selected intent) (string, error) {
	capability, err := s.resolveCapability(ctx, selected)
	if err != nil {
		return "", err
	}
	if capability.MCP == nil {
		return "", fmt.Errorf("mcp capability %s is missing server configuration", capability.Name)
	}

	payload := mergePayloads(capability.DefaultPayload, selected.Payload)
	requestBody := map[string]any{
		"action": selected.Kind,
		"tool":   selected.Tool,
		"input":  payload,
		"session": map[string]any{
			"clusterId": session.Context.ClusterID,
			"namespace": session.Context.Namespace,
			"modelId":   session.Context.ModelID,
		},
		"userMessage": userMessage.Content,
		"capability": map[string]any{
			"id":   capability.ID,
			"name": capability.Name,
			"type": capability.CapabilityType,
		},
	}

	switch strings.ToLower(defaultString(capability.Transport, capability.MCP.Transport)) {
	case "http", "https":
		return invokeHTTPMCP(ctx, *capability.MCP, requestBody)
	case "command", "stdio", "process":
		return invokeCommandMCP(ctx, *capability.MCP, requestBody)
	default:
		return "", fmt.Errorf("unsupported mcp transport: %s", defaultString(capability.Transport, capability.MCP.Transport))
	}
}

func (s *Service) resolveCapability(ctx context.Context, selected intent) (*capability, error) {
	capabilities := s.loadCapabilities(ctx)
	if capability := matchCapabilityBySelection(selected, capabilities); capability != nil {
		return capability, nil
	}

	for idx := range capabilities {
		item := capabilities[idx]
		if item.Tool != "" && item.Tool == selected.Tool {
			return &item, nil
		}
	}

	return nil, fmt.Errorf("capability not found for %s", defaultString(selected.CapabilityName, selected.Tool))
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

func buildApprovalIntentPayload(selected intent) map[string]any {
	payload := mergePayloads(selected.Payload, nil)
	payload["intentKind"] = selected.Kind
	payload["tool"] = selected.Tool
	payload["capabilityType"] = selected.CapabilityType
	payload["capabilityId"] = selected.CapabilityID
	payload["capabilityName"] = selected.CapabilityName
	payload["capabilityRole"] = selected.CapabilityRole
	return payload
}

func approvalIntentFromPayload(payload map[string]any) intent {
	selected := intent{
		Kind:           normalizePlannedKind(stringField(payload["intentKind"])),
		Tool:           stringField(payload["tool"]),
		CapabilityType: stringField(payload["capabilityType"]),
		CapabilityID:   int64(numberField(payload["capabilityId"])),
		CapabilityName: stringField(payload["capabilityName"]),
		CapabilityRole: stringField(payload["capabilityRole"]),
		Payload:        map[string]any{},
	}

	for key, value := range payload {
		switch key {
		case "intentKind", "tool", "capabilityType", "capabilityId", "capabilityName", "capabilityRole", "matchedWord":
		default:
			selected.Payload[key] = value
		}
	}

	if selected.Kind == "" {
		selected.Kind = "llm"
	}
	if selected.Kind == "llm" {
		selected.CapabilityType = "llm"
		selected.Tool = "llm.chat"
	}
	return selected
}

func isMutationKind(kind string) bool {
	switch kind {
	case "delete_resource", "scale_deployment", "restart_deployment", "apply_yaml":
		return true
	default:
		return false
	}
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

	if strings.Contains(lower, "namespace") || strings.Contains(text, "命名空间") {
		actions = append(actions, "list_namespaces")
	}
	if strings.Contains(lower, "event") || strings.Contains(text, "事件") {
		actions = append(actions, "list_events")
	}
	if matchesAny(lower, []string{"pod", "pods", "deployment", "deployments", "service", "services", "configmap", "secret", "k8s", "kubernetes", "cluster"}) || matchesAny(text, []string{"容器组", "部署", "服务", "资源"}) {
		actions = append(actions, "list_resources")
	}
	if strings.Contains(lower, "delete") || strings.Contains(lower, "remove") || strings.Contains(text, "删除") {
		actions = append(actions, "delete_resource")
	}
	if strings.Contains(lower, "scale") || strings.Contains(lower, "replica") || strings.Contains(text, "扩容") || strings.Contains(text, "缩容") || strings.Contains(text, "副本") {
		actions = append(actions, "scale_deployment")
	}
	if strings.Contains(lower, "restart") || strings.Contains(lower, "rollout") || strings.Contains(text, "重启") {
		actions = append(actions, "restart_deployment")
	}
	if strings.Contains(lower, "apply") || strings.Contains(lower, "manifest") || strings.Contains(lower, "yaml") || strings.Contains(text, "应用yaml") || strings.Contains(text, "应用 yaml") {
		actions = append(actions, "apply_yaml")
	}
	if strings.Contains(lower, "model") || strings.Contains(text, "模型") {
		actions = append(actions, "list_models")
	}
	if strings.Contains(lower, "skill") || strings.Contains(text, "技能") {
		actions = append(actions, "list_skills")
	}
	if strings.Contains(lower, "mcp") {
		actions = append(actions, "list_mcp")
	}

	return normalizeActionList(actions)
}

func inferResourceTypes(text string) []string {
	lower := strings.ToLower(text)
	resourceTypes := make([]string, 0, 5)
	if strings.Contains(lower, "pod") || strings.Contains(text, "容器组") {
		resourceTypes = append(resourceTypes, "pods")
	}
	if strings.Contains(lower, "deployment") || strings.Contains(text, "部署") {
		resourceTypes = append(resourceTypes, "deployments")
	}
	if strings.Contains(lower, "service") || strings.Contains(text, "服务") {
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
		normalized := normalizePlannedKind(value)
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

func stringInSlice(value string, items []string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func normalizeCapabilityText(text string) string {
	lower := strings.ToLower(text)
	replacer := strings.NewReplacer(" ", "", "-", "", "_", "", ".", "", "/", "", "\\", "")
	return replacer.Replace(lower)
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

func applyCapabilityTemplate(template string, userInput string, session applicationchat.Session, payload map[string]any) string {
	if strings.TrimSpace(template) == "" {
		return userInput
	}

	payloadJSON, _ := json.Marshal(payload)
	clusterID := ""
	if session.Context.ClusterID != nil {
		clusterID = fmt.Sprintf("%d", *session.Context.ClusterID)
	}

	replacer := strings.NewReplacer(
		"{input}", userInput,
		"{userInput}", userInput,
		"{namespace}", session.Context.Namespace,
		"{clusterId}", clusterID,
		"{payload}", string(payloadJSON),
	)
	return replacer.Replace(template)
}
