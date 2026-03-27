package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	applicationuser "kubeclaw/backend/internal/application/user"
	domainuser "kubeclaw/backend/internal/domain/user"
	"kubeclaw/backend/internal/httpapi/middleware"

	"github.com/gin-gonic/gin"
)

func TestUserHandlerListAllowsAdminAndCallsGlobalList(t *testing.T) {
	t.Parallel()

	repo := &stubUserRepository{
		listResult: []applicationuser.Profile{{ID: 1, Username: "alice"}},
	}
	handler := NewUserHandler(applicationuser.NewService(repo))
	router := gin.New()
	router.GET("/users", withCurrentUser(&domainuser.User{ID: 10, Role: domainuser.RoleAdmin}), handler.List)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if repo.listCalls != 1 {
		t.Fatalf("expected List to be called once, got %d", repo.listCalls)
	}
	if repo.listByTenantCalls != 0 {
		t.Fatalf("expected ListByTenant not to be called, got %d", repo.listByTenantCalls)
	}
}

func TestUserHandlerListClusterAdminUsesTenantScopedQuery(t *testing.T) {
	t.Parallel()

	tenantID := int64(7)
	repo := &stubUserRepository{
		listByTenantResult: []applicationuser.Profile{{ID: 2, Username: "bob", TenantID: &tenantID}},
	}
	handler := NewUserHandler(applicationuser.NewService(repo))
	router := gin.New()
	router.GET("/users", withCurrentUser(&domainuser.User{ID: 20, Role: domainuser.RoleClusterAdmin, TenantID: &tenantID}), handler.List)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if repo.listCalls != 0 {
		t.Fatalf("expected global List not to be called, got %d", repo.listCalls)
	}
	if repo.listByTenantCalls != 1 {
		t.Fatalf("expected ListByTenant to be called once, got %d", repo.listByTenantCalls)
	}
	if repo.lastTenantID != tenantID {
		t.Fatalf("expected tenant id %d, got %d", tenantID, repo.lastTenantID)
	}
}

func TestUserHandlerListRejectsRegularUser(t *testing.T) {
	t.Parallel()

	repo := &stubUserRepository{}
	handler := NewUserHandler(applicationuser.NewService(repo))
	router := gin.New()
	router.GET("/users", withCurrentUser(&domainuser.User{ID: 30, Role: domainuser.RoleUser}), handler.List)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status 403, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["code"] != "FORBIDDEN" {
		t.Fatalf("expected code FORBIDDEN, got %#v", payload["code"])
	}
}

func withCurrentUser(currentUser *domainuser.User) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(middleware.CurrentUserKey, currentUser)
		c.Next()
	}
}

type stubUserRepository struct {
	listCalls          int
	listByTenantCalls  int
	lastTenantID       int64
	listResult         []applicationuser.Profile
	listByTenantResult []applicationuser.Profile
}

func (r *stubUserRepository) FindByID(context.Context, int64) (*domainuser.User, error) {
	return nil, domainuser.ErrNotFound
}

func (r *stubUserRepository) FindByLogin(context.Context, string) (*domainuser.User, error) {
	return nil, domainuser.ErrNotFound
}

func (r *stubUserRepository) List(context.Context) ([]applicationuser.Profile, error) {
	r.listCalls++
	return r.listResult, nil
}

func (r *stubUserRepository) ListByTenant(_ context.Context, tenantID int64) ([]applicationuser.Profile, error) {
	r.listByTenantCalls++
	r.lastTenantID = tenantID
	return r.listByTenantResult, nil
}

func (r *stubUserRepository) Get(context.Context, int64) (*applicationuser.Profile, error) {
	return nil, applicationuser.ErrNotFound
}

func (r *stubUserRepository) Create(context.Context, applicationuser.CreateInput) (*applicationuser.Profile, error) {
	return nil, nil
}

func (r *stubUserRepository) Update(context.Context, int64, applicationuser.UpdateInput) (*applicationuser.Profile, error) {
	return nil, nil
}

func (r *stubUserRepository) Delete(context.Context, int64) error {
	return nil
}
