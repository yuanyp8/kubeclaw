package mysql

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type TenantModel struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	Name        string `gorm:"size:100;not null"`
	Slug        string `gorm:"size:100;not null;uniqueIndex"`
	Description string `gorm:"type:text"`
	Status      string `gorm:"size:20;not null;default:active"`
	IsSystem    bool   `gorm:"not null;default:false"`
	OwnerUserID *int64 `gorm:"index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (TenantModel) TableName() string { return "tenants" }

type UserModel struct {
	ID           int64  `gorm:"primaryKey;autoIncrement"`
	TenantID     *int64 `gorm:"index"`
	Username     string `gorm:"size:50;not null;uniqueIndex"`
	Email        string `gorm:"size:100;not null;uniqueIndex"`
	DisplayName  string `gorm:"size:100"`
	Phone        string `gorm:"size:30"`
	AvatarURL    string `gorm:"size:255"`
	PasswordHash string `gorm:"size:255;not null"`
	Role         string `gorm:"size:20;not null;default:user"`
	Status       string `gorm:"size:20;not null;default:active"`
	LastLoginAt  *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

func (UserModel) TableName() string { return "users" }

type TeamModel struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	TenantID    *int64 `gorm:"index"`
	Name        string `gorm:"size:100;not null"`
	Description string `gorm:"type:text"`
	OwnerUserID *int64 `gorm:"index"`
	Visibility  string `gorm:"size:20;not null;default:private"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (TeamModel) TableName() string { return "teams" }

type UserTeamModel struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	UserID    int64  `gorm:"not null;index"`
	TeamID    int64  `gorm:"not null;index"`
	Role      string `gorm:"size:20;not null;default:member"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (UserTeamModel) TableName() string { return "user_teams" }

type AIModelConfigModel struct {
	ID               int64          `gorm:"primaryKey;autoIncrement"`
	TenantID         *int64         `gorm:"index"`
	Name             string         `gorm:"size:100;not null"`
	Provider         string         `gorm:"size:50;not null"`
	Model            string         `gorm:"size:100;not null"`
	BaseURL          string         `gorm:"size:255"`
	APIKeyEncrypted  string         `gorm:"type:text"`
	Description      string         `gorm:"type:text"`
	CapabilitiesJSON datatypes.JSON `gorm:"type:json"`
	IsDefault        bool           `gorm:"not null;default:false"`
	IsEnabled        bool           `gorm:"not null;default:true"`
	MaxTokens        int
	Temperature      float64
	TopP             float64
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        gorm.DeletedAt `gorm:"index"`
}

func (AIModelConfigModel) TableName() string { return "ai_model_configs" }

type ClusterModel struct {
	ID                   int64  `gorm:"primaryKey;autoIncrement"`
	TenantID             *int64 `gorm:"index"`
	Name                 string `gorm:"size:100;not null"`
	Description          string `gorm:"type:text"`
	APIserver            string `gorm:"size:255;not null"`
	Environment          string `gorm:"size:30;not null;default:prod"`
	AuthType             string `gorm:"size:30;not null"`
	KubeConfigEncrypted  string `gorm:"type:longtext"`
	TokenEncrypted       string `gorm:"type:text"`
	CACertEncrypted      string `gorm:"type:longtext"`
	CredentialsEncrypted string `gorm:"type:longtext"`
	OwnerUserID          *int64 `gorm:"index"`
	IsPublic             bool   `gorm:"not null;default:false"`
	Status               string `gorm:"size:20;not null;default:active"`
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            gorm.DeletedAt `gorm:"index"`
}

func (ClusterModel) TableName() string { return "clusters" }

type ClusterPermissionModel struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	ClusterID int64  `gorm:"not null;index"`
	UserID    *int64 `gorm:"index"`
	TeamID    *int64 `gorm:"index"`
	Role      string `gorm:"size:20;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (ClusterPermissionModel) TableName() string { return "cluster_permissions" }

type MCPServerModel struct {
	ID              int64          `gorm:"primaryKey;autoIncrement"`
	TenantID        *int64         `gorm:"index"`
	Name            string         `gorm:"size:100;not null"`
	Type            string         `gorm:"size:30;not null;default:custom"`
	Transport       string         `gorm:"size:30;not null;default:http"`
	Endpoint        string         `gorm:"size:255"`
	Command         string         `gorm:"size:255"`
	ArgsJSON        datatypes.JSON `gorm:"type:json"`
	HeadersJSON     datatypes.JSON `gorm:"type:json"`
	AuthType        string         `gorm:"size:30;default:none"`
	SecretEncrypted string         `gorm:"type:text"`
	Description     string         `gorm:"type:text"`
	HealthStatus    string         `gorm:"size:20;not null;default:unknown"`
	IsEnabled       bool           `gorm:"not null;default:true"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (MCPServerModel) TableName() string { return "mcp_servers" }

type SkillModel struct {
	ID             int64          `gorm:"primaryKey;autoIncrement"`
	TenantID       *int64         `gorm:"index"`
	Name           string         `gorm:"size:100;not null"`
	Description    string         `gorm:"type:text"`
	Type           string         `gorm:"size:30;not null"`
	Version        int            `gorm:"not null;default:1"`
	Status         string         `gorm:"size:20;not null;default:draft"`
	DefinitionJSON datatypes.JSON `gorm:"type:json"`
	IsPublic       bool           `gorm:"not null;default:false"`
	CreatorID      *int64         `gorm:"index"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (SkillModel) TableName() string { return "skills" }

type KnowledgeBaseModel struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	TenantID  *int64 `gorm:"index"`
	Name      string `gorm:"size:100;not null"`
	OwnerID   *int64 `gorm:"index"`
	ClusterID *int64 `gorm:"index"`
	IsPublic  bool   `gorm:"not null;default:false"`
	FilePath  string `gorm:"size:255"`
	FileType  string `gorm:"size:20"`
	Status    string `gorm:"size:20;not null;default:processing"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (KnowledgeBaseModel) TableName() string { return "knowledge_bases" }

type ScheduledTaskModel struct {
	ID         int64          `gorm:"primaryKey;autoIncrement"`
	TenantID   *int64         `gorm:"index"`
	Name       string         `gorm:"size:100;not null"`
	Cron       string         `gorm:"size:50;not null"`
	ClusterID  *int64         `gorm:"index"`
	Action     string         `gorm:"size:20;not null"`
	Status     string         `gorm:"size:20;not null;default:active"`
	ParamsJSON datatypes.JSON `gorm:"type:json"`
	LastRunAt  *time.Time
	NextRunAt  *time.Time
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}

func (ScheduledTaskModel) TableName() string { return "scheduled_tasks" }

type ChatSessionModel struct {
	ID          int64          `gorm:"primaryKey;autoIncrement"`
	TenantID    *int64         `gorm:"index"`
	UserID      int64          `gorm:"not null;index"`
	Title       string         `gorm:"size:200"`
	ContextJSON datatypes.JSON `gorm:"type:json"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (ChatSessionModel) TableName() string { return "chat_sessions" }

type ChatMessageModel struct {
	ID            int64          `gorm:"primaryKey;autoIncrement"`
	SessionID     int64          `gorm:"not null;index"`
	Role          string         `gorm:"size:20;not null"`
	Content       string         `gorm:"type:longtext"`
	ToolCallsJSON datatypes.JSON `gorm:"type:json"`
	ToolCallID    string         `gorm:"size:100"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (ChatMessageModel) TableName() string { return "chat_messages" }

type ToolExecutionModel struct {
	ID             int64          `gorm:"primaryKey;autoIncrement"`
	RunID          *int64         `gorm:"index"`
	UserID         *int64         `gorm:"index"`
	ClusterID      *int64         `gorm:"index"`
	ToolName       string         `gorm:"size:100;not null"`
	ParametersJSON datatypes.JSON `gorm:"type:json"`
	Result         string         `gorm:"type:longtext"`
	Status         string         `gorm:"size:20;not null;default:running"`
	DurationMS     int
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (ToolExecutionModel) TableName() string { return "tool_executions" }

type AgentRunModel struct {
	ID                 int64          `gorm:"primaryKey;autoIncrement"`
	SessionID          int64          `gorm:"not null;index"`
	UserID             int64          `gorm:"not null;index"`
	ModelID            *int64         `gorm:"index"`
	ClusterID          *int64         `gorm:"index"`
	Status             string         `gorm:"size:30;not null;default:running"`
	UserMessageID      *int64         `gorm:"index"`
	AssistantMessageID *int64         `gorm:"index"`
	Input              string         `gorm:"type:longtext"`
	Output             string         `gorm:"type:longtext"`
	ErrorMessage       string         `gorm:"type:text"`
	ContextJSON        datatypes.JSON `gorm:"type:json"`
	StartedAt          *time.Time
	FinishedAt         *time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
	DeletedAt          gorm.DeletedAt `gorm:"index"`
}

func (AgentRunModel) TableName() string { return "agent_runs" }

type AgentEventModel struct {
	ID          int64          `gorm:"primaryKey;autoIncrement"`
	RunID       int64          `gorm:"not null;index"`
	SessionID   int64          `gorm:"not null;index"`
	EventType   string         `gorm:"size:50;not null;index"`
	Role        string         `gorm:"size:50"`
	Status      string         `gorm:"size:30"`
	Message     string         `gorm:"type:longtext"`
	PayloadJSON datatypes.JSON `gorm:"type:json"`
	RequestID   string         `gorm:"size:100;index"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (AgentEventModel) TableName() string { return "agent_events" }

type ApprovalRequestModel struct {
	ID          int64          `gorm:"primaryKey;autoIncrement"`
	RunID       int64          `gorm:"not null;index"`
	SessionID   int64          `gorm:"not null;index"`
	UserID      int64          `gorm:"not null;index"`
	Type        string         `gorm:"size:50;not null"`
	Title       string         `gorm:"size:200;not null"`
	Reason      string         `gorm:"type:text"`
	Status      string         `gorm:"size:30;not null;default:pending"`
	PayloadJSON datatypes.JSON `gorm:"type:json"`
	ApprovedBy  *int64         `gorm:"index"`
	ResolvedAt  *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (ApprovalRequestModel) TableName() string { return "approval_requests" }

type AuditLogModel struct {
	ID        int64  `gorm:"primaryKey;autoIncrement"`
	TenantID  *int64 `gorm:"index"`
	UserID    *int64 `gorm:"index"`
	Action    string `gorm:"size:50;not null"`
	Target    string `gorm:"size:200"`
	Details   string `gorm:"type:longtext"`
	IP        string `gorm:"size:45"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (AuditLogModel) TableName() string { return "audit_logs" }

type IPWhitelistModel struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	TenantID    *int64 `gorm:"index"`
	Name        string `gorm:"size:100;not null"`
	IPOrCIDR    string `gorm:"size:100;not null"`
	Scope       string `gorm:"size:30;not null;default:global"`
	Description string `gorm:"type:text"`
	IsEnabled   bool   `gorm:"not null;default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (IPWhitelistModel) TableName() string { return "ip_whitelists" }

type SensitiveWordModel struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	TenantID    *int64 `gorm:"index"`
	Word        string `gorm:"size:100;not null;uniqueIndex"`
	Category    string `gorm:"size:50;not null;default:command"`
	Level       string `gorm:"size:20;not null;default:medium"`
	Action      string `gorm:"size:20;not null;default:review"`
	Description string `gorm:"type:text"`
	IsEnabled   bool   `gorm:"not null;default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (SensitiveWordModel) TableName() string { return "sensitive_words" }

type SensitiveFieldRuleModel struct {
	ID          int64  `gorm:"primaryKey;autoIncrement"`
	TenantID    *int64 `gorm:"index"`
	Name        string `gorm:"size:100;not null"`
	Resource    string `gorm:"size:50;not null"`
	FieldPath   string `gorm:"size:255;not null"`
	Action      string `gorm:"size:20;not null;default:mask"`
	Description string `gorm:"type:text"`
	IsEnabled   bool   `gorm:"not null;default:true"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (SensitiveFieldRuleModel) TableName() string { return "sensitive_field_rules" }
