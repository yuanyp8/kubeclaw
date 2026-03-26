package repository

import (
	"context"
	"errors"
	"fmt"

	appsecurity "kubeclaw/backend/internal/application/security"
	mysqlinfra "kubeclaw/backend/internal/infrastructure/mysql"

	"gorm.io/gorm"
)

type SecurityRepository struct {
	db *gorm.DB
}

func NewSecurityRepository(db *gorm.DB) *SecurityRepository {
	return &SecurityRepository{db: db}
}

func (r *SecurityRepository) ListIPWhitelists(ctx context.Context) ([]appsecurity.IPWhitelistRecord, error) {
	var models []mysqlinfra.IPWhitelistModel
	if err := r.db.WithContext(ctx).Order("id asc").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list ip whitelists: %w", err)
	}
	result := make([]appsecurity.IPWhitelistRecord, 0, len(models))
	for _, item := range models {
		result = append(result, appsecurity.IPWhitelistRecord{
			ID:          item.ID,
			TenantID:    item.TenantID,
			Name:        item.Name,
			IPOrCIDR:    item.IPOrCIDR,
			Scope:       item.Scope,
			Description: item.Description,
			IsEnabled:   item.IsEnabled,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}
	return result, nil
}

func (r *SecurityRepository) GetIPWhitelist(ctx context.Context, id int64) (*appsecurity.IPWhitelistRecord, error) {
	var model mysqlinfra.IPWhitelistModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, appsecurity.ErrNotFound
		}
		return nil, fmt.Errorf("get ip whitelist: %w", err)
	}
	record := appsecurity.IPWhitelistRecord{
		ID:          model.ID,
		TenantID:    model.TenantID,
		Name:        model.Name,
		IPOrCIDR:    model.IPOrCIDR,
		Scope:       model.Scope,
		Description: model.Description,
		IsEnabled:   model.IsEnabled,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
	return &record, nil
}

func (r *SecurityRepository) CreateIPWhitelist(ctx context.Context, input appsecurity.IPWhitelistInput) (*appsecurity.IPWhitelistRecord, error) {
	model := mysqlinfra.IPWhitelistModel{
		TenantID:    input.TenantID,
		Name:        input.Name,
		IPOrCIDR:    input.IPOrCIDR,
		Scope:       input.Scope,
		Description: input.Description,
		IsEnabled:   input.IsEnabled,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, fmt.Errorf("create ip whitelist: %w", err)
	}
	return r.GetIPWhitelist(ctx, model.ID)
}

func (r *SecurityRepository) UpdateIPWhitelist(ctx context.Context, id int64, input appsecurity.IPWhitelistInput) (*appsecurity.IPWhitelistRecord, error) {
	var model mysqlinfra.IPWhitelistModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, appsecurity.ErrNotFound
		}
		return nil, fmt.Errorf("load ip whitelist for update: %w", err)
	}
	model.TenantID = input.TenantID
	model.Name = input.Name
	model.IPOrCIDR = input.IPOrCIDR
	model.Scope = input.Scope
	model.Description = input.Description
	model.IsEnabled = input.IsEnabled
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, fmt.Errorf("update ip whitelist: %w", err)
	}
	return r.GetIPWhitelist(ctx, model.ID)
}

func (r *SecurityRepository) DeleteIPWhitelist(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&mysqlinfra.IPWhitelistModel{}, id).Error
}

func (r *SecurityRepository) ListSensitiveWords(ctx context.Context) ([]appsecurity.SensitiveWordRecord, error) {
	var models []mysqlinfra.SensitiveWordModel
	if err := r.db.WithContext(ctx).Order("id asc").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list sensitive words: %w", err)
	}
	result := make([]appsecurity.SensitiveWordRecord, 0, len(models))
	for _, item := range models {
		result = append(result, appsecurity.SensitiveWordRecord{
			ID:          item.ID,
			TenantID:    item.TenantID,
			Word:        item.Word,
			Category:    item.Category,
			Level:       item.Level,
			Action:      item.Action,
			Description: item.Description,
			IsEnabled:   item.IsEnabled,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}
	return result, nil
}

func (r *SecurityRepository) GetSensitiveWord(ctx context.Context, id int64) (*appsecurity.SensitiveWordRecord, error) {
	var model mysqlinfra.SensitiveWordModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, appsecurity.ErrNotFound
		}
		return nil, fmt.Errorf("get sensitive word: %w", err)
	}
	record := appsecurity.SensitiveWordRecord{
		ID:          model.ID,
		TenantID:    model.TenantID,
		Word:        model.Word,
		Category:    model.Category,
		Level:       model.Level,
		Action:      model.Action,
		Description: model.Description,
		IsEnabled:   model.IsEnabled,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
	return &record, nil
}

func (r *SecurityRepository) CreateSensitiveWord(ctx context.Context, input appsecurity.SensitiveWordInput) (*appsecurity.SensitiveWordRecord, error) {
	model := mysqlinfra.SensitiveWordModel{
		TenantID:    input.TenantID,
		Word:        input.Word,
		Category:    input.Category,
		Level:       input.Level,
		Action:      input.Action,
		Description: input.Description,
		IsEnabled:   input.IsEnabled,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, fmt.Errorf("create sensitive word: %w", err)
	}
	return r.GetSensitiveWord(ctx, model.ID)
}

func (r *SecurityRepository) UpdateSensitiveWord(ctx context.Context, id int64, input appsecurity.SensitiveWordInput) (*appsecurity.SensitiveWordRecord, error) {
	var model mysqlinfra.SensitiveWordModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, appsecurity.ErrNotFound
		}
		return nil, fmt.Errorf("load sensitive word for update: %w", err)
	}
	model.TenantID = input.TenantID
	model.Word = input.Word
	model.Category = input.Category
	model.Level = input.Level
	model.Action = input.Action
	model.Description = input.Description
	model.IsEnabled = input.IsEnabled
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, fmt.Errorf("update sensitive word: %w", err)
	}
	return r.GetSensitiveWord(ctx, model.ID)
}

func (r *SecurityRepository) DeleteSensitiveWord(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&mysqlinfra.SensitiveWordModel{}, id).Error
}

func (r *SecurityRepository) ListSensitiveFieldRules(ctx context.Context) ([]appsecurity.SensitiveFieldRuleRecord, error) {
	var models []mysqlinfra.SensitiveFieldRuleModel
	if err := r.db.WithContext(ctx).Order("id asc").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list sensitive field rules: %w", err)
	}
	result := make([]appsecurity.SensitiveFieldRuleRecord, 0, len(models))
	for _, item := range models {
		result = append(result, appsecurity.SensitiveFieldRuleRecord{
			ID:          item.ID,
			TenantID:    item.TenantID,
			Name:        item.Name,
			Resource:    item.Resource,
			FieldPath:   item.FieldPath,
			Action:      item.Action,
			Description: item.Description,
			IsEnabled:   item.IsEnabled,
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}
	return result, nil
}

func (r *SecurityRepository) GetSensitiveFieldRule(ctx context.Context, id int64) (*appsecurity.SensitiveFieldRuleRecord, error) {
	var model mysqlinfra.SensitiveFieldRuleModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, appsecurity.ErrNotFound
		}
		return nil, fmt.Errorf("get sensitive field rule: %w", err)
	}
	record := appsecurity.SensitiveFieldRuleRecord{
		ID:          model.ID,
		TenantID:    model.TenantID,
		Name:        model.Name,
		Resource:    model.Resource,
		FieldPath:   model.FieldPath,
		Action:      model.Action,
		Description: model.Description,
		IsEnabled:   model.IsEnabled,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
	return &record, nil
}

func (r *SecurityRepository) CreateSensitiveFieldRule(ctx context.Context, input appsecurity.SensitiveFieldRuleInput) (*appsecurity.SensitiveFieldRuleRecord, error) {
	model := mysqlinfra.SensitiveFieldRuleModel{
		TenantID:    input.TenantID,
		Name:        input.Name,
		Resource:    input.Resource,
		FieldPath:   input.FieldPath,
		Action:      input.Action,
		Description: input.Description,
		IsEnabled:   input.IsEnabled,
	}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, fmt.Errorf("create sensitive field rule: %w", err)
	}
	return r.GetSensitiveFieldRule(ctx, model.ID)
}

func (r *SecurityRepository) UpdateSensitiveFieldRule(ctx context.Context, id int64, input appsecurity.SensitiveFieldRuleInput) (*appsecurity.SensitiveFieldRuleRecord, error) {
	var model mysqlinfra.SensitiveFieldRuleModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, appsecurity.ErrNotFound
		}
		return nil, fmt.Errorf("load sensitive field rule for update: %w", err)
	}
	model.TenantID = input.TenantID
	model.Name = input.Name
	model.Resource = input.Resource
	model.FieldPath = input.FieldPath
	model.Action = input.Action
	model.Description = input.Description
	model.IsEnabled = input.IsEnabled
	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, fmt.Errorf("update sensitive field rule: %w", err)
	}
	return r.GetSensitiveFieldRule(ctx, model.ID)
}

func (r *SecurityRepository) DeleteSensitiveFieldRule(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&mysqlinfra.SensitiveFieldRuleModel{}, id).Error
}
