package mysql

import (
	"fmt"

	"gorm.io/gorm"
)

// AutoMigrate 将当前阶段所有核心表结构同步到数据库。
func AutoMigrate(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&TenantModel{},
		&UserModel{},
		&TeamModel{},
		&UserTeamModel{},
		&AIModelConfigModel{},
		&ClusterModel{},
		&ClusterPermissionModel{},
		&MCPServerModel{},
		&SkillModel{},
		&KnowledgeBaseModel{},
		&ScheduledTaskModel{},
		&ChatSessionModel{},
		&ChatMessageModel{},
		&ToolExecutionModel{},
		&AgentRunModel{},
		&AgentEventModel{},
		&ApprovalRequestModel{},
		&AuditLogModel{},
		&IPWhitelistModel{},
		&SensitiveWordModel{},
		&SensitiveFieldRuleModel{},
	); err != nil {
		return fmt.Errorf("auto migrate tables: %w", err)
	}

	return nil
}
