package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	appskill "kubeclaw/backend/internal/application/skill"
	mysqlinfra "kubeclaw/backend/internal/infrastructure/mysql"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type SkillRepository struct {
	db *gorm.DB
}

func NewSkillRepository(db *gorm.DB) *SkillRepository {
	return &SkillRepository{db: db}
}

func (r *SkillRepository) List(ctx context.Context) ([]appskill.Record, error) {
	var models []mysqlinfra.SkillModel
	if err := r.db.WithContext(ctx).Order("id asc").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list skills: %w", err)
	}

	result := make([]appskill.Record, 0, len(models))
	for _, item := range models {
		result = append(result, toSkillRecord(item))
	}
	return result, nil
}

func (r *SkillRepository) Get(ctx context.Context, id int64) (*appskill.Record, error) {
	var model mysqlinfra.SkillModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, appskill.ErrNotFound
		}
		return nil, fmt.Errorf("get skill: %w", err)
	}
	record := toSkillRecord(model)
	return &record, nil
}

func (r *SkillRepository) Create(ctx context.Context, input appskill.CreateInput) (*appskill.Record, error) {
	model := mysqlinfra.SkillModel{
		TenantID:       input.TenantID,
		Name:           input.Name,
		Description:    input.Description,
		Type:           input.Type,
		Version:        input.Version,
		Status:         input.Status,
		DefinitionJSON: datatypes.JSON(input.Definition),
		IsPublic:       input.IsPublic,
		CreatorID:      input.CreatorID,
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, fmt.Errorf("create skill: %w", err)
	}

	return r.Get(ctx, model.ID)
}

func (r *SkillRepository) Update(ctx context.Context, id int64, input appskill.UpdateInput) (*appskill.Record, error) {
	var model mysqlinfra.SkillModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, appskill.ErrNotFound
		}
		return nil, fmt.Errorf("load skill for update: %w", err)
	}

	model.TenantID = input.TenantID
	model.Name = input.Name
	model.Description = input.Description
	model.Type = input.Type
	model.Version = input.Version
	model.Status = input.Status
	model.DefinitionJSON = datatypes.JSON(input.Definition)
	model.IsPublic = input.IsPublic
	model.CreatorID = input.CreatorID

	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, fmt.Errorf("update skill: %w", err)
	}

	return r.Get(ctx, model.ID)
}

func (r *SkillRepository) Delete(ctx context.Context, id int64) error {
	if err := r.db.WithContext(ctx).Delete(&mysqlinfra.SkillModel{}, id).Error; err != nil {
		return fmt.Errorf("delete skill: %w", err)
	}
	return nil
}

func toSkillRecord(model mysqlinfra.SkillModel) appskill.Record {
	return appskill.Record{
		ID:          model.ID,
		TenantID:    model.TenantID,
		Name:        model.Name,
		Description: model.Description,
		Type:        model.Type,
		Version:     model.Version,
		Status:      model.Status,
		Definition:  json.RawMessage(model.DefinitionJSON),
		IsPublic:    model.IsPublic,
		CreatorID:   model.CreatorID,
		CreatedAt:   model.CreatedAt,
		UpdatedAt:   model.UpdatedAt,
	}
}
