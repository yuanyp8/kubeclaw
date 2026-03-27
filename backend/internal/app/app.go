package app

import (
	"context"
	"fmt"

	applicationagent "kubeclaw/backend/internal/application/agent"
	applicationaudit "kubeclaw/backend/internal/application/audit"
	applicationauth "kubeclaw/backend/internal/application/auth"
	applicationcapability "kubeclaw/backend/internal/application/capability"
	applicationchat "kubeclaw/backend/internal/application/chat"
	applicationcluster "kubeclaw/backend/internal/application/cluster"
	applicationlogs "kubeclaw/backend/internal/application/logs"
	applicationmcp "kubeclaw/backend/internal/application/mcp"
	applicationmodel "kubeclaw/backend/internal/application/model"
	appsecurity "kubeclaw/backend/internal/application/security"
	appskill "kubeclaw/backend/internal/application/skill"
	applicationteam "kubeclaw/backend/internal/application/team"
	applicationtenant "kubeclaw/backend/internal/application/tenant"
	applicationuser "kubeclaw/backend/internal/application/user"
	"kubeclaw/backend/internal/config"
	"kubeclaw/backend/internal/httpapi"
	"kubeclaw/backend/internal/httpapi/handlers"
	"kubeclaw/backend/internal/httpapi/middleware"
	"kubeclaw/backend/internal/infrastructure/agentruntime"
	jwtauth "kubeclaw/backend/internal/infrastructure/auth"
	cryptoinfra "kubeclaw/backend/internal/infrastructure/crypto"
	k8sinfra "kubeclaw/backend/internal/infrastructure/kubernetes"
	"kubeclaw/backend/internal/infrastructure/llm"
	mysqlinfra "kubeclaw/backend/internal/infrastructure/mysql"
	mysqlrepo "kubeclaw/backend/internal/infrastructure/mysql/repository"
	"kubeclaw/backend/internal/logger"
)

type App struct {
	server *httpapi.Server
}

func New(cfg config.Config) (*App, error) {
	ctx := context.Background()

	database, err := mysqlinfra.Open(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open mysql database: %w", err)
	}

	if cfg.MySQLAutoMigrate {
		if err := mysqlinfra.AutoMigrate(database.Gorm); err != nil {
			return nil, fmt.Errorf("auto migrate mysql schema: %w", err)
		}
	}

	if err := mysqlinfra.Bootstrap(ctx, database.Gorm, cfg); err != nil {
		return nil, fmt.Errorf("bootstrap mysql data: %w", err)
	}

	secretBox := cryptoinfra.NewSecretBox(cfg.DataSecret)

	userRepo := mysqlrepo.NewUserRepository(database.Gorm)
	tenantRepo := mysqlrepo.NewTenantRepository(database.Gorm)
	teamRepo := mysqlrepo.NewTeamRepository(database.Gorm)
	auditRepo := mysqlrepo.NewAuditRepository(database.Gorm)
	modelRepo := mysqlrepo.NewModelRepository(database.Gorm, secretBox)
	clusterRepo := mysqlrepo.NewClusterRepository(database.Gorm, secretBox)
	mcpRepo := mysqlrepo.NewMCPRepository(database.Gorm, secretBox)
	skillRepo := mysqlrepo.NewSkillRepository(database.Gorm)
	securityRepo := mysqlrepo.NewSecurityRepository(database.Gorm)
	chatRepo := mysqlrepo.NewChatRepository(database.Gorm)
	agentRepo := mysqlrepo.NewAgentRepository(database.Gorm)

	k8sGateway := k8sinfra.NewGateway()
	llmClient := llm.NewOpenAICompatibleClient()
	streams := agentruntime.NewHub[applicationagent.Event]()

	tokenManager := jwtauth.NewJWTManager(cfg.JWTSecret, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)
	authService := applicationauth.NewService(userRepo, tokenManager)
	userService := applicationuser.NewService(userRepo)
	tenantService := applicationtenant.NewService(tenantRepo)
	teamService := applicationteam.NewService(teamRepo)
	auditService := applicationaudit.NewService(auditRepo)
	logService := applicationlogs.NewService(logger.GlobalBus())
	modelService := applicationmodel.NewService(modelRepo, llmClient)
	clusterService := applicationcluster.NewService(clusterRepo, k8sGateway)
	mcpService := applicationmcp.NewService(mcpRepo)
	skillService := appskill.NewService(skillRepo)
	capabilityService := applicationcapability.NewService(skillService, mcpService).WithRuntime(modelService, clusterService, llmClient)
	securityService := appsecurity.NewService(securityRepo)
	chatService := applicationchat.NewService(chatRepo)
	agentService := applicationagent.NewService(
		agentRepo,
		chatService,
		modelService,
		clusterService,
		capabilityService,
		skillService,
		mcpService,
		securityService,
		llmClient,
		streams,
	)

	authMiddleware := middleware.NewAuthMiddleware(authService)
	auditMiddleware := middleware.NewAuditMiddleware(auditService)

	server, err := httpapi.NewServer(cfg, httpapi.Dependencies{
		HealthHandler:     handlers.NewHealthHandler(cfg),
		AuthHandler:       handlers.NewAuthHandler(authService),
		CapabilityHandler: handlers.NewCapabilityHandler(capabilityService, agentService),
		UserHandler:       handlers.NewUserHandler(userService),
		TenantHandler:     handlers.NewTenantHandler(tenantService),
		TeamHandler:       handlers.NewTeamHandler(teamService),
		AuditHandler:      handlers.NewAuditHandler(auditService),
		LogHandler:        handlers.NewLogHandler(logService, auditService),
		ModelHandler:      handlers.NewModelHandler(modelService),
		ClusterHandler:    handlers.NewClusterHandler(clusterService, agentService),
		MCPHandler:        handlers.NewMCPHandler(mcpService),
		SkillHandler:      handlers.NewSkillHandler(skillService),
		SecurityHandler:   handlers.NewSecurityHandler(securityService),
		AgentHandler:      handlers.NewAgentHandler(agentService, streams),
		StubHandler:       handlers.NewStubHandler(),
		AuthMiddleware:    authMiddleware,
		AuditMiddleware:   auditMiddleware,
	})
	if err != nil {
		return nil, fmt.Errorf("new http server: %w", err)
	}

	return &App{server: server}, nil
}

func (a *App) Run(ctx context.Context) error {
	return a.server.Run(ctx)
}
