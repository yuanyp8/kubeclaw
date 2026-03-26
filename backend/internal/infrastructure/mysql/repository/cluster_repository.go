package repository

import (
	"context"
	"errors"
	"fmt"

	applicationcluster "kubeclaw/backend/internal/application/cluster"
	cryptoinfra "kubeclaw/backend/internal/infrastructure/crypto"
	mysqlinfra "kubeclaw/backend/internal/infrastructure/mysql"

	"gorm.io/gorm"
)

type ClusterRepository struct {
	db        *gorm.DB
	secretBox *cryptoinfra.SecretBox
}

func NewClusterRepository(db *gorm.DB, secretBox *cryptoinfra.SecretBox) *ClusterRepository {
	return &ClusterRepository{db: db, secretBox: secretBox}
}

func (r *ClusterRepository) List(ctx context.Context) ([]applicationcluster.Record, error) {
	var models []mysqlinfra.ClusterModel
	if err := r.db.WithContext(ctx).Order("id asc").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list clusters: %w", err)
	}

	result := make([]applicationcluster.Record, 0, len(models))
	for _, item := range models {
		result = append(result, toClusterRecord(item))
	}
	return result, nil
}

func (r *ClusterRepository) Get(ctx context.Context, id int64) (*applicationcluster.Record, error) {
	model, err := r.loadModel(ctx, id)
	if err != nil {
		return nil, err
	}

	record := toClusterRecord(*model)
	return &record, nil
}

func (r *ClusterRepository) GetConnection(ctx context.Context, id int64) (*applicationcluster.Connection, error) {
	model, err := r.loadModel(ctx, id)
	if err != nil {
		return nil, err
	}

	kubeConfig, err := r.secretBox.Decrypt(model.KubeConfigEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt kubeconfig: %w", err)
	}
	token, err := r.secretBox.Decrypt(model.TokenEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt token: %w", err)
	}
	caCert, err := r.secretBox.Decrypt(model.CACertEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt ca cert: %w", err)
	}
	credentials, err := r.secretBox.Decrypt(model.CredentialsEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt credentials: %w", err)
	}

	return &applicationcluster.Connection{
		ID:          model.ID,
		Name:        model.Name,
		APIserver:   model.APIserver,
		AuthType:    model.AuthType,
		KubeConfig:  kubeConfig,
		Token:       token,
		CACert:      caCert,
		Credentials: credentials,
	}, nil
}

func (r *ClusterRepository) Create(ctx context.Context, input applicationcluster.CreateInput) (*applicationcluster.Record, error) {
	kubeConfig, err := r.secretBox.Encrypt(input.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("encrypt kubeconfig: %w", err)
	}
	token, err := r.secretBox.Encrypt(input.Token)
	if err != nil {
		return nil, fmt.Errorf("encrypt token: %w", err)
	}
	caCert, err := r.secretBox.Encrypt(input.CACert)
	if err != nil {
		return nil, fmt.Errorf("encrypt ca cert: %w", err)
	}
	credentials, err := r.secretBox.Encrypt(input.Credentials)
	if err != nil {
		return nil, fmt.Errorf("encrypt credentials: %w", err)
	}

	model := mysqlinfra.ClusterModel{
		TenantID:             input.TenantID,
		Name:                 input.Name,
		Description:          input.Description,
		APIserver:            input.APIserver,
		Environment:          input.Environment,
		AuthType:             input.AuthType,
		KubeConfigEncrypted:  kubeConfig,
		TokenEncrypted:       token,
		CACertEncrypted:      caCert,
		CredentialsEncrypted: credentials,
		IsPublic:             input.IsPublic,
		Status:               input.Status,
		OwnerUserID:          input.OwnerUserID,
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, fmt.Errorf("create cluster: %w", err)
	}

	return r.Get(ctx, model.ID)
}

func (r *ClusterRepository) Update(ctx context.Context, id int64, input applicationcluster.UpdateInput) (*applicationcluster.Record, error) {
	model, err := r.loadModel(ctx, id)
	if err != nil {
		return nil, err
	}

	model.TenantID = input.TenantID
	model.Name = input.Name
	model.Description = input.Description
	model.APIserver = input.APIserver
	model.Environment = input.Environment
	model.AuthType = input.AuthType
	model.IsPublic = input.IsPublic
	model.Status = input.Status
	model.OwnerUserID = input.OwnerUserID

	if input.KubeConfig != "" {
		cipherText, err := r.secretBox.Encrypt(input.KubeConfig)
		if err != nil {
			return nil, fmt.Errorf("encrypt kubeconfig during update: %w", err)
		}
		model.KubeConfigEncrypted = cipherText
	}
	if input.Token != "" {
		cipherText, err := r.secretBox.Encrypt(input.Token)
		if err != nil {
			return nil, fmt.Errorf("encrypt token during update: %w", err)
		}
		model.TokenEncrypted = cipherText
	}
	if input.CACert != "" {
		cipherText, err := r.secretBox.Encrypt(input.CACert)
		if err != nil {
			return nil, fmt.Errorf("encrypt ca cert during update: %w", err)
		}
		model.CACertEncrypted = cipherText
	}
	if input.Credentials != "" {
		cipherText, err := r.secretBox.Encrypt(input.Credentials)
		if err != nil {
			return nil, fmt.Errorf("encrypt credentials during update: %w", err)
		}
		model.CredentialsEncrypted = cipherText
	}

	if err := r.db.WithContext(ctx).Save(model).Error; err != nil {
		return nil, fmt.Errorf("update cluster: %w", err)
	}

	return r.Get(ctx, model.ID)
}

func (r *ClusterRepository) Delete(ctx context.Context, id int64) error {
	if err := r.db.WithContext(ctx).Delete(&mysqlinfra.ClusterModel{}, id).Error; err != nil {
		return fmt.Errorf("delete cluster: %w", err)
	}
	return nil
}

func (r *ClusterRepository) Share(ctx context.Context, clusterID int64, input applicationcluster.ShareInput) (*applicationcluster.PermissionRecord, error) {
	model := mysqlinfra.ClusterPermissionModel{
		ClusterID: clusterID,
		UserID:    input.UserID,
		TeamID:    input.TeamID,
		Role:      input.Role,
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, fmt.Errorf("create cluster permission: %w", err)
	}

	record := applicationcluster.PermissionRecord{
		ID:        model.ID,
		ClusterID: model.ClusterID,
		UserID:    model.UserID,
		TeamID:    model.TeamID,
		Role:      model.Role,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}
	return &record, nil
}

func (r *ClusterRepository) ListPermissions(ctx context.Context, clusterID int64) ([]applicationcluster.PermissionRecord, error) {
	var models []mysqlinfra.ClusterPermissionModel
	if err := r.db.WithContext(ctx).Where("cluster_id = ?", clusterID).Order("id asc").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list cluster permissions: %w", err)
	}

	result := make([]applicationcluster.PermissionRecord, 0, len(models))
	for _, item := range models {
		result = append(result, applicationcluster.PermissionRecord{
			ID:        item.ID,
			ClusterID: item.ClusterID,
			UserID:    item.UserID,
			TeamID:    item.TeamID,
			Role:      item.Role,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return result, nil
}

func (r *ClusterRepository) loadModel(ctx context.Context, id int64) (*mysqlinfra.ClusterModel, error) {
	var model mysqlinfra.ClusterModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, applicationcluster.ErrNotFound
		}
		return nil, fmt.Errorf("get cluster: %w", err)
	}
	return &model, nil
}

func toClusterRecord(model mysqlinfra.ClusterModel) applicationcluster.Record {
	return applicationcluster.Record{
		ID:             model.ID,
		TenantID:       model.TenantID,
		Name:           model.Name,
		Description:    model.Description,
		APIserver:      model.APIserver,
		Environment:    model.Environment,
		AuthType:       model.AuthType,
		IsPublic:       model.IsPublic,
		Status:         model.Status,
		OwnerUserID:    model.OwnerUserID,
		HasKubeConfig:  model.KubeConfigEncrypted != "",
		HasToken:       model.TokenEncrypted != "",
		HasCACert:      model.CACertEncrypted != "",
		HasCredentials: model.CredentialsEncrypted != "",
		CreatedAt:      model.CreatedAt,
		UpdatedAt:      model.UpdatedAt,
	}
}
