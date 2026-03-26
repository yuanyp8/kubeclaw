package repository

import (
	"context"
	"errors"
	"fmt"

	applicationtenant "kubeclaw/backend/internal/application/tenant"
	mysqlinfra "kubeclaw/backend/internal/infrastructure/mysql"

	"gorm.io/gorm"
)

type TenantRepository struct {
	db *gorm.DB
}

func NewTenantRepository(db *gorm.DB) *TenantRepository {
	return &TenantRepository{db: db}
}

func (r *TenantRepository) List(ctx context.Context) ([]applicationtenant.Record, error) {
	var models []mysqlinfra.TenantModel
	if err := r.db.WithContext(ctx).Order("id asc").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}

	return r.buildTenantRecords(ctx, models)
}

func (r *TenantRepository) Get(ctx context.Context, id int64) (*applicationtenant.Record, error) {
	var model mysqlinfra.TenantModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, applicationtenant.ErrNotFound
		}
		return nil, fmt.Errorf("get tenant: %w", err)
	}

	items, err := r.buildTenantRecords(ctx, []mysqlinfra.TenantModel{model})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, applicationtenant.ErrNotFound
	}
	return &items[0], nil
}

func (r *TenantRepository) Create(ctx context.Context, input applicationtenant.Input) (*applicationtenant.Record, error) {
	model := mysqlinfra.TenantModel{
		Name:        input.Name,
		Slug:        input.Slug,
		Description: input.Description,
		Status:      input.Status,
		IsSystem:    input.IsSystem,
		OwnerUserID: input.OwnerUserID,
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}

	return r.Get(ctx, model.ID)
}

func (r *TenantRepository) Update(ctx context.Context, id int64, input applicationtenant.Input) (*applicationtenant.Record, error) {
	var model mysqlinfra.TenantModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, applicationtenant.ErrNotFound
		}
		return nil, fmt.Errorf("load tenant for update: %w", err)
	}

	model.Name = input.Name
	model.Slug = input.Slug
	model.Description = input.Description
	model.Status = input.Status
	model.IsSystem = input.IsSystem
	model.OwnerUserID = input.OwnerUserID

	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, fmt.Errorf("update tenant: %w", err)
	}

	return r.Get(ctx, model.ID)
}

func (r *TenantRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&mysqlinfra.UserModel{}).Where("tenant_id = ?", id).Update("tenant_id", nil).Error; err != nil {
			return fmt.Errorf("clear tenant from users: %w", err)
		}
		if err := tx.Model(&mysqlinfra.TeamModel{}).Where("tenant_id = ?", id).Update("tenant_id", nil).Error; err != nil {
			return fmt.Errorf("clear tenant from teams: %w", err)
		}
		if err := tx.Delete(&mysqlinfra.TenantModel{}, id).Error; err != nil {
			return fmt.Errorf("delete tenant: %w", err)
		}
		return nil
	})
}

func (r *TenantRepository) buildTenantRecords(ctx context.Context, tenants []mysqlinfra.TenantModel) ([]applicationtenant.Record, error) {
	if len(tenants) == 0 {
		return []applicationtenant.Record{}, nil
	}

	tenantIDs := make([]int64, 0, len(tenants))
	for _, item := range tenants {
		tenantIDs = append(tenantIDs, item.ID)
	}

	userCounts, err := r.loadCounts(ctx, "users", tenantIDs)
	if err != nil {
		return nil, err
	}
	teamCounts, err := r.loadCounts(ctx, "teams", tenantIDs)
	if err != nil {
		return nil, err
	}

	result := make([]applicationtenant.Record, 0, len(tenants))
	for _, item := range tenants {
		result = append(result, applicationtenant.Record{
			ID:          item.ID,
			Name:        item.Name,
			Slug:        item.Slug,
			Description: item.Description,
			Status:      item.Status,
			IsSystem:    item.IsSystem,
			OwnerUserID: item.OwnerUserID,
			UserCount:   userCounts[item.ID],
			TeamCount:   teamCounts[item.ID],
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}

	return result, nil
}

func (r *TenantRepository) loadCounts(ctx context.Context, table string, tenantIDs []int64) (map[int64]int, error) {
	result := make(map[int64]int)
	if len(tenantIDs) == 0 {
		return result, nil
	}

	var rows []struct {
		TenantID int64
		Count    int64
	}
	if err := r.db.WithContext(ctx).
		Table(table).
		Select("tenant_id, count(*) AS count").
		Where("tenant_id IN ? AND deleted_at IS NULL", tenantIDs).
		Group("tenant_id").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load %s tenant counts: %w", table, err)
	}

	for _, row := range rows {
		result[row.TenantID] = int(row.Count)
	}

	return result, nil
}
