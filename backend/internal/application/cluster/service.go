package cluster

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

var ErrNotFound = errors.New("cluster not found")
var ErrInvalidResourceType = errors.New("invalid resource type")

// Connection 表示建立 K8s 客户端所需的集群连接信息。
type Connection struct {
	ID          int64
	Name        string
	APIserver   string
	AuthType    string
	KubeConfig  string
	Token       string
	CACert      string
	Credentials string
}

type PermissionRecord struct {
	ID        int64     `json:"id"`
	ClusterID int64     `json:"clusterId"`
	UserID    *int64    `json:"userId"`
	TeamID    *int64    `json:"teamId"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Record struct {
	ID             int64     `json:"id"`
	TenantID       *int64    `json:"tenantId"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	APIserver      string    `json:"apiServer"`
	Environment    string    `json:"environment"`
	AuthType       string    `json:"authType"`
	IsPublic       bool      `json:"isPublic"`
	Status         string    `json:"status"`
	OwnerUserID    *int64    `json:"ownerUserId"`
	HasKubeConfig  bool      `json:"hasKubeConfig"`
	HasToken       bool      `json:"hasToken"`
	HasCACert      bool      `json:"hasCaCert"`
	HasCredentials bool      `json:"hasCredentials"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

// ValidationRecord 表示集群连接校验结果。
type ValidationRecord struct {
	Reachable       bool      `json:"reachable"`
	Version         string    `json:"version"`
	NamespacesCount int       `json:"namespacesCount"`
	Message         string    `json:"message"`
	CheckedAt       time.Time `json:"checkedAt"`
}

// NamespaceRecord 表示命名空间摘要。
type NamespaceRecord struct {
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	Labels    map[string]string `json:"labels"`
	CreatedAt time.Time         `json:"createdAt"`
}

// ResourceQuery 表示基础资源查询条件。
type ResourceQuery struct {
	Type      string `json:"type"`
	Namespace string `json:"namespace"`
}

// ResourceRecord 表示基础资源摘要。
type ResourceRecord struct {
	Type      string            `json:"type"`
	Kind      string            `json:"kind"`
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Status    string            `json:"status"`
	Labels    map[string]string `json:"labels"`
	CreatedAt time.Time         `json:"createdAt"`
}

// ResourceDetail 表示单个资源的完整对象。
type ResourceDetail struct {
	Type      string         `json:"type"`
	Kind      string         `json:"kind"`
	Name      string         `json:"name"`
	Namespace string         `json:"namespace"`
	Object    map[string]any `json:"object"`
}

// EventRecord 表示 K8s 事件。
type EventRecord struct {
	Type           string     `json:"type"`
	Reason         string     `json:"reason"`
	Message        string     `json:"message"`
	Namespace      string     `json:"namespace"`
	InvolvedObject string     `json:"involvedObject"`
	Count          int32      `json:"count"`
	FirstSeenAt    *time.Time `json:"firstSeenAt"`
	LastSeenAt     *time.Time `json:"lastSeenAt"`
}

type ApplyResult struct {
	Summary   string   `json:"summary"`
	Resources []string `json:"resources"`
}

type PodHealthRecord struct {
	Namespace       string `json:"namespace"`
	Name            string `json:"name"`
	Status          string `json:"status"`
	NodeName        string `json:"nodeName"`
	ReadyContainers int    `json:"readyContainers"`
	TotalContainers int    `json:"totalContainers"`
	RestartCount    int32  `json:"restartCount"`
}

type DeploymentHealthRecord struct {
	Namespace         string `json:"namespace"`
	Name              string `json:"name"`
	Status            string `json:"status"`
	ReadyReplicas     int32  `json:"readyReplicas"`
	Replicas          int32  `json:"replicas"`
	AvailableReplicas int32  `json:"availableReplicas"`
	UpdatedReplicas   int32  `json:"updatedReplicas"`
}

type OverviewRecord struct {
	ClusterID            int64                    `json:"clusterId"`
	ClusterName          string                   `json:"clusterName"`
	NamespacesCount      int                      `json:"namespacesCount"`
	NodeCount            int                      `json:"nodeCount"`
	ReadyNodeCount       int                      `json:"readyNodeCount"`
	PodCount             int                      `json:"podCount"`
	RunningPodCount      int                      `json:"runningPodCount"`
	PendingPodCount      int                      `json:"pendingPodCount"`
	FailedPodCount       int                      `json:"failedPodCount"`
	DeploymentCount      int                      `json:"deploymentCount"`
	ReadyDeploymentCount int                      `json:"readyDeploymentCount"`
	ServiceCount         int                      `json:"serviceCount"`
	ProblemPods          []PodHealthRecord        `json:"problemPods"`
	Deployments          []DeploymentHealthRecord `json:"deployments"`
	RecentEvents         []EventRecord            `json:"recentEvents"`
	CollectedAt          time.Time                `json:"collectedAt"`
}

type PodLogQuery struct {
	Namespace    string `json:"namespace"`
	PodName      string `json:"podName"`
	Container    string `json:"container"`
	Follow       bool   `json:"follow"`
	TailLines    int64  `json:"tailLines"`
	SinceSeconds int64  `json:"sinceSeconds"`
}

type CreateInput struct {
	TenantID    *int64 `json:"tenantId"`
	Name        string `json:"name"`
	Description string `json:"description"`
	APIserver   string `json:"apiServer"`
	Environment string `json:"environment"`
	AuthType    string `json:"authType"`
	KubeConfig  string `json:"kubeConfig"`
	Token       string `json:"token"`
	CACert      string `json:"caCert"`
	Credentials string `json:"credentials"`
	IsPublic    bool   `json:"isPublic"`
	Status      string `json:"status"`
	OwnerUserID *int64 `json:"ownerUserId"`
}

type UpdateInput = CreateInput

type ShareInput struct {
	UserID *int64 `json:"userId"`
	TeamID *int64 `json:"teamId"`
	Role   string `json:"role"`
}

type Repository interface {
	List(ctx context.Context) ([]Record, error)
	Get(ctx context.Context, id int64) (*Record, error)
	GetConnection(ctx context.Context, id int64) (*Connection, error)
	Create(ctx context.Context, input CreateInput) (*Record, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*Record, error)
	Delete(ctx context.Context, id int64) error
	Share(ctx context.Context, clusterID int64, input ShareInput) (*PermissionRecord, error)
	ListPermissions(ctx context.Context, clusterID int64) ([]PermissionRecord, error)
}

type KubernetesGateway interface {
	Validate(ctx context.Context, connection Connection) (*ValidationRecord, error)
	GetOverview(ctx context.Context, connection Connection, namespace string) (*OverviewRecord, error)
	ListNamespaces(ctx context.Context, connection Connection) ([]NamespaceRecord, error)
	ListResources(ctx context.Context, connection Connection, query ResourceQuery) ([]ResourceRecord, error)
	GetResource(ctx context.Context, connection Connection, query ResourceQuery, name string) (*ResourceDetail, error)
	ListEvents(ctx context.Context, connection Connection, namespace string) ([]EventRecord, error)
	StreamPodLogs(ctx context.Context, connection Connection, query PodLogQuery) (io.ReadCloser, error)
	DeleteResource(ctx context.Context, connection Connection, query ResourceQuery, name string) error
	ScaleDeployment(ctx context.Context, connection Connection, namespace string, name string, replicas int32) error
	RestartDeployment(ctx context.Context, connection Connection, namespace string, name string) error
	ApplyYAML(ctx context.Context, connection Connection, manifest string) (*ApplyResult, error)
}

type Service struct {
	repo       Repository
	k8sGateway KubernetesGateway
}

func NewService(repo Repository, k8sGateway KubernetesGateway) *Service {
	return &Service{repo: repo, k8sGateway: k8sGateway}
}

func (s *Service) List(ctx context.Context) ([]Record, error) {
	return s.repo.List(ctx)
}

func (s *Service) Get(ctx context.Context, id int64) (*Record, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*Record, error) {
	return s.repo.Create(ctx, input)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*Record, error) {
	return s.repo.Update(ctx, id, input)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) Share(ctx context.Context, clusterID int64, input ShareInput) (*PermissionRecord, error) {
	return s.repo.Share(ctx, clusterID, input)
}

func (s *Service) ListPermissions(ctx context.Context, clusterID int64) ([]PermissionRecord, error) {
	return s.repo.ListPermissions(ctx, clusterID)
}

func (s *Service) Validate(ctx context.Context, clusterID int64) (*ValidationRecord, error) {
	connection, err := s.repo.GetConnection(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return s.k8sGateway.Validate(ctx, *connection)
}

func (s *Service) GetOverview(ctx context.Context, clusterID int64, namespace string) (*OverviewRecord, error) {
	connection, err := s.repo.GetConnection(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	record, err := s.k8sGateway.GetOverview(ctx, *connection, namespace)
	if err != nil {
		return nil, err
	}
	record.ClusterID = connection.ID
	record.ClusterName = connection.Name
	return record, nil
}

func (s *Service) ListNamespaces(ctx context.Context, clusterID int64) ([]NamespaceRecord, error) {
	connection, err := s.repo.GetConnection(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return s.k8sGateway.ListNamespaces(ctx, *connection)
}

func (s *Service) ListResources(ctx context.Context, clusterID int64, query ResourceQuery) ([]ResourceRecord, error) {
	if query.Type == "" {
		return nil, fmt.Errorf("resource type is required: %w", ErrInvalidResourceType)
	}

	connection, err := s.repo.GetConnection(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return s.k8sGateway.ListResources(ctx, *connection, query)
}

func (s *Service) GetResource(ctx context.Context, clusterID int64, query ResourceQuery, name string) (*ResourceDetail, error) {
	if query.Type == "" {
		return nil, fmt.Errorf("resource type is required: %w", ErrInvalidResourceType)
	}

	connection, err := s.repo.GetConnection(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return s.k8sGateway.GetResource(ctx, *connection, query, name)
}

func (s *Service) ListEvents(ctx context.Context, clusterID int64, namespace string) ([]EventRecord, error) {
	connection, err := s.repo.GetConnection(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return s.k8sGateway.ListEvents(ctx, *connection, namespace)
}

func (s *Service) StreamPodLogs(ctx context.Context, clusterID int64, query PodLogQuery) (io.ReadCloser, error) {
	connection, err := s.repo.GetConnection(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	if query.PodName == "" {
		return nil, fmt.Errorf("pod name is required")
	}
	return s.k8sGateway.StreamPodLogs(ctx, *connection, query)
}

func (s *Service) DeleteResource(ctx context.Context, clusterID int64, query ResourceQuery, name string) error {
	connection, err := s.repo.GetConnection(ctx, clusterID)
	if err != nil {
		return err
	}
	return s.k8sGateway.DeleteResource(ctx, *connection, query, name)
}

func (s *Service) ScaleDeployment(ctx context.Context, clusterID int64, namespace string, name string, replicas int32) error {
	connection, err := s.repo.GetConnection(ctx, clusterID)
	if err != nil {
		return err
	}
	return s.k8sGateway.ScaleDeployment(ctx, *connection, namespace, name, replicas)
}

func (s *Service) RestartDeployment(ctx context.Context, clusterID int64, namespace string, name string) error {
	connection, err := s.repo.GetConnection(ctx, clusterID)
	if err != nil {
		return err
	}
	return s.k8sGateway.RestartDeployment(ctx, *connection, namespace, name)
}

func (s *Service) ApplyYAML(ctx context.Context, clusterID int64, manifest string) (*ApplyResult, error) {
	connection, err := s.repo.GetConnection(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	return s.k8sGateway.ApplyYAML(ctx, *connection, manifest)
}
