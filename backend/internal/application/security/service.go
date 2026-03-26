package security

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("security rule not found")

type IPWhitelistRecord struct {
	ID          int64     `json:"id"`
	TenantID    *int64    `json:"tenantId"`
	Name        string    `json:"name"`
	IPOrCIDR    string    `json:"ipOrCidr"`
	Scope       string    `json:"scope"`
	Description string    `json:"description"`
	IsEnabled   bool      `json:"isEnabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type SensitiveWordRecord struct {
	ID          int64     `json:"id"`
	TenantID    *int64    `json:"tenantId"`
	Word        string    `json:"word"`
	Category    string    `json:"category"`
	Level       string    `json:"level"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	IsEnabled   bool      `json:"isEnabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type SensitiveFieldRuleRecord struct {
	ID          int64     `json:"id"`
	TenantID    *int64    `json:"tenantId"`
	Name        string    `json:"name"`
	Resource    string    `json:"resource"`
	FieldPath   string    `json:"fieldPath"`
	Action      string    `json:"action"`
	Description string    `json:"description"`
	IsEnabled   bool      `json:"isEnabled"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type IPWhitelistInput struct {
	TenantID    *int64 `json:"tenantId"`
	Name        string `json:"name"`
	IPOrCIDR    string `json:"ipOrCidr"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
	IsEnabled   bool   `json:"isEnabled"`
}

type SensitiveWordInput struct {
	TenantID    *int64 `json:"tenantId"`
	Word        string `json:"word"`
	Category    string `json:"category"`
	Level       string `json:"level"`
	Action      string `json:"action"`
	Description string `json:"description"`
	IsEnabled   bool   `json:"isEnabled"`
}

type SensitiveFieldRuleInput struct {
	TenantID    *int64 `json:"tenantId"`
	Name        string `json:"name"`
	Resource    string `json:"resource"`
	FieldPath   string `json:"fieldPath"`
	Action      string `json:"action"`
	Description string `json:"description"`
	IsEnabled   bool   `json:"isEnabled"`
}

type Repository interface {
	ListIPWhitelists(ctx context.Context) ([]IPWhitelistRecord, error)
	GetIPWhitelist(ctx context.Context, id int64) (*IPWhitelistRecord, error)
	CreateIPWhitelist(ctx context.Context, input IPWhitelistInput) (*IPWhitelistRecord, error)
	UpdateIPWhitelist(ctx context.Context, id int64, input IPWhitelistInput) (*IPWhitelistRecord, error)
	DeleteIPWhitelist(ctx context.Context, id int64) error

	ListSensitiveWords(ctx context.Context) ([]SensitiveWordRecord, error)
	GetSensitiveWord(ctx context.Context, id int64) (*SensitiveWordRecord, error)
	CreateSensitiveWord(ctx context.Context, input SensitiveWordInput) (*SensitiveWordRecord, error)
	UpdateSensitiveWord(ctx context.Context, id int64, input SensitiveWordInput) (*SensitiveWordRecord, error)
	DeleteSensitiveWord(ctx context.Context, id int64) error

	ListSensitiveFieldRules(ctx context.Context) ([]SensitiveFieldRuleRecord, error)
	GetSensitiveFieldRule(ctx context.Context, id int64) (*SensitiveFieldRuleRecord, error)
	CreateSensitiveFieldRule(ctx context.Context, input SensitiveFieldRuleInput) (*SensitiveFieldRuleRecord, error)
	UpdateSensitiveFieldRule(ctx context.Context, id int64, input SensitiveFieldRuleInput) (*SensitiveFieldRuleRecord, error)
	DeleteSensitiveFieldRule(ctx context.Context, id int64) error
}

type Service struct{ repo Repository }

func NewService(repo Repository) *Service { return &Service{repo: repo} }

func (s *Service) ListIPWhitelists(ctx context.Context) ([]IPWhitelistRecord, error) {
	return s.repo.ListIPWhitelists(ctx)
}
func (s *Service) GetIPWhitelist(ctx context.Context, id int64) (*IPWhitelistRecord, error) {
	return s.repo.GetIPWhitelist(ctx, id)
}
func (s *Service) CreateIPWhitelist(ctx context.Context, input IPWhitelistInput) (*IPWhitelistRecord, error) {
	return s.repo.CreateIPWhitelist(ctx, input)
}
func (s *Service) UpdateIPWhitelist(ctx context.Context, id int64, input IPWhitelistInput) (*IPWhitelistRecord, error) {
	return s.repo.UpdateIPWhitelist(ctx, id, input)
}
func (s *Service) DeleteIPWhitelist(ctx context.Context, id int64) error {
	return s.repo.DeleteIPWhitelist(ctx, id)
}

func (s *Service) ListSensitiveWords(ctx context.Context) ([]SensitiveWordRecord, error) {
	return s.repo.ListSensitiveWords(ctx)
}
func (s *Service) GetSensitiveWord(ctx context.Context, id int64) (*SensitiveWordRecord, error) {
	return s.repo.GetSensitiveWord(ctx, id)
}
func (s *Service) CreateSensitiveWord(ctx context.Context, input SensitiveWordInput) (*SensitiveWordRecord, error) {
	return s.repo.CreateSensitiveWord(ctx, input)
}
func (s *Service) UpdateSensitiveWord(ctx context.Context, id int64, input SensitiveWordInput) (*SensitiveWordRecord, error) {
	return s.repo.UpdateSensitiveWord(ctx, id, input)
}
func (s *Service) DeleteSensitiveWord(ctx context.Context, id int64) error {
	return s.repo.DeleteSensitiveWord(ctx, id)
}

func (s *Service) ListSensitiveFieldRules(ctx context.Context) ([]SensitiveFieldRuleRecord, error) {
	return s.repo.ListSensitiveFieldRules(ctx)
}
func (s *Service) GetSensitiveFieldRule(ctx context.Context, id int64) (*SensitiveFieldRuleRecord, error) {
	return s.repo.GetSensitiveFieldRule(ctx, id)
}
func (s *Service) CreateSensitiveFieldRule(ctx context.Context, input SensitiveFieldRuleInput) (*SensitiveFieldRuleRecord, error) {
	return s.repo.CreateSensitiveFieldRule(ctx, input)
}
func (s *Service) UpdateSensitiveFieldRule(ctx context.Context, id int64, input SensitiveFieldRuleInput) (*SensitiveFieldRuleRecord, error) {
	return s.repo.UpdateSensitiveFieldRule(ctx, id, input)
}
func (s *Service) DeleteSensitiveFieldRule(ctx context.Context, id int64) error {
	return s.repo.DeleteSensitiveFieldRule(ctx, id)
}
