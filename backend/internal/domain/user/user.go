package user

import (
	"context"
	"errors"
	"time"
)

// Role 表示平台内置的用户角色。
type Role string

const (
	RoleAdmin        Role = "admin"
	RoleClusterAdmin Role = "cluster_admin"
	RoleUser         Role = "user"
	RoleReadonly     Role = "readonly"
)

var ErrNotFound = errors.New("user not found")

// User 是用户领域实体。
type User struct {
	ID           int64
	TenantID     *int64
	Username     string
	Email        string
	DisplayName  string
	Phone        string
	AvatarURL    string
	PasswordHash string
	Role         Role
	Status       string
	LastLoginAt  *time.Time
	Enabled      bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsActive 用于判断用户当前是否可登录。
func (u User) IsActive() bool {
	return u.Enabled
}

// Repository 定义用户聚合的基础读取能力。
type Repository interface {
	FindByID(ctx context.Context, id int64) (*User, error)
	FindByLogin(ctx context.Context, login string) (*User, error)
}
