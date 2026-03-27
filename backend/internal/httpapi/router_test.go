package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	applicationauth "kubeclaw/backend/internal/application/auth"
	appskill "kubeclaw/backend/internal/application/skill"
	applicationuser "kubeclaw/backend/internal/application/user"
	"kubeclaw/backend/internal/config"
	domainauth "kubeclaw/backend/internal/domain/auth"
	domainuser "kubeclaw/backend/internal/domain/user"
	"kubeclaw/backend/internal/httpapi/handlers"
	"kubeclaw/backend/internal/httpapi/middleware"
)

func TestRouterPermissionMatrix(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)

	tests := []struct {
		name       string
		method     string
		path       string
		token      string
		wantStatus int
	}{
		{name: "protected route requires auth", method: http.MethodGet, path: "/api/users", wantStatus: http.StatusUnauthorized},
		{name: "cluster admin can access protected user list", method: http.MethodGet, path: "/api/users", token: "cluster-admin-token", wantStatus: http.StatusOK},
		{name: "regular user cannot access protected user list", method: http.MethodGet, path: "/api/users", token: "user-token", wantStatus: http.StatusForbidden},
		{name: "cluster admin cannot access admin-only create user", method: http.MethodPost, path: "/api/users", token: "cluster-admin-token", wantStatus: http.StatusForbidden},
		{name: "admin can access admin-only create user route", method: http.MethodPost, path: "/api/users", token: "admin-token", wantStatus: http.StatusBadRequest},
		{name: "cluster admin can access ops skill list", method: http.MethodGet, path: "/api/skills", token: "cluster-admin-token", wantStatus: http.StatusOK},
		{name: "regular user cannot access ops skill list", method: http.MethodGet, path: "/api/skills", token: "user-token", wantStatus: http.StatusForbidden},
		{name: "skill execute endpoint is still stubbed", method: http.MethodPost, path: "/api/skills/1/execute", token: "cluster-admin-token", wantStatus: http.StatusNotImplemented},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.token != "" {
				req.Header.Set("Authorization", "Bearer "+tc.token)
			}
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d, body=%s", tc.wantStatus, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestSkillExecuteStubPayload(t *testing.T) {
	t.Parallel()

	router := newTestRouter(t)
	req := httptest.NewRequest(http.MethodPost, "/api/skills/1/execute", nil)
	req.Header.Set("Authorization", "Bearer cluster-admin-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	var payload struct {
		Code string `json:"code"`
		Data struct {
			Module string `json:"module"`
			Action string `json:"action"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Code != "NOT_IMPLEMENTED" {
		t.Fatalf("expected NOT_IMPLEMENTED, got %s", payload.Code)
	}
	if payload.Data.Module != "skill" || payload.Data.Action != "execute" {
		t.Fatalf("unexpected stub payload: %+v", payload.Data)
	}
}

func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	tenantID := int64(7)
	userRepo := &routerUserRepo{
		users: map[int64]*domainuser.User{
			1: {ID: 1, Username: "admin", Role: domainuser.RoleAdmin, Enabled: true},
			2: {ID: 2, Username: "ops", Role: domainuser.RoleClusterAdmin, TenantID: &tenantID, Enabled: true},
			3: {ID: 3, Username: "user", Role: domainuser.RoleUser, TenantID: &tenantID, Enabled: true},
		},
		listResult:         []applicationuser.Profile{{ID: 1, Username: "admin"}},
		listByTenantResult: []applicationuser.Profile{{ID: 2, Username: "ops", TenantID: &tenantID}},
	}

	tokenManager := routerTokenManager{}
	authService := applicationauth.NewService(userRepo, tokenManager)
	authMiddleware := middleware.NewAuthMiddleware(authService)

	userHandler := handlers.NewUserHandler(applicationuser.NewService(userRepo))
	skillHandler := handlers.NewSkillHandler(appskill.NewService(routerSkillRepo{
		items: []appskill.Record{{ID: 1, Name: "demo-skill", Status: "draft"}},
	}))

	return NewRouter(config.Config{Env: "test", AppName: "kubeclaw"}, Dependencies{
		HealthHandler:  handlers.NewHealthHandler(config.Config{AppName: "kubeclaw", Env: "test"}),
		UserHandler:    userHandler,
		SkillHandler:   skillHandler,
		StubHandler:    handlers.NewStubHandler(),
		AuthMiddleware: authMiddleware,
	})
}

type routerUserRepo struct {
	users              map[int64]*domainuser.User
	listResult         []applicationuser.Profile
	listByTenantResult []applicationuser.Profile
}

func (r *routerUserRepo) FindByID(_ context.Context, id int64) (*domainuser.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, domainuser.ErrNotFound
	}
	return user, nil
}

func (r *routerUserRepo) FindByLogin(_ context.Context, login string) (*domainuser.User, error) {
	for _, user := range r.users {
		if user.Username == login || user.Email == login {
			return user, nil
		}
	}
	return nil, domainuser.ErrNotFound
}

func (r *routerUserRepo) List(context.Context) ([]applicationuser.Profile, error) {
	return r.listResult, nil
}

func (r *routerUserRepo) ListByTenant(_ context.Context, tenantID int64) ([]applicationuser.Profile, error) {
	for _, item := range r.listByTenantResult {
		if item.TenantID != nil && *item.TenantID == tenantID {
			return r.listByTenantResult, nil
		}
	}
	return []applicationuser.Profile{}, nil
}

func (r *routerUserRepo) Get(context.Context, int64) (*applicationuser.Profile, error) {
	return nil, applicationuser.ErrNotFound
}

func (r *routerUserRepo) Create(context.Context, applicationuser.CreateInput) (*applicationuser.Profile, error) {
	return nil, nil
}

func (r *routerUserRepo) Update(context.Context, int64, applicationuser.UpdateInput) (*applicationuser.Profile, error) {
	return nil, nil
}

func (r *routerUserRepo) Delete(context.Context, int64) error {
	return nil
}

type routerTokenManager struct{}

func (routerTokenManager) Issue(identity domainauth.Identity) (domainauth.TokenPair, error) {
	return domainauth.TokenPair{AccessToken: identity.Username, RefreshToken: identity.Username + "-refresh"}, nil
}

func (routerTokenManager) Parse(token string) (*domainauth.Claims, error) {
	switch token {
	case "admin-token":
		return &domainauth.Claims{UserID: 1, Username: "admin", Role: string(domainuser.RoleAdmin), TokenType: domainauth.TokenTypeAccess, ExpiresAt: time.Now().Add(time.Hour)}, nil
	case "cluster-admin-token":
		return &domainauth.Claims{UserID: 2, Username: "ops", Role: string(domainuser.RoleClusterAdmin), TokenType: domainauth.TokenTypeAccess, ExpiresAt: time.Now().Add(time.Hour)}, nil
	case "user-token":
		return &domainauth.Claims{UserID: 3, Username: "user", Role: string(domainuser.RoleUser), TokenType: domainauth.TokenTypeAccess, ExpiresAt: time.Now().Add(time.Hour)}, nil
	default:
		return nil, errors.New("invalid token")
	}
}

type routerSkillRepo struct {
	items []appskill.Record
}

func (r routerSkillRepo) List(context.Context) ([]appskill.Record, error) {
	return r.items, nil
}

func (r routerSkillRepo) Get(context.Context, int64) (*appskill.Record, error) {
	return nil, appskill.ErrNotFound
}

func (r routerSkillRepo) Create(context.Context, appskill.CreateInput) (*appskill.Record, error) {
	return nil, nil
}

func (r routerSkillRepo) Update(context.Context, int64, appskill.UpdateInput) (*appskill.Record, error) {
	return nil, nil
}

func (r routerSkillRepo) Delete(context.Context, int64) error {
	return nil
}
