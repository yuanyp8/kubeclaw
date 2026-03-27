package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	applicationuser "kubeclaw/backend/internal/application/user"
	domainuser "kubeclaw/backend/internal/domain/user"
	mysqlinfra "kubeclaw/backend/internal/infrastructure/mysql"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) FindByID(ctx context.Context, id int64) (*domainuser.User, error) {
	var model mysqlinfra.UserModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainuser.ErrNotFound
		}
		return nil, fmt.Errorf("query user by id: %w", err)
	}

	return toDomainUser(model), nil
}

func (r *UserRepository) FindByLogin(ctx context.Context, login string) (*domainuser.User, error) {
	var model mysqlinfra.UserModel
	if err := r.db.WithContext(ctx).
		Where("username = ? OR email = ?", login, login).
		First(&model).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainuser.ErrNotFound
		}
		return nil, fmt.Errorf("query user by login: %w", err)
	}

	return toDomainUser(model), nil
}

func (r *UserRepository) List(ctx context.Context) ([]applicationuser.Profile, error) {
	var models []mysqlinfra.UserModel
	if err := r.db.WithContext(ctx).Order("id asc").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}

	return r.buildProfiles(ctx, models)
}

func (r *UserRepository) ListByTenant(ctx context.Context, tenantID int64) ([]applicationuser.Profile, error) {
	var models []mysqlinfra.UserModel
	if err := r.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("id asc").
		Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list users by tenant: %w", err)
	}

	return r.buildProfiles(ctx, models)
}

func (r *UserRepository) Get(ctx context.Context, id int64) (*applicationuser.Profile, error) {
	var model mysqlinfra.UserModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, applicationuser.ErrNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	profiles, err := r.buildProfiles(ctx, []mysqlinfra.UserModel{model})
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, applicationuser.ErrNotFound
	}
	return &profiles[0], nil
}

func (r *UserRepository) Create(ctx context.Context, input applicationuser.CreateInput) (*applicationuser.Profile, error) {
	model := mysqlinfra.UserModel{
		TenantID:     input.TenantID,
		Username:     input.Username,
		Email:        input.Email,
		DisplayName:  input.DisplayName,
		Phone:        input.Phone,
		AvatarURL:    input.AvatarURL,
		PasswordHash: input.PasswordHash,
		Role:         input.Role,
		Status:       input.Status,
	}

	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	return r.Get(ctx, model.ID)
}

func (r *UserRepository) Update(ctx context.Context, id int64, input applicationuser.UpdateInput) (*applicationuser.Profile, error) {
	var model mysqlinfra.UserModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, applicationuser.ErrNotFound
		}
		return nil, fmt.Errorf("load user for update: %w", err)
	}

	model.TenantID = input.TenantID
	model.Email = input.Email
	model.DisplayName = input.DisplayName
	model.Phone = input.Phone
	model.AvatarURL = input.AvatarURL
	model.Role = input.Role
	model.Status = input.Status
	if input.PasswordHash != "" {
		model.PasswordHash = input.PasswordHash
	}

	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return r.Get(ctx, model.ID)
}

func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", id).Delete(&mysqlinfra.UserTeamModel{}).Error; err != nil {
			return fmt.Errorf("delete user team memberships: %w", err)
		}
		if err := tx.Model(&mysqlinfra.TeamModel{}).Where("owner_user_id = ?", id).Update("owner_user_id", nil).Error; err != nil {
			return fmt.Errorf("clear owned teams: %w", err)
		}
		if err := tx.Model(&mysqlinfra.TenantModel{}).Where("owner_user_id = ?", id).Update("owner_user_id", nil).Error; err != nil {
			return fmt.Errorf("clear owned tenants: %w", err)
		}
		if err := tx.Delete(&mysqlinfra.UserModel{}, id).Error; err != nil {
			return fmt.Errorf("delete user: %w", err)
		}
		return nil
	})
}

func (r *UserRepository) UpdateLastLoginAt(ctx context.Context, userID int64, loginAt time.Time) error {
	if err := r.db.WithContext(ctx).
		Model(&mysqlinfra.UserModel{}).
		Where("id = ?", userID).
		Update("last_login_at", loginAt).Error; err != nil {
		return fmt.Errorf("update last login at: %w", err)
	}

	return nil
}

func (r *UserRepository) buildProfiles(ctx context.Context, users []mysqlinfra.UserModel) ([]applicationuser.Profile, error) {
	if len(users) == 0 {
		return []applicationuser.Profile{}, nil
	}

	userIDs := make([]int64, 0, len(users))
	tenantIDs := make([]int64, 0, len(users))
	tenantSeen := make(map[int64]struct{})
	for _, item := range users {
		userIDs = append(userIDs, item.ID)
		if item.TenantID != nil {
			if _, ok := tenantSeen[*item.TenantID]; !ok {
				tenantIDs = append(tenantIDs, *item.TenantID)
				tenantSeen[*item.TenantID] = struct{}{}
			}
		}
	}

	tenantsByID, err := r.loadTenantSummaries(ctx, tenantIDs)
	if err != nil {
		return nil, err
	}

	membershipsByUser, err := r.loadTeamMemberships(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	result := make([]applicationuser.Profile, 0, len(users))
	for _, item := range users {
		var tenant *applicationuser.TenantSummary
		if item.TenantID != nil {
			tenant = tenantsByID[*item.TenantID]
		}

		result = append(result, applicationuser.Profile{
			ID:          item.ID,
			TenantID:    item.TenantID,
			Tenant:      tenant,
			Teams:       membershipsByUser[item.ID],
			Username:    item.Username,
			Email:       item.Email,
			DisplayName: item.DisplayName,
			Phone:       item.Phone,
			AvatarURL:   item.AvatarURL,
			Role:        item.Role,
			Status:      item.Status,
			LastLoginAt: item.LastLoginAt,
			Enabled:     item.Status == "active",
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}

	return result, nil
}

func (r *UserRepository) loadTenantSummaries(ctx context.Context, tenantIDs []int64) (map[int64]*applicationuser.TenantSummary, error) {
	result := make(map[int64]*applicationuser.TenantSummary)
	if len(tenantIDs) == 0 {
		return result, nil
	}

	var models []mysqlinfra.TenantModel
	if err := r.db.WithContext(ctx).Where("id IN ?", tenantIDs).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("load tenant summaries: %w", err)
	}

	for _, item := range models {
		summary := applicationuser.TenantSummary{
			ID:   item.ID,
			Name: item.Name,
			Slug: item.Slug,
		}
		result[item.ID] = &summary
	}

	return result, nil
}

func (r *UserRepository) loadTeamMemberships(ctx context.Context, userIDs []int64) (map[int64][]applicationuser.TeamMembership, error) {
	result := make(map[int64][]applicationuser.TeamMembership)
	if len(userIDs) == 0 {
		return result, nil
	}

	type joinedMembership struct {
		UserID   int64
		TeamID   int64
		TeamName string
		Role     string
	}

	var rows []joinedMembership
	if err := r.db.WithContext(ctx).
		Table("user_teams AS ut").
		Select("ut.user_id, ut.team_id, ut.role, t.name AS team_name").
		Joins("JOIN teams AS t ON t.id = ut.team_id AND t.deleted_at IS NULL").
		Where("ut.user_id IN ? AND ut.deleted_at IS NULL", userIDs).
		Order("ut.id asc").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load user team memberships: %w", err)
	}

	for _, row := range rows {
		result[row.UserID] = append(result[row.UserID], applicationuser.TeamMembership{
			TeamID:   row.TeamID,
			TeamName: row.TeamName,
			Role:     row.Role,
		})
	}

	return result, nil
}

func toDomainUser(model mysqlinfra.UserModel) *domainuser.User {
	role := domainuser.Role(model.Role)
	return &domainuser.User{
		ID:           model.ID,
		TenantID:     model.TenantID,
		Username:     model.Username,
		Email:        model.Email,
		DisplayName:  model.DisplayName,
		Phone:        model.Phone,
		AvatarURL:    model.AvatarURL,
		PasswordHash: model.PasswordHash,
		Role:         role,
		Status:       model.Status,
		LastLoginAt:  model.LastLoginAt,
		Enabled:      model.Status == "active",
		CreatedAt:    model.CreatedAt,
		UpdatedAt:    model.UpdatedAt,
	}
}
