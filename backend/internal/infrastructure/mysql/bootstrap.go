package mysql

import (
	"context"
	"errors"
	"fmt"
	"time"

	"kubeclaw/backend/internal/config"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Bootstrap 会创建系统默认租户和管理员账号，避免新环境无法登录。
func Bootstrap(ctx context.Context, db *gorm.DB, cfg config.Config) error {
	var tenant TenantModel
	if err := db.WithContext(ctx).
		Where("slug = ?", cfg.SystemTenantSlug).
		First(&tenant).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("query system tenant: %w", err)
		}

		tenant = TenantModel{
			Name:     cfg.SystemTenantName,
			Slug:     cfg.SystemTenantSlug,
			Status:   "active",
			IsSystem: true,
		}

		if err := db.WithContext(ctx).Create(&tenant).Error; err != nil {
			return fmt.Errorf("create system tenant: %w", err)
		}
	}

	var user UserModel
	if err := db.WithContext(ctx).
		Where("username = ?", cfg.BootstrapAdminUsername).
		First(&user).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("query bootstrap admin: %w", err)
		}

		passwordHash, err := bcrypt.GenerateFromPassword([]byte(cfg.BootstrapAdminPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash bootstrap admin password: %w", err)
		}

		now := time.Now()
		user = UserModel{
			TenantID:     &tenant.ID,
			Username:     cfg.BootstrapAdminUsername,
			Email:        cfg.BootstrapAdminEmail,
			DisplayName:  "System Admin",
			PasswordHash: string(passwordHash),
			Role:         "admin",
			Status:       "active",
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if err := db.WithContext(ctx).Create(&user).Error; err != nil {
			return fmt.Errorf("create bootstrap admin: %w", err)
		}
	}

	return nil
}
