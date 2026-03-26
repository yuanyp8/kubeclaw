package repository

import (
	"context"
	"errors"
	"fmt"

	applicationmcp "kubeclaw/backend/internal/application/mcp"
	cryptoinfra "kubeclaw/backend/internal/infrastructure/crypto"
	mysqlinfra "kubeclaw/backend/internal/infrastructure/mysql"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type MCPRepository struct {
	db        *gorm.DB
	secretBox *cryptoinfra.SecretBox
}

func NewMCPRepository(db *gorm.DB, secretBox *cryptoinfra.SecretBox) *MCPRepository {
	return &MCPRepository{db: db, secretBox: secretBox}
}

func (r *MCPRepository) List(ctx context.Context) ([]applicationmcp.Record, error) {
	var models []mysqlinfra.MCPServerModel
	if err := r.db.WithContext(ctx).Order("id asc").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list mcp servers: %w", err)
	}

	result := make([]applicationmcp.Record, 0, len(models))
	for _, item := range models {
		result = append(result, r.toRecord(item))
	}
	return result, nil
}

func (r *MCPRepository) Get(ctx context.Context, id int64) (*applicationmcp.Record, error) {
	var model mysqlinfra.MCPServerModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, applicationmcp.ErrNotFound
		}
		return nil, fmt.Errorf("get mcp server: %w", err)
	}

	record := r.toRecord(model)
	return &record, nil
}

func (r *MCPRepository) Create(ctx context.Context, input applicationmcp.CreateInput) (*applicationmcp.Record, error) {
	args, err := marshalJSON(input.Args)
	if err != nil {
		return nil, fmt.Errorf("marshal mcp args: %w", err)
	}
	headers, err := marshalJSON(input.Headers)
	if err != nil {
		return nil, fmt.Errorf("marshal mcp headers: %w", err)
	}
	secret, err := r.secretBox.Encrypt(input.Secret)
	if err != nil {
		return nil, fmt.Errorf("encrypt mcp secret: %w", err)
	}

	model := mysqlinfra.MCPServerModel{
		TenantID:        input.TenantID,
		Name:            input.Name,
		Type:            input.Type,
		Transport:       input.Transport,
		Endpoint:        input.Endpoint,
		Command:         input.Command,
		ArgsJSON:        datatypes.JSON(args),
		HeadersJSON:     datatypes.JSON(headers),
		AuthType:        input.AuthType,
		SecretEncrypted: secret,
		Description:     input.Description,
		HealthStatus:    input.HealthStatus,
		IsEnabled:       input.IsEnabled,
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, fmt.Errorf("create mcp server: %w", err)
	}

	return r.Get(ctx, model.ID)
}

func (r *MCPRepository) Update(ctx context.Context, id int64, input applicationmcp.UpdateInput) (*applicationmcp.Record, error) {
	var model mysqlinfra.MCPServerModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, applicationmcp.ErrNotFound
		}
		return nil, fmt.Errorf("load mcp server for update: %w", err)
	}

	args, err := marshalJSON(input.Args)
	if err != nil {
		return nil, fmt.Errorf("marshal mcp args during update: %w", err)
	}
	headers, err := marshalJSON(input.Headers)
	if err != nil {
		return nil, fmt.Errorf("marshal mcp headers during update: %w", err)
	}

	model.TenantID = input.TenantID
	model.Name = input.Name
	model.Type = input.Type
	model.Transport = input.Transport
	model.Endpoint = input.Endpoint
	model.Command = input.Command
	model.ArgsJSON = datatypes.JSON(args)
	model.HeadersJSON = datatypes.JSON(headers)
	model.AuthType = input.AuthType
	model.Description = input.Description
	model.HealthStatus = input.HealthStatus
	model.IsEnabled = input.IsEnabled

	if input.Secret != "" {
		secret, err := r.secretBox.Encrypt(input.Secret)
		if err != nil {
			return nil, fmt.Errorf("encrypt mcp secret during update: %w", err)
		}
		model.SecretEncrypted = secret
	}

	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, fmt.Errorf("update mcp server: %w", err)
	}

	return r.Get(ctx, model.ID)
}

func (r *MCPRepository) Delete(ctx context.Context, id int64) error {
	if err := r.db.WithContext(ctx).Delete(&mysqlinfra.MCPServerModel{}, id).Error; err != nil {
		return fmt.Errorf("delete mcp server: %w", err)
	}
	return nil
}

func (r *MCPRepository) toRecord(model mysqlinfra.MCPServerModel) applicationmcp.Record {
	plain, _ := r.secretBox.Decrypt(model.SecretEncrypted)
	return applicationmcp.Record{
		ID:           model.ID,
		TenantID:     model.TenantID,
		Name:         model.Name,
		Type:         model.Type,
		Transport:    model.Transport,
		Endpoint:     model.Endpoint,
		Command:      model.Command,
		Args:         unmarshalStringSlice(model.ArgsJSON),
		Headers:      unmarshalStringMap(model.HeadersJSON),
		AuthType:     model.AuthType,
		Description:  model.Description,
		HealthStatus: model.HealthStatus,
		IsEnabled:    model.IsEnabled,
		HasSecret:    model.SecretEncrypted != "",
		MaskedSecret: cryptoinfra.MaskSecret(plain),
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}
