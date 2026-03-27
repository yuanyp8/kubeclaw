package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	applicationauth "kubeclaw/backend/internal/application/auth"
	domainauth "kubeclaw/backend/internal/domain/auth"
	domainuser "kubeclaw/backend/internal/domain/user"

	"github.com/gin-gonic/gin"
)

func TestExtractBearerToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		header        string
		expectedToken string
	}{
		{name: "valid bearer token", header: "Bearer abc123", expectedToken: "abc123"},
		{name: "case insensitive scheme", header: "bearer xyz", expectedToken: "xyz"},
		{name: "missing scheme", header: "abc123", expectedToken: ""},
		{name: "wrong scheme", header: "Basic abc123", expectedToken: ""},
		{name: "empty token", header: "Bearer   ", expectedToken: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ExtractBearerToken(tc.header); got != tc.expectedToken {
				t.Fatalf("expected token %q, got %q", tc.expectedToken, got)
			}
		})
	}
}

func TestRequireAuthRejectsMissingBearerToken(t *testing.T) {
	t.Parallel()

	router := gin.New()
	router.Use(RequestID())
	router.Use(newTestAuthMiddleware().RequireAuth())
	router.GET("/protected", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["code"] != "UNAUTHORIZED" {
		t.Fatalf("expected code UNAUTHORIZED, got %#v", payload["code"])
	}
}

func TestRequireRolesRejectsUserWithoutRequiredRole(t *testing.T) {
	t.Parallel()

	middleware := newTestAuthMiddleware()
	router := gin.New()
	router.Use(RequestID())
	router.Use(middleware.RequireAuth())
	router.Use(middleware.RequireRoles(domainuser.RoleAdmin))
	router.GET("/admin", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer user-token")
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

func TestRequireRolesAllowsAdmin(t *testing.T) {
	t.Parallel()

	middleware := newTestAuthMiddleware()
	router := gin.New()
	router.Use(RequestID())
	router.Use(middleware.RequireAuth())
	router.Use(middleware.RequireRoles(domainuser.RoleAdmin))
	router.GET("/admin", func(c *gin.Context) {
		currentUser, ok := CurrentUser(c)
		if !ok {
			t.Fatal("expected current user in context")
		}
		c.JSON(http.StatusOK, gin.H{"user": currentUser.Username})
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("Authorization", "Bearer admin-token")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func newTestAuthMiddleware() *AuthMiddleware {
	tenantID := int64(42)
	userRepo := stubAuthUserRepo{
		users: map[int64]*domainuser.User{
			1: {
				ID:       1,
				Username: "admin",
				Role:     domainuser.RoleAdmin,
				Enabled:  true,
			},
			2: {
				ID:       2,
				Username: "alice",
				Role:     domainuser.RoleUser,
				TenantID: &tenantID,
				Enabled:  true,
			},
		},
	}
	tokenManager := stubTokenManager{
		parseFn: func(token string) (*domainauth.Claims, error) {
			switch token {
			case "admin-token":
				return &domainauth.Claims{UserID: 1, Username: "admin", Role: string(domainuser.RoleAdmin), TokenType: domainauth.TokenTypeAccess, ExpiresAt: time.Now().Add(time.Hour)}, nil
			case "user-token":
				return &domainauth.Claims{UserID: 2, Username: "alice", Role: string(domainuser.RoleUser), TokenType: domainauth.TokenTypeAccess, ExpiresAt: time.Now().Add(time.Hour)}, nil
			default:
				return nil, domainauth.ErrInvalidToken
			}
		},
	}
	return NewAuthMiddleware(applicationauth.NewService(userRepo, tokenManager))
}

type stubAuthUserRepo struct {
	users map[int64]*domainuser.User
}

func (r stubAuthUserRepo) FindByID(_ context.Context, id int64) (*domainuser.User, error) {
	user, ok := r.users[id]
	if !ok {
		return nil, domainuser.ErrNotFound
	}
	return user, nil
}

func (r stubAuthUserRepo) FindByLogin(_ context.Context, login string) (*domainuser.User, error) {
	for _, user := range r.users {
		if user.Username == login || user.Email == login {
			return user, nil
		}
	}
	return nil, domainuser.ErrNotFound
}

type stubTokenManager struct {
	parseFn func(token string) (*domainauth.Claims, error)
}

func (m stubTokenManager) Issue(identity domainauth.Identity) (domainauth.TokenPair, error) {
	return domainauth.TokenPair{AccessToken: identity.Username, RefreshToken: identity.Username + "-refresh"}, nil
}

func (m stubTokenManager) Parse(token string) (*domainauth.Claims, error) {
	if m.parseFn == nil {
		return nil, errors.New("parse function is not configured")
	}
	return m.parseFn(token)
}
