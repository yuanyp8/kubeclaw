package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	applicationteam "kubeclaw/backend/internal/application/team"
	mysqlinfra "kubeclaw/backend/internal/infrastructure/mysql"

	"gorm.io/gorm"
)

type TeamRepository struct {
	db *gorm.DB
}

func NewTeamRepository(db *gorm.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) List(ctx context.Context) ([]applicationteam.Record, error) {
	var models []mysqlinfra.TeamModel
	if err := r.db.WithContext(ctx).Order("id asc").Find(&models).Error; err != nil {
		return nil, fmt.Errorf("list teams: %w", err)
	}

	return r.buildTeamRecords(ctx, models)
}

func (r *TeamRepository) Get(ctx context.Context, id int64) (*applicationteam.Record, error) {
	var model mysqlinfra.TeamModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, applicationteam.ErrNotFound
		}
		return nil, fmt.Errorf("get team: %w", err)
	}

	items, err := r.buildTeamRecords(ctx, []mysqlinfra.TeamModel{model})
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, applicationteam.ErrNotFound
	}
	return &items[0], nil
}

func (r *TeamRepository) Create(ctx context.Context, input applicationteam.Input) (*applicationteam.Record, error) {
	var teamID int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		model := mysqlinfra.TeamModel{
			TenantID:    input.TenantID,
			Name:        input.Name,
			Description: input.Description,
			OwnerUserID: input.OwnerUserID,
			Visibility:  input.Visibility,
		}

		if err := tx.Create(&model).Error; err != nil {
			return fmt.Errorf("create team: %w", err)
		}

		teamID = model.ID

		if input.OwnerUserID != nil {
			member := mysqlinfra.UserTeamModel{
				UserID: *input.OwnerUserID,
				TeamID: model.ID,
				Role:   "owner",
			}
			if err := tx.Create(&member).Error; err != nil {
				return fmt.Errorf("create team owner member: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return r.Get(ctx, teamID)
}

func (r *TeamRepository) Update(ctx context.Context, id int64, input applicationteam.Input) (*applicationteam.Record, error) {
	var model mysqlinfra.TeamModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, applicationteam.ErrNotFound
		}
		return nil, fmt.Errorf("load team for update: %w", err)
	}

	model.TenantID = input.TenantID
	model.Name = input.Name
	model.Description = input.Description
	model.OwnerUserID = input.OwnerUserID
	model.Visibility = input.Visibility

	if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
		return nil, fmt.Errorf("update team: %w", err)
	}

	return r.Get(ctx, model.ID)
}

func (r *TeamRepository) Delete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("team_id = ?", id).Delete(&mysqlinfra.UserTeamModel{}).Error; err != nil {
			return fmt.Errorf("delete team members: %w", err)
		}
		if err := tx.Delete(&mysqlinfra.TeamModel{}, id).Error; err != nil {
			return fmt.Errorf("delete team: %w", err)
		}
		return nil
	})
}

func (r *TeamRepository) ListMembers(ctx context.Context, teamID int64) ([]applicationteam.MemberRecord, error) {
	var rows []struct {
		ID          int64
		TeamID      int64
		UserID      int64
		Username    string
		Email       string
		DisplayName string
		Role        string
		CreatedAt   time.Time
		UpdatedAt   time.Time
	}

	if err := r.db.WithContext(ctx).
		Table("user_teams AS ut").
		Select("ut.id, ut.team_id, ut.user_id, ut.role, ut.created_at, ut.updated_at, u.username, u.email, u.display_name").
		Joins("JOIN users AS u ON u.id = ut.user_id AND u.deleted_at IS NULL").
		Where("ut.team_id = ? AND ut.deleted_at IS NULL", teamID).
		Order("ut.id asc").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list team members: %w", err)
	}

	result := make([]applicationteam.MemberRecord, 0, len(rows))
	for _, row := range rows {
		result = append(result, applicationteam.MemberRecord{
			ID:          row.ID,
			TeamID:      row.TeamID,
			UserID:      row.UserID,
			Username:    row.Username,
			Email:       row.Email,
			DisplayName: row.DisplayName,
			Role:        row.Role,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return result, nil
}

func (r *TeamRepository) AddMember(ctx context.Context, teamID int64, input applicationteam.AddMemberInput) (*applicationteam.MemberRecord, error) {
	var model mysqlinfra.UserTeamModel
	err := r.db.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, input.UserID).
		First(&model).Error

	switch {
	case err == nil:
		model.Role = input.Role
		if err := r.db.WithContext(ctx).Save(&model).Error; err != nil {
			return nil, fmt.Errorf("update team member role: %w", err)
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		model = mysqlinfra.UserTeamModel{
			TeamID: teamID,
			UserID: input.UserID,
			Role:   input.Role,
		}
		if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
			return nil, fmt.Errorf("create team member: %w", err)
		}
	default:
		return nil, fmt.Errorf("load team member: %w", err)
	}

	items, err := r.ListMembers(ctx, teamID)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.UserID == input.UserID {
			member := item
			return &member, nil
		}
	}

	return nil, fmt.Errorf("team member not found after upsert")
}

func (r *TeamRepository) RemoveMember(ctx context.Context, teamID int64, userID int64) error {
	if err := r.db.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		Delete(&mysqlinfra.UserTeamModel{}).Error; err != nil {
		return fmt.Errorf("remove team member: %w", err)
	}
	return nil
}

func (r *TeamRepository) buildTeamRecords(ctx context.Context, teams []mysqlinfra.TeamModel) ([]applicationteam.Record, error) {
	if len(teams) == 0 {
		return []applicationteam.Record{}, nil
	}

	teamIDs := make([]int64, 0, len(teams))
	tenantIDs := make([]int64, 0, len(teams))
	tenantSeen := make(map[int64]struct{})
	for _, item := range teams {
		teamIDs = append(teamIDs, item.ID)
		if item.TenantID != nil {
			if _, ok := tenantSeen[*item.TenantID]; !ok {
				tenantIDs = append(tenantIDs, *item.TenantID)
				tenantSeen[*item.TenantID] = struct{}{}
			}
		}
	}

	memberCounts, err := r.loadMemberCounts(ctx, teamIDs)
	if err != nil {
		return nil, err
	}
	tenantsByID, err := r.loadTenantSummaries(ctx, tenantIDs)
	if err != nil {
		return nil, err
	}

	result := make([]applicationteam.Record, 0, len(teams))
	for _, item := range teams {
		var tenant *applicationteam.TenantSummary
		if item.TenantID != nil {
			tenant = tenantsByID[*item.TenantID]
		}

		result = append(result, applicationteam.Record{
			ID:          item.ID,
			TenantID:    item.TenantID,
			Tenant:      tenant,
			Name:        item.Name,
			Description: item.Description,
			OwnerUserID: item.OwnerUserID,
			Visibility:  item.Visibility,
			MemberCount: memberCounts[item.ID],
			CreatedAt:   item.CreatedAt,
			UpdatedAt:   item.UpdatedAt,
		})
	}

	return result, nil
}

func (r *TeamRepository) loadMemberCounts(ctx context.Context, teamIDs []int64) (map[int64]int, error) {
	result := make(map[int64]int)
	if len(teamIDs) == 0 {
		return result, nil
	}

	var rows []struct {
		TeamID int64
		Count  int64
	}
	if err := r.db.WithContext(ctx).
		Table("user_teams").
		Select("team_id, count(*) AS count").
		Where("team_id IN ? AND deleted_at IS NULL", teamIDs).
		Group("team_id").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("load team member counts: %w", err)
	}

	for _, row := range rows {
		result[row.TeamID] = int(row.Count)
	}

	return result, nil
}

func (r *TeamRepository) loadTenantSummaries(ctx context.Context, tenantIDs []int64) (map[int64]*applicationteam.TenantSummary, error) {
	result := make(map[int64]*applicationteam.TenantSummary)
	if len(tenantIDs) == 0 {
		return result, nil
	}

	var models []mysqlinfra.TenantModel
	if err := r.db.WithContext(ctx).Where("id IN ?", tenantIDs).Find(&models).Error; err != nil {
		return nil, fmt.Errorf("load team tenants: %w", err)
	}

	for _, item := range models {
		summary := applicationteam.TenantSummary{
			ID:   item.ID,
			Name: item.Name,
			Slug: item.Slug,
		}
		result[item.ID] = &summary
	}

	return result, nil
}
