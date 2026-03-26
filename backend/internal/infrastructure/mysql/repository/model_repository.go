package repository

import (
	"context"
	"errors"
	"fmt"

	applicationmodel "kubeclaw/backend/internal/application/model"
	cryptoinfra "kubeclaw/backend/internal/infrastructure/crypto"
	mysqlinfra "kubeclaw/backend/internal/infrastructure/mysql"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ModelRepository struct {
	db        *gorm.DB
	secretBox *cryptoinfra.SecretBox
}

func NewModelRepository(db *gorm.DB, secretBox *cryptoinfra.SecretBox) *ModelRepository {
	return &ModelRepository{db: db, secretBox: secretBox}
}

func (r *ModelRepository) List(ctx context.Context) ([]applicationmodel.Record, error) {
	var models []mysqlinfra.AIModelConfigModel
	if err := r.db.WithContext(ctx).Order("is_default desc, id asc").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list model configs: %w", err)
	}

	result := make([]applicationmodel.Record, 0, len(models))
	for _, item := range models {
		result = append(result, r.toRecord(item))
	}
	return result, nil
}

func (r *ModelRepository) Get(ctx context.Context, id int64) (*applicationmodel.Record, error) {
	model, err := r.load(ctx, id)
	if err != nil {
		return nil, err
	}

	record := r.toRecord(*model)
	return &record, nil
}

func (r *ModelRepository) GetDefault(ctx context.Context) (*applicationmodel.Record, error) {
	model, err := r.loadDefault(ctx)
	if err != nil {
		return nil, err
	}

	record := r.toRecord(*model)
	return &record, nil
}

func (r *ModelRepository) Resolve(ctx context.Context, id int64) (*applicationmodel.ResolvedRecord, error) {
	model, err := r.load(ctx, id)
	if err != nil {
		return nil, err
	}

	record := r.toResolvedRecord(*model)
	return &record, nil
}

func (r *ModelRepository) ResolveDefault(ctx context.Context) (*applicationmodel.ResolvedRecord, error) {
	model, err := r.loadDefault(ctx)
	if err != nil {
		return nil, err
	}

	record := r.toResolvedRecord(*model)
	return &record, nil
}

func (r *ModelRepository) Create(ctx context.Context, input applicationmodel.CreateInput) (*applicationmodel.Record, error) {
	cipherText, err := r.secretBox.Encrypt(input.APIKey)
	if err != nil {
		return nil, fmt.Errorf("encrypt api key: %w", err)
	}

	capabilities, err := marshalJSON(input.Capabilities)
	if err != nil {
		return nil, fmt.Errorf("marshal capabilities: %w", err)
	}

	model := mysqlinfra.AIModelConfigModel{
		TenantID:         input.TenantID,
		Name:             input.Name,
		Provider:         input.Provider,
		Model:            input.Model,
		BaseURL:          input.BaseURL,
		APIKeyEncrypted:  cipherText,
		Description:      input.Description,
		CapabilitiesJSON: datatypes.JSON(capabilities),
		IsDefault:        input.IsDefault,
		IsEnabled:        input.IsEnabled,
		MaxTokens:        input.MaxTokens,
		Temperature:      input.Temperature,
		TopP:             input.TopP,
	}

	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if input.IsDefault {
			if err := tx.Model(&mysqlinfra.AIModelConfigModel{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Create(&model).Error
	}); err != nil {
		return nil, fmt.Errorf("create model config: %w", err)
	}

	return r.Get(ctx, model.ID)
}

func (r *ModelRepository) Update(ctx context.Context, id int64, input applicationmodel.UpdateInput) (*applicationmodel.Record, error) {
	model, err := r.load(ctx, id)
	if err != nil {
		return nil, err
	}

	capabilities, err := marshalJSON(input.Capabilities)
	if err != nil {
		return nil, fmt.Errorf("marshal capabilities: %w", err)
	}

	model.TenantID = input.TenantID
	model.Name = input.Name
	model.Provider = input.Provider
	model.Model = input.Model
	model.BaseURL = input.BaseURL
	model.Description = input.Description
	model.CapabilitiesJSON = datatypes.JSON(capabilities)
	model.IsDefault = input.IsDefault
	model.IsEnabled = input.IsEnabled
	model.MaxTokens = input.MaxTokens
	model.Temperature = input.Temperature
	model.TopP = input.TopP

	if input.APIKey != "" {
		cipherText, encryptErr := r.secretBox.Encrypt(input.APIKey)
		if encryptErr != nil {
			return nil, fmt.Errorf("encrypt api key during update: %w", encryptErr)
		}
		model.APIKeyEncrypted = cipherText
	}

	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if model.IsDefault {
			if err := tx.Model(&mysqlinfra.AIModelConfigModel{}).
				Where("id <> ? AND is_default = ?", model.ID, true).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return tx.Save(model).Error
	}); err != nil {
		return nil, fmt.Errorf("update model config: %w", err)
	}

	return r.Get(ctx, model.ID)
}

func (r *ModelRepository) Delete(ctx context.Context, id int64) error {
	if err := r.db.WithContext(ctx).Delete(&mysqlinfra.AIModelConfigModel{}, id).Error; err != nil {
		return fmt.Errorf("delete model config: %w", err)
	}

	return nil
}

func (r *ModelRepository) SetDefault(ctx context.Context, id int64) (*applicationmodel.Record, error) {
	model, err := r.load(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&mysqlinfra.AIModelConfigModel{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
			return err
		}
		model.IsDefault = true
		return tx.Save(model).Error
	}); err != nil {
		return nil, fmt.Errorf("set default model: %w", err)
	}

	return r.Get(ctx, id)
}

func (r *ModelRepository) load(ctx context.Context, id int64) (*mysqlinfra.AIModelConfigModel, error) {
	var model mysqlinfra.AIModelConfigModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, applicationmodel.ErrNotFound
		}
		return nil, fmt.Errorf("get model config: %w", err)
	}
	return &model, nil
}

func (r *ModelRepository) loadDefault(ctx context.Context) (*mysqlinfra.AIModelConfigModel, error) {
	var model mysqlinfra.AIModelConfigModel
	if err := r.db.WithContext(ctx).Where("is_default = ?", true).Order("id asc").First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, applicationmodel.ErrNotFound
		}
		return nil, fmt.Errorf("get default model config: %w", err)
	}
	return &model, nil
}

func (r *ModelRepository) toRecord(model mysqlinfra.AIModelConfigModel) applicationmodel.Record {
	plain, _ := r.secretBox.Decrypt(model.APIKeyEncrypted)
	return applicationmodel.Record{
		ID:           model.ID,
		TenantID:     model.TenantID,
		Name:         model.Name,
		Provider:     model.Provider,
		Model:        model.Model,
		BaseURL:      model.BaseURL,
		Description:  model.Description,
		Capabilities: unmarshalStringSlice(model.CapabilitiesJSON),
		IsDefault:    model.IsDefault,
		IsEnabled:    model.IsEnabled,
		MaxTokens:    model.MaxTokens,
		Temperature:  model.Temperature,
		TopP:         model.TopP,
		HasAPIKey:    model.APIKeyEncrypted != "",
		MaskedAPIKey: cryptoinfra.MaskSecret(plain),
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}

func (r *ModelRepository) toResolvedRecord(model mysqlinfra.AIModelConfigModel) applicationmodel.ResolvedRecord {
	plain, _ := r.secretBox.Decrypt(model.APIKeyEncrypted)
	return applicationmodel.ResolvedRecord{
		Record: applicationmodel.Record{
			ID:           model.ID,
			TenantID:     model.TenantID,
			Name:         model.Name,
			Provider:     model.Provider,
			Model:        model.Model,
			BaseURL:      model.BaseURL,
			Description:  model.Description,
			Capabilities: unmarshalStringSlice(model.CapabilitiesJSON),
			IsDefault:    model.IsDefault,
			IsEnabled:    model.IsEnabled,
			MaxTokens:    model.MaxTokens,
			Temperature:  model.Temperature,
			TopP:         model.TopP,
			HasAPIKey:    model.APIKeyEncrypted != "",
			MaskedAPIKey: cryptoinfra.MaskSecret(plain),
			CreatedAt:    model.CreatedAt,
			UpdatedAt:    model.UpdatedAt,
		},
		APIKey: plain,
	}
}
