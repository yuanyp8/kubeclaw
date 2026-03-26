package memory

import (
	"context"
	"strings"
	"sync"

	domainuser "kubeclaw/backend/internal/domain/user"
)

// UserRepository 使用内存存储用户，适合 MVP 阶段与本地联调。
type UserRepository struct {
	mu        sync.RWMutex
	usersByID map[int64]domainuser.User
}

// NewUserRepository 初始化内存用户仓库。
func NewUserRepository(seedUsers []domainuser.User) *UserRepository {
	repo := &UserRepository{
		usersByID: make(map[int64]domainuser.User, len(seedUsers)),
	}

	for _, item := range seedUsers {
		repo.usersByID[item.ID] = item
	}

	return repo
}

// FindByID 根据 ID 查询用户。
func (r *UserRepository) FindByID(_ context.Context, id int64) (*domainuser.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	value, ok := r.usersByID[id]
	if !ok {
		return nil, domainuser.ErrNotFound
	}

	userCopy := value
	return &userCopy, nil
}

// FindByLogin 允许通过用户名或邮箱登录。
func (r *UserRepository) FindByLogin(_ context.Context, login string) (*domainuser.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	normalized := strings.ToLower(strings.TrimSpace(login))
	for _, item := range r.usersByID {
		if strings.ToLower(item.Username) == normalized || strings.ToLower(item.Email) == normalized {
			userCopy := item
			return &userCopy, nil
		}
	}

	return nil, domainuser.ErrNotFound
}
