package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"sort"
	"strings"
	"time"

	applicationcluster "kubeclaw/backend/internal/application/cluster"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	k8syaml "sigs.k8s.io/yaml"
)

const defaultNamespace = "default"

type storedCredentials struct {
	Username              string   `json:"username"`
	Password              string   `json:"password"`
	Namespace             string   `json:"namespace"`
	ServerName            string   `json:"serverName"`
	InsecureSkipTLSVerify bool     `json:"insecureSkipTlsVerify"`
	ImpersonateUser       string   `json:"impersonateUser"`
	ImpersonateGroups     []string `json:"impersonateGroups"`
}

type resourceMeta struct {
	Type       string
	Kind       string
	GVR        schema.GroupVersionResource
	Namespaced bool
}

type Gateway struct{}

func NewGateway() *Gateway {
	return &Gateway{}
}

func (g *Gateway) Validate(ctx context.Context, connection applicationcluster.Connection) (*applicationcluster.ValidationRecord, error) {
	clientset, _, err := g.clients(connection)
	if err != nil {
		return nil, err
	}

	version, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, fmt.Errorf("query kubernetes version: %w", err)
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 20})
	if err != nil {
		return nil, fmt.Errorf("list namespaces during validation: %w", err)
	}

	return &applicationcluster.ValidationRecord{
		Reachable:       true,
		Version:         version.GitVersion,
		NamespacesCount: len(namespaces.Items),
		Message:         "cluster connection is healthy",
		CheckedAt:       time.Now(),
	}, nil
}

func (g *Gateway) GetOverview(ctx context.Context, connection applicationcluster.Connection, namespace string) (*applicationcluster.OverviewRecord, error) {
	clientset, _, err := g.clients(connection)
	if err != nil {
		return nil, err
	}

	scopeNamespace := namespace
	if strings.TrimSpace(scopeNamespace) == "" || strings.EqualFold(scopeNamespace, "all") {
		scopeNamespace = metav1.NamespaceAll
	}

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list namespaces for overview: %w", err)
	}

	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list nodes for overview: %w", err)
	}

	pods, err := clientset.CoreV1().Pods(scopeNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list pods for overview: %w", err)
	}

	deployments, err := clientset.AppsV1().Deployments(scopeNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list deployments for overview: %w", err)
	}

	services, err := clientset.CoreV1().Services(scopeNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list services for overview: %w", err)
	}

	events, err := g.ListEvents(ctx, connection, namespace)
	if err != nil {
		return nil, err
	}

	overview := &applicationcluster.OverviewRecord{
		NamespacesCount: len(namespaces.Items),
		NodeCount:       len(nodes.Items),
		ServiceCount:    len(services.Items),
		CollectedAt:     time.Now(),
	}

	for _, node := range nodes.Items {
		if nodeReady(node) {
			overview.ReadyNodeCount++
		}
	}

	for _, pod := range pods.Items {
		overview.PodCount++
		switch pod.Status.Phase {
		case corev1.PodRunning:
			overview.RunningPodCount++
		case corev1.PodPending:
			overview.PendingPodCount++
		case corev1.PodFailed:
			overview.FailedPodCount++
		}

		podSummary := toProblemPodRecord(pod)
		if shouldShowProblemPod(podSummary) {
			overview.ProblemPods = append(overview.ProblemPods, podSummary)
		}
	}

	sort.Slice(overview.ProblemPods, func(i, j int) bool {
		if overview.ProblemPods[i].Status == overview.ProblemPods[j].Status {
			return overview.ProblemPods[i].Name < overview.ProblemPods[j].Name
		}
		return overview.ProblemPods[i].Status > overview.ProblemPods[j].Status
	})
	if len(overview.ProblemPods) > 8 {
		overview.ProblemPods = overview.ProblemPods[:8]
	}

	for _, deployment := range deployments.Items {
		overview.DeploymentCount++
		item := toDeploymentHealthRecord(deployment)
		if item.ReadyReplicas >= item.Replicas && item.Replicas > 0 {
			overview.ReadyDeploymentCount++
		}
		overview.Deployments = append(overview.Deployments, item)
	}

	sort.Slice(overview.Deployments, func(i, j int) bool {
		if overview.Deployments[i].Namespace == overview.Deployments[j].Namespace {
			return overview.Deployments[i].Name < overview.Deployments[j].Name
		}
		return overview.Deployments[i].Namespace < overview.Deployments[j].Namespace
	})
	if len(overview.Deployments) > 12 {
		overview.Deployments = overview.Deployments[:12]
	}

	if len(events) > 10 {
		overview.RecentEvents = events[:10]
	} else {
		overview.RecentEvents = events
	}

	return overview, nil
}

func (g *Gateway) ListNamespaces(ctx context.Context, connection applicationcluster.Connection) ([]applicationcluster.NamespaceRecord, error) {
	clientset, _, err := g.clients(connection)
	if err != nil {
		return nil, err
	}

	list, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list namespaces: %w", err)
	}

	result := make([]applicationcluster.NamespaceRecord, 0, len(list.Items))
	for _, item := range list.Items {
		result = append(result, applicationcluster.NamespaceRecord{
			Name:      item.Name,
			Status:    string(item.Status.Phase),
			Labels:    item.Labels,
			CreatedAt: item.CreationTimestamp.Time,
		})
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (g *Gateway) ListResources(ctx context.Context, connection applicationcluster.Connection, query applicationcluster.ResourceQuery) ([]applicationcluster.ResourceRecord, error) {
	_, dynamicClient, err := g.clients(connection)
	if err != nil {
		return nil, err
	}

	meta, err := resolveResourceMeta(query.Type)
	if err != nil {
		return nil, err
	}

	resource := g.resourceInterface(dynamicClient, connection, meta, query.Namespace)
	list, err := resource.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list resources: %w", err)
	}

	result := make([]applicationcluster.ResourceRecord, 0, len(list.Items))
	for _, item := range list.Items {
		result = append(result, applicationcluster.ResourceRecord{
			Type:      meta.Type,
			Kind:      item.GetKind(),
			Name:      item.GetName(),
			Namespace: item.GetNamespace(),
			Status:    summarizeResourceStatus(meta.Type, item),
			Labels:    item.GetLabels(),
			CreatedAt: item.GetCreationTimestamp().Time,
		})
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func (g *Gateway) GetResource(ctx context.Context, connection applicationcluster.Connection, query applicationcluster.ResourceQuery, name string) (*applicationcluster.ResourceDetail, error) {
	_, dynamicClient, err := g.clients(connection)
	if err != nil {
		return nil, err
	}

	meta, err := resolveResourceMeta(query.Type)
	if err != nil {
		return nil, err
	}

	resource := g.resourceInterface(dynamicClient, connection, meta, query.Namespace)
	item, err := resource.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get resource: %w", err)
	}

	return &applicationcluster.ResourceDetail{
		Type:      meta.Type,
		Kind:      item.GetKind(),
		Name:      item.GetName(),
		Namespace: item.GetNamespace(),
		Object:    item.Object,
	}, nil
}

func (g *Gateway) ListEvents(ctx context.Context, connection applicationcluster.Connection, namespace string) ([]applicationcluster.EventRecord, error) {
	clientset, _, err := g.clients(connection)
	if err != nil {
		return nil, err
	}

	ns := namespace
	if strings.TrimSpace(ns) == "" {
		ns = metav1.NamespaceAll
	}

	list, err := clientset.CoreV1().Events(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	result := make([]applicationcluster.EventRecord, 0, len(list.Items))
	for _, item := range list.Items {
		result = append(result, toEventRecord(item))
	}

	sort.Slice(result, func(i, j int) bool {
		return eventSortTime(result[i]).After(eventSortTime(result[j]))
	})
	return result, nil
}

func (g *Gateway) StreamPodLogs(ctx context.Context, connection applicationcluster.Connection, query applicationcluster.PodLogQuery) (io.ReadCloser, error) {
	cfg, err := g.restConfig(connection)
	if err != nil {
		return nil, err
	}
	// Pod 日志跟随需要长连接，这里关闭 client-go 默认超时。
	cfg.Timeout = 0

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("build kubernetes clientset for logs: %w", err)
	}

	ns := g.namespaceOrDefault(connection, query.Namespace)
	tailLines := query.TailLines
	if tailLines <= 0 {
		tailLines = 200
	}

	options := &corev1.PodLogOptions{
		Container: query.Container,
		Follow:    query.Follow,
		TailLines: &tailLines,
	}
	if query.SinceSeconds > 0 {
		options.SinceSeconds = &query.SinceSeconds
	}

	stream, err := clientset.CoreV1().Pods(ns).GetLogs(query.PodName, options).Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("stream pod logs: %w", err)
	}

	return stream, nil
}

func (g *Gateway) DeleteResource(ctx context.Context, connection applicationcluster.Connection, query applicationcluster.ResourceQuery, name string) error {
	_, dynamicClient, err := g.clients(connection)
	if err != nil {
		return err
	}

	meta, err := resolveResourceMeta(query.Type)
	if err != nil {
		return err
	}

	resource := g.resourceInterface(dynamicClient, connection, meta, query.Namespace)
	if err := resource.Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return fmt.Errorf("delete resource: %w", err)
	}
	return nil
}

func (g *Gateway) ScaleDeployment(ctx context.Context, connection applicationcluster.Connection, namespace string, name string, replicas int32) error {
	clientset, _, err := g.clients(connection)
	if err != nil {
		return err
	}

	ns := g.namespaceOrDefault(connection, namespace)
	deployment, err := clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get deployment for scaling: %w", err)
	}

	deployment.Spec.Replicas = &replicas
	if _, err := clientset.AppsV1().Deployments(ns).Update(ctx, deployment, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("scale deployment: %w", err)
	}
	return nil
}

func (g *Gateway) RestartDeployment(ctx context.Context, connection applicationcluster.Connection, namespace string, name string) error {
	clientset, _, err := g.clients(connection)
	if err != nil {
		return err
	}

	ns := g.namespaceOrDefault(connection, namespace)
	restartedAt := time.Now().UTC().Format(time.RFC3339)
	patch := fmt.Sprintf(`{"spec":{"template":{"metadata":{"annotations":{"kubectl.kubernetes.io/restartedAt":"%s"}}}}}`, restartedAt)
	if _, err := clientset.AppsV1().Deployments(ns).Patch(ctx, name, types.StrategicMergePatchType, []byte(patch), metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("restart deployment: %w", err)
	}
	return nil
}

func (g *Gateway) ApplyYAML(ctx context.Context, connection applicationcluster.Connection, manifest string) (*applicationcluster.ApplyResult, error) {
	_, dynamicClient, err := g.clients(connection)
	if err != nil {
		return nil, err
	}

	documents := splitYAMLDocuments(manifest)
	if len(documents) == 0 {
		return nil, fmt.Errorf("manifest is empty")
	}

	resources := make([]string, 0, len(documents))
	for _, doc := range documents {
		jsonBody, err := k8syaml.YAMLToJSON([]byte(doc))
		if err != nil {
			return nil, fmt.Errorf("convert yaml to json: %w", err)
		}

		var object map[string]any
		if err := json.Unmarshal(jsonBody, &object); err != nil {
			return nil, fmt.Errorf("decode manifest json: %w", err)
		}

		item := &unstructured.Unstructured{Object: object}
		meta, err := resolveResourceMetaByManifest(item)
		if err != nil {
			return nil, err
		}

		resource := g.resourceInterface(dynamicClient, connection, meta, item.GetNamespace())

		current, err := resource.Get(ctx, item.GetName(), metav1.GetOptions{})
		if err != nil {
			if _, createErr := resource.Create(ctx, item, metav1.CreateOptions{}); createErr != nil {
				return nil, fmt.Errorf("create manifest resource: %w", createErr)
			}
		} else {
			item.SetResourceVersion(current.GetResourceVersion())
			if _, updateErr := resource.Update(ctx, item, metav1.UpdateOptions{}); updateErr != nil {
				return nil, fmt.Errorf("update manifest resource: %w", updateErr)
			}
		}

		resources = append(resources, fmt.Sprintf("%s/%s", strings.ToLower(item.GetKind()), item.GetName()))
	}

	return &applicationcluster.ApplyResult{
		Summary:   fmt.Sprintf("Applied %d resource(s).", len(resources)),
		Resources: resources,
	}, nil
}

func (g *Gateway) clients(connection applicationcluster.Connection) (*kubernetes.Clientset, dynamic.Interface, error) {
	restConfig, err := g.restConfig(connection)
	if err != nil {
		return nil, nil, err
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("build kubernetes clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("build kubernetes dynamic client: %w", err)
	}

	return clientset, dynamicClient, nil
}

func (g *Gateway) restConfig(connection applicationcluster.Connection) (*rest.Config, error) {
	credentials := parseCredentials(connection.Credentials)
	serverNameFromKubeconfig := detectKubeconfigServerName(connection.KubeConfig)
	explicitServerName := chooseServerName(
		credentials.ServerName,
		serverNameFromKubeconfig,
		hostNameFromServer(connection.APIserver),
	)

	var cfg *rest.Config
	var err error

	if strings.TrimSpace(connection.KubeConfig) != "" {
		cfg, err = clientcmd.RESTConfigFromKubeConfig([]byte(connection.KubeConfig))
		if err != nil {
			return nil, fmt.Errorf("build rest config from kubeconfig: %w", err)
		}

		if strings.TrimSpace(connection.CACert) != "" {
			cfg.TLSClientConfig.CAData = []byte(connection.CACert)
		}
		if strings.TrimSpace(connection.Token) != "" {
			cfg.BearerToken = connection.Token
		}
		if strings.TrimSpace(connection.APIserver) != "" {
			cfg.Host = connection.APIserver
		}
	} else {
		if strings.TrimSpace(connection.APIserver) == "" {
			return nil, fmt.Errorf("cluster apiServer is empty")
		}

		cfg = &rest.Config{
			Host:        connection.APIserver,
			BearerToken: connection.Token,
			Username:    credentials.Username,
			Password:    credentials.Password,
			TLSClientConfig: rest.TLSClientConfig{
				CAData:   []byte(connection.CACert),
				Insecure: credentials.InsecureSkipTLSVerify,
			},
		}
	}

	if explicitServerName != "" {
		cfg.TLSClientConfig.ServerName = explicitServerName
	}
	if credentials.InsecureSkipTLSVerify {
		cfg.TLSClientConfig.Insecure = true
	}

	cfg.Timeout = 10 * time.Second
	if credentials.ImpersonateUser != "" {
		cfg.Impersonate.UserName = credentials.ImpersonateUser
		cfg.Impersonate.Groups = credentials.ImpersonateGroups
	}

	return cfg, nil
}

func (g *Gateway) namespaceOrDefault(connection applicationcluster.Connection, namespace string) string {
	if strings.TrimSpace(namespace) != "" {
		return namespace
	}

	credentials := parseCredentials(connection.Credentials)
	if strings.TrimSpace(credentials.Namespace) != "" {
		return credentials.Namespace
	}

	return defaultNamespace
}

func (g *Gateway) resourceInterface(dynamicClient dynamic.Interface, connection applicationcluster.Connection, meta resourceMeta, namespace string) dynamic.ResourceInterface {
	resource := dynamicClient.Resource(meta.GVR)
	if meta.Namespaced {
		return resource.Namespace(g.namespaceOrDefault(connection, namespace))
	}
	return resource
}

func parseCredentials(raw string) storedCredentials {
	if strings.TrimSpace(raw) == "" {
		return storedCredentials{}
	}

	var result storedCredentials
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return storedCredentials{}
	}
	return result
}

func detectKubeconfigServerName(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}

	cfg, err := clientcmd.Load([]byte(raw))
	if err != nil {
		return ""
	}

	contextName := cfg.CurrentContext
	if contextName == "" {
		for name := range cfg.Contexts {
			contextName = name
			break
		}
	}
	if contextName == "" {
		return ""
	}

	currentContext, ok := cfg.Contexts[contextName]
	if !ok || currentContext == nil {
		return ""
	}

	clusterInfo, ok := cfg.Clusters[currentContext.Cluster]
	if !ok || clusterInfo == nil {
		return ""
	}

	return hostNameFromServer(clusterInfo.Server)
}

func hostNameFromServer(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}

	host := parsed.Hostname()
	if host == "" {
		return ""
	}
	if net.ParseIP(host) != nil {
		return ""
	}

	return host
}

func chooseServerName(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func resolveResourceMeta(resourceType string) (resourceMeta, error) {
	switch strings.ToLower(strings.TrimSpace(resourceType)) {
	case "pod", "pods":
		return resourceMeta{Type: "pods", Kind: "Pod", Namespaced: true, GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"}}, nil
	case "service", "services", "svc":
		return resourceMeta{Type: "services", Kind: "Service", Namespaced: true, GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "services"}}, nil
	case "configmap", "configmaps":
		return resourceMeta{Type: "configmaps", Kind: "ConfigMap", Namespaced: true, GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}}, nil
	case "secret", "secrets":
		return resourceMeta{Type: "secrets", Kind: "Secret", Namespaced: true, GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}}, nil
	case "deployment", "deployments":
		return resourceMeta{Type: "deployments", Kind: "Deployment", Namespaced: true, GVR: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}}, nil
	case "statefulset", "statefulsets":
		return resourceMeta{Type: "statefulsets", Kind: "StatefulSet", Namespaced: true, GVR: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "statefulsets"}}, nil
	case "daemonset", "daemonsets":
		return resourceMeta{Type: "daemonsets", Kind: "DaemonSet", Namespaced: true, GVR: schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonsets"}}, nil
	case "namespace", "namespaces", "ns":
		return resourceMeta{Type: "namespaces", Kind: "Namespace", Namespaced: false, GVR: schema.GroupVersionResource{Group: "", Version: "v1", Resource: "namespaces"}}, nil
	default:
		return resourceMeta{}, fmt.Errorf("%w: %s", applicationcluster.ErrInvalidResourceType, resourceType)
	}
}

func resolveResourceMetaByManifest(item *unstructured.Unstructured) (resourceMeta, error) {
	return resolveResourceMeta(item.GetKind())
}

func summarizeResourceStatus(resourceType string, item unstructured.Unstructured) string {
	switch strings.ToLower(resourceType) {
	case "pods", "pod":
		if phase, ok, _ := unstructured.NestedString(item.Object, "status", "phase"); ok {
			return phase
		}
	case "deployments", "deployment":
		ready, _, _ := unstructured.NestedInt64(item.Object, "status", "readyReplicas")
		replicas, _, _ := unstructured.NestedInt64(item.Object, "status", "replicas")
		return fmt.Sprintf("%d/%d ready", ready, replicas)
	case "statefulsets", "statefulset":
		ready, _, _ := unstructured.NestedInt64(item.Object, "status", "readyReplicas")
		replicas, _, _ := unstructured.NestedInt64(item.Object, "status", "replicas")
		return fmt.Sprintf("%d/%d ready", ready, replicas)
	case "daemonsets", "daemonset":
		ready, _, _ := unstructured.NestedInt64(item.Object, "status", "numberReady")
		desired, _, _ := unstructured.NestedInt64(item.Object, "status", "desiredNumberScheduled")
		return fmt.Sprintf("%d/%d ready", ready, desired)
	case "services", "service":
		if serviceType, ok, _ := unstructured.NestedString(item.Object, "spec", "type"); ok {
			return serviceType
		}
	case "configmaps", "configmap":
		return "Active"
	case "secrets", "secret":
		if secretType, ok, _ := unstructured.NestedString(item.Object, "type"); ok {
			return secretType
		}
	case "namespaces", "namespace":
		if phase, ok, _ := unstructured.NestedString(item.Object, "status", "phase"); ok {
			return phase
		}
	}

	return "Unknown"
}

func toEventRecord(item corev1.Event) applicationcluster.EventRecord {
	var firstSeenAt *time.Time
	if !item.FirstTimestamp.IsZero() {
		t := item.FirstTimestamp.Time
		firstSeenAt = &t
	}

	var lastSeenAt *time.Time
	switch {
	case !item.LastTimestamp.IsZero():
		t := item.LastTimestamp.Time
		lastSeenAt = &t
	case !item.EventTime.IsZero():
		t := item.EventTime.Time
		lastSeenAt = &t
	}

	return applicationcluster.EventRecord{
		Type:           item.Type,
		Reason:         item.Reason,
		Message:        item.Message,
		Namespace:      item.Namespace,
		InvolvedObject: fmt.Sprintf("%s/%s", item.InvolvedObject.Kind, item.InvolvedObject.Name),
		Count:          item.Count,
		FirstSeenAt:    firstSeenAt,
		LastSeenAt:     lastSeenAt,
	}
}

func eventSortTime(item applicationcluster.EventRecord) time.Time {
	switch {
	case item.LastSeenAt != nil:
		return *item.LastSeenAt
	case item.FirstSeenAt != nil:
		return *item.FirstSeenAt
	default:
		return time.Time{}
	}
}

func splitYAMLDocuments(manifest string) []string {
	raw := strings.Split(manifest, "\n---")
	result := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func nodeReady(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func toProblemPodRecord(pod corev1.Pod) applicationcluster.PodHealthRecord {
	readyContainers := 0
	restartCount := int32(0)
	for _, status := range pod.Status.ContainerStatuses {
		if status.Ready {
			readyContainers++
		}
		restartCount += status.RestartCount
	}

	return applicationcluster.PodHealthRecord{
		Namespace:       pod.Namespace,
		Name:            pod.Name,
		Status:          string(pod.Status.Phase),
		NodeName:        pod.Spec.NodeName,
		ReadyContainers: readyContainers,
		TotalContainers: len(pod.Spec.Containers),
		RestartCount:    restartCount,
	}
}

func shouldShowProblemPod(item applicationcluster.PodHealthRecord) bool {
	if strings.EqualFold(item.Status, "Running") && item.ReadyContainers == item.TotalContainers && item.RestartCount == 0 {
		return false
	}
	return true
}

func toDeploymentHealthRecord(deployment appsv1.Deployment) applicationcluster.DeploymentHealthRecord {
	replicas := int32(1)
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	status := "NotReady"
	if deployment.Status.ReadyReplicas >= replicas && replicas > 0 {
		status = "Healthy"
	}
	if deployment.Spec.Paused {
		status = "Paused"
	}

	return applicationcluster.DeploymentHealthRecord{
		Namespace:         deployment.Namespace,
		Name:              deployment.Name,
		Status:            status,
		ReadyReplicas:     deployment.Status.ReadyReplicas,
		Replicas:          replicas,
		AvailableReplicas: deployment.Status.AvailableReplicas,
		UpdatedReplicas:   deployment.Status.UpdatedReplicas,
	}
}
