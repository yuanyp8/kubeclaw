package repository

import (
	"context"
	"errors"
	"fmt"

	applicationaudit "kubeclaw/backend/internal/application/audit"
	mysqlinfra "kubeclaw/backend/internal/infrastructure/mysql"

	"gorm.io/gorm"
)

// AuditRepository 提供审计日志的 MySQL 持久化实现。
type AuditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) List(ctx context.Context) ([]applicationaudit.Record, error) {
	var models []mysqlinfra.AuditLogModel
	if err := r.db.WithContext(ctx).Order("id desc").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}

	result := make([]applicationaudit.Record, 0, len(models))
	for _, item := range models {
		result = append(result, toAuditRecord(item))
	}
	return result, nil
}

func (r *AuditRepository) Get(ctx context.Context, id int64) (*applicationaudit.Record, error) {
	var model mysqlinfra.AuditLogModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, applicationaudit.ErrNotFound
		}
		return nil, fmt.Errorf("get audit log: %w", err)
	}

	record := toAuditRecord(model)
	return &record, nil
}

func (r *AuditRepository) Create(ctx context.Context, input applicationaudit.CreateInput) (*applicationaudit.Record, error) {
	model := mysqlinfra.AuditLogModel{
		TenantID: input.TenantID,
		UserID:   input.UserID,
		Action:   input.Action,
		Target:   input.Target,
		Details:  input.Details,
		IP:       input.IP,
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, fmt.Errorf("create audit log: %w", err)
	}

	record := toAuditRecord(model)
	return &record, nil
}

func toAuditRecord(model mysqlinfra.AuditLogModel) applicationaudit.Record {
	return applicationaudit.Record{
		ID:        model.ID,
		TenantID:  model.TenantID,
		UserID:    model.UserID,
		Action:    model.Action,
		Target:    model.Target,
		Details:   model.Details,
		IP:        model.IP,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
}
