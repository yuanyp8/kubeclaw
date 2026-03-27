package httpapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"kubeclaw/backend/internal/config"
	"kubeclaw/backend/internal/httpapi/handlers"
	"kubeclaw/backend/internal/httpapi/middleware"
)

type Dependencies struct {
	HealthHandler     *handlers.HealthHandler
	AuthHandler       *handlers.AuthHandler
	CapabilityHandler *handlers.CapabilityHandler
	UserHandler       *handlers.UserHandler
	TenantHandler     *handlers.TenantHandler
	TeamHandler       *handlers.TeamHandler
	AuditHandler      *handlers.AuditHandler
	LogHandler        *handlers.LogHandler
	ModelHandler      *handlers.ModelHandler
	ClusterHandler    *handlers.ClusterHandler
	MCPHandler        *handlers.MCPHandler
	SkillHandler      *handlers.SkillHandler
	SecurityHandler   *handlers.SecurityHandler
	AgentHandler      *handlers.AgentHandler
	StubHandler       *handlers.StubHandler
	AuthMiddleware    *middleware.AuthMiddleware
	AuditMiddleware   *middleware.AuditMiddleware
}

// Server 包装 http.Server，负责启动与优雅关闭。
type Server struct {
	httpServer      *http.Server
	shutdownTimeout time.Duration
}

func NewServer(cfg config.Config, deps Dependencies) (*Server, error) {
	handler := NewRouter(cfg, deps)

	return &Server{
		httpServer: &http.Server{
			Addr:              cfg.HTTPAddr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       cfg.ReadTimeout,
			WriteTimeout:      cfg.WriteTimeout,
		},
		shutdownTimeout: cfg.ShutdownTimeout,
	}, nil
}

func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("listen and serve: %w", err)
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
		return nil
	}
}
