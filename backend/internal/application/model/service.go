package model

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("model config not found")

type Repository interface {
	List(ctx context.Context) ([]Record, error)
	Get(ctx context.Context, id int64) (*Record, error)
	GetDefault(ctx context.Context) (*Record, error)
	Resolve(ctx context.Context, id int64) (*ResolvedRecord, error)
	ResolveDefault(ctx context.Context) (*ResolvedRecord, error)
	Create(ctx context.Context, input CreateInput) (*Record, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*Record, error)
	Delete(ctx context.Context, id int64) error
	SetDefault(ctx context.Context, id int64) (*Record, error)
}

type Tester interface {
	TestConnection(ctx context.Context, model ResolvedRecord) (*TestResult, error)
}

type Record struct {
	ID           int64     `json:"id"`
	TenantID     *int64    `json:"tenantId"`
	Name         string    `json:"name"`
	Provider     string    `json:"provider"`
	Model        string    `json:"model"`
	BaseURL      string    `json:"baseUrl"`
	Description  string    `json:"description"`
	Capabilities []string  `json:"capabilities"`
	IsDefault    bool      `json:"isDefault"`
	IsEnabled    bool      `json:"isEnabled"`
	MaxTokens    int       `json:"maxTokens"`
	Temperature  float64   `json:"temperature"`
	TopP         float64   `json:"topP"`
	HasAPIKey    bool      `json:"hasApiKey"`
	MaskedAPIKey string    `json:"maskedApiKey"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type ResolvedRecord struct {
	Record
	APIKey string `json:"-"`
}

type CreateInput struct {
	TenantID     *int64   `json:"tenantId"`
	Name         string   `json:"name"`
	Provider     string   `json:"provider"`
	Model        string   `json:"model"`
	BaseURL      string   `json:"baseUrl"`
	APIKey       string   `json:"apiKey"`
	Description  string   `json:"description"`
	Capabilities []string `json:"capabilities"`
	IsDefault    bool     `json:"isDefault"`
	IsEnabled    bool     `json:"isEnabled"`
	MaxTokens    int      `json:"maxTokens"`
	Temperature  float64  `json:"temperature"`
	TopP         float64  `json:"topP"`
}

type UpdateInput = CreateInput

type TestResult struct {
	Reachable bool      `json:"reachable"`
	Model     string    `json:"model"`
	Provider  string    `json:"provider"`
	Message   string    `json:"message"`
	CheckedAt time.Time `json:"checkedAt"`
}

type Service struct {
	repo   Repository
	tester Tester
}

func NewService(repo Repository, tester Tester) *Service {
	return &Service{repo: repo, tester: tester}
}

func (s *Service) List(ctx context.Context) ([]Record, error) {
	return s.repo.List(ctx)
}

func (s *Service) Get(ctx context.Context, id int64) (*Record, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) GetDefault(ctx context.Context) (*Record, error) {
	return s.repo.GetDefault(ctx)
}

func (s *Service) Resolve(ctx context.Context, id int64) (*ResolvedRecord, error) {
	return s.repo.Resolve(ctx, id)
}

func (s *Service) ResolveDefault(ctx context.Context) (*ResolvedRecord, error) {
	return s.repo.ResolveDefault(ctx)
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*Record, error) {
	return s.repo.Create(ctx, input)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*Record, error) {
	return s.repo.Update(ctx, id, input)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) SetDefault(ctx context.Context, id int64) (*Record, error) {
	return s.repo.SetDefault(ctx, id)
}

func (s *Service) TestConnection(ctx context.Context, id int64) (*TestResult, error) {
	if s.tester == nil {
		return nil, errors.New("model tester is not configured")
	}

	resolved, err := s.repo.Resolve(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.tester.TestConnection(ctx, *resolved)
}

func (s *Service) TestDraft(ctx context.Context, input CreateInput) (*TestResult, error) {
	if s.tester == nil {
		return nil, errors.New("model tester is not configured")
	}

	return s.tester.TestConnection(ctx, ResolvedRecord{
		Record: Record{
			TenantID:     input.TenantID,
			Name:         input.Name,
			Provider:     input.Provider,
			Model:        input.Model,
			BaseURL:      input.BaseURL,
			Description:  input.Description,
			Capabilities: input.Capabilities,
			IsDefault:    input.IsDefault,
			IsEnabled:    input.IsEnabled,
			MaxTokens:    input.MaxTokens,
			Temperature:  input.Temperature,
			TopP:         input.TopP,
			HasAPIKey:    input.APIKey != "",
		},
		APIKey: input.APIKey,
	})
}

func (s *Service) TestUpdatedDraft(ctx context.Context, id int64, input UpdateInput) (*TestResult, error) {
	if s.tester == nil {
		return nil, errors.New("model tester is not configured")
	}

	resolved, err := s.repo.Resolve(ctx, id)
	if err != nil {
		return nil, err
	}

	resolved.TenantID = input.TenantID
	resolved.Name = input.Name
	resolved.Provider = input.Provider
	resolved.Model = input.Model
	resolved.BaseURL = input.BaseURL
	resolved.Description = input.Description
	resolved.Capabilities = input.Capabilities
	resolved.IsDefault = input.IsDefault
	resolved.IsEnabled = input.IsEnabled
	resolved.MaxTokens = input.MaxTokens
	resolved.Temperature = input.Temperature
	resolved.TopP = input.TopP
	if input.APIKey != "" {
		resolved.APIKey = input.APIKey
		resolved.HasAPIKey = true
	}

	return s.tester.TestConnection(ctx, *resolved)
}
