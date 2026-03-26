package httpapi

import (
	"github.com/gin-gonic/gin"

	"kubeclaw/backend/internal/config"
	domainuser "kubeclaw/backend/internal/domain/user"
	"kubeclaw/backend/internal/httpapi/middleware"
)

func NewRouter(cfg config.Config, deps Dependencies) *gin.Engine {
	setGinMode(cfg.Env)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.AccessLog())

	router.GET("/healthz", deps.HealthHandler.Get)
	router.GET("/readyz", deps.HealthHandler.Ready)

	api := router.Group("/api")
	{
		if deps.AuditMiddleware != nil {
			api.Use(deps.AuditMiddleware.Record())
		}

		registerPublicAuthRoutes(api.Group("/auth"), deps)

		protected := api.Group("")
		protected.Use(deps.AuthMiddleware.RequireAuth())
		{
			registerProtectedAuthRoutes(protected.Group("/auth"), deps)
			protected.GET("/users/me", deps.UserHandler.GetMe)
			protected.PUT("/users/me", deps.StubHandler.Handle("user", "update_profile"))

			if deps.LogHandler != nil {
				protected.GET("/logs", deps.LogHandler.List)
				protected.GET("/logs/scopes", deps.LogHandler.ListScopes)
				protected.POST("/logs/client", deps.LogHandler.CreateClientLog)
			}

			if deps.AgentHandler != nil {
				registerAgentRoutes(protected.Group("/agent"), deps)
			}
		}

		admin := api.Group("")
		admin.Use(
			deps.AuthMiddleware.RequireAuth(),
			deps.AuthMiddleware.RequireRoles(domainuser.RoleAdmin),
		)
		{
			registerAdminUserRoutes(admin, deps)
			registerModelRoutes(admin.Group("/models"), deps)
			registerMCPRoutes(admin.Group("/mcp/servers"), deps)
			registerSecurityRoutes(admin.Group("/security"), deps)
			registerTenantRoutes(admin.Group("/tenants"), deps)
			registerAuditRoutes(admin.Group("/audit"), deps)
		}

		ops := api.Group("")
		ops.Use(
			deps.AuthMiddleware.RequireAuth(),
			deps.AuthMiddleware.RequireRoles(domainuser.RoleAdmin, domainuser.RoleClusterAdmin),
		)
		{
			registerClusterRoutes(ops.Group("/clusters"), deps)
			registerSkillRoutes(ops.Group("/skills"), deps)
			registerTeamRoutes(ops.Group("/teams"), deps)
			registerKnowledgeRoutes(ops.Group("/knowledge"), deps)
			registerTaskRoutes(ops.Group("/tasks"), deps)
		}
	}

	return router
}

func setGinMode(env string) {
	if env == "prod" {
		gin.SetMode(gin.ReleaseMode)
		return
	}
	gin.SetMode(gin.DebugMode)
}

func registerPublicAuthRoutes(group *gin.RouterGroup, deps Dependencies) {
	group.POST("/login", deps.AuthHandler.Login)
	group.POST("/refresh", deps.AuthHandler.Refresh)
}

func registerProtectedAuthRoutes(group *gin.RouterGroup, deps Dependencies) {
	group.POST("/logout", deps.AuthHandler.Logout)
}

func registerAdminUserRoutes(group *gin.RouterGroup, deps Dependencies) {
	group.GET("/users", deps.UserHandler.List)
	group.POST("/users", deps.UserHandler.Create)
	group.GET("/users/:id", deps.UserHandler.Get)
	group.PUT("/users/:id", deps.UserHandler.Update)
	group.DELETE("/users/:id", deps.UserHandler.Delete)
}

func registerTenantRoutes(group *gin.RouterGroup, deps Dependencies) {
	group.GET("", deps.TenantHandler.List)
	group.POST("", deps.TenantHandler.Create)
	group.GET("/:id", deps.TenantHandler.Get)
	group.PUT("/:id", deps.TenantHandler.Update)
	group.DELETE("/:id", deps.TenantHandler.Delete)
}

func registerModelRoutes(group *gin.RouterGroup, deps Dependencies) {
	group.GET("", deps.ModelHandler.List)
	group.POST("", deps.ModelHandler.Create)
	group.GET("/:id", deps.ModelHandler.Get)
	group.PUT("/:id", deps.ModelHandler.Update)
	group.DELETE("/:id", deps.ModelHandler.Delete)
	group.POST("/:id/test", deps.ModelHandler.Test)
	group.POST("/:id/set-default", deps.ModelHandler.SetDefault)
}

func registerTeamRoutes(group *gin.RouterGroup, deps Dependencies) {
	group.GET("", deps.TeamHandler.List)
	group.POST("", deps.TeamHandler.Create)
	group.GET("/:id", deps.TeamHandler.Get)
	group.PUT("/:id", deps.TeamHandler.Update)
	group.DELETE("/:id", deps.TeamHandler.Delete)
	group.GET("/:id/members", deps.TeamHandler.ListMembers)
	group.POST("/:id/members", deps.TeamHandler.AddMember)
	group.DELETE("/:id/members/:userId", deps.TeamHandler.RemoveMember)
}

func registerClusterRoutes(group *gin.RouterGroup, deps Dependencies) {
	group.GET("", deps.ClusterHandler.List)
	group.POST("", deps.ClusterHandler.Create)
	group.GET("/:id", deps.ClusterHandler.Get)
	group.PUT("/:id", deps.ClusterHandler.Update)
	group.DELETE("/:id", deps.ClusterHandler.Delete)
	group.POST("/:id/validate", deps.ClusterHandler.Validate)
	group.GET("/:id/overview", deps.ClusterHandler.Overview)
	group.POST("/:id/share", deps.ClusterHandler.Share)
	group.GET("/:id/permissions", deps.ClusterHandler.ListPermissions)
	group.GET("/:id/namespaces", deps.ClusterHandler.ListNamespaces)
	group.GET("/:id/resources", deps.ClusterHandler.ListResources)
	group.GET("/:id/resources/:type/:name", deps.ClusterHandler.GetResource)
	group.GET("/:id/pods/:name/logs", deps.ClusterHandler.StreamPodLogs)
	group.GET("/:id/events", deps.ClusterHandler.ListEvents)
	group.POST("/:id/actions/delete-resource", deps.ClusterHandler.RequestDeleteResource)
	group.POST("/:id/actions/scale-deployment", deps.ClusterHandler.RequestScaleDeployment)
	group.POST("/:id/actions/restart-deployment", deps.ClusterHandler.RequestRestartDeployment)
	group.POST("/:id/actions/apply-yaml", deps.ClusterHandler.RequestApplyYAML)
	group.POST("/:id/resources", deps.StubHandler.Handle("k8s", "apply_resource_yaml"))
	group.DELETE("/:id/resources/:type/:name", deps.StubHandler.Handle("k8s", "delete_resource"))
	group.GET("/:id/exec", deps.StubHandler.Handle("k8s", "exec"))
}

func registerAgentRoutes(group *gin.RouterGroup, deps Dependencies) {
	sessions := group.Group("/sessions")
	{
		sessions.POST("", deps.AgentHandler.CreateSession)
		sessions.GET("", deps.AgentHandler.ListSessions)
		sessions.GET("/:id", deps.AgentHandler.GetSession)
		sessions.GET("/:id/messages", deps.AgentHandler.ListMessages)
		sessions.POST("/:id/messages", deps.AgentHandler.SendMessage)
		sessions.DELETE("/:id", deps.AgentHandler.DeleteSession)
	}

	runs := group.Group("/runs")
	{
		runs.GET("/:id/events", deps.AgentHandler.ListRunEvents)
		runs.GET("/:id/stream", deps.AgentHandler.StreamRun)
	}

	approvals := group.Group("/approvals")
	{
		approvals.POST("/:id/approve", deps.AgentHandler.Approve)
		approvals.POST("/:id/reject", deps.AgentHandler.Reject)
	}
}

func registerMCPRoutes(group *gin.RouterGroup, deps Dependencies) {
	group.GET("", deps.MCPHandler.List)
	group.POST("", deps.MCPHandler.Create)
	group.GET("/:id", deps.MCPHandler.Get)
	group.PUT("/:id", deps.MCPHandler.Update)
	group.DELETE("/:id", deps.MCPHandler.Delete)
}

func registerSkillRoutes(group *gin.RouterGroup, deps Dependencies) {
	group.GET("", deps.SkillHandler.List)
	group.POST("", deps.SkillHandler.Create)
	group.GET("/:id", deps.SkillHandler.Get)
	group.PUT("/:id", deps.SkillHandler.Update)
	group.DELETE("/:id", deps.SkillHandler.Delete)
	group.POST("/:id/execute", deps.StubHandler.Handle("skill", "execute"))
}

func registerKnowledgeRoutes(group *gin.RouterGroup, deps Dependencies) {
	group.POST("", deps.StubHandler.Handle("knowledge", "upload"))
	group.GET("", deps.StubHandler.Handle("knowledge", "list"))
	group.GET("/:id", deps.StubHandler.Handle("knowledge", "detail"))
	group.DELETE("/:id", deps.StubHandler.Handle("knowledge", "delete"))
	group.POST("/search", deps.StubHandler.Handle("knowledge", "search"))
}

func registerTaskRoutes(group *gin.RouterGroup, deps Dependencies) {
	group.GET("", deps.StubHandler.Handle("task", "list"))
	group.POST("", deps.StubHandler.Handle("task", "create"))
	group.GET("/:id", deps.StubHandler.Handle("task", "detail"))
	group.PUT("/:id", deps.StubHandler.Handle("task", "update"))
	group.DELETE("/:id", deps.StubHandler.Handle("task", "delete"))
	group.POST("/:id/run", deps.StubHandler.Handle("task", "run_once"))
	group.GET("/:id/history", deps.StubHandler.Handle("task", "history"))
}

func registerSecurityRoutes(group *gin.RouterGroup, deps Dependencies) {
	group.GET("/ip-whitelists", deps.SecurityHandler.ListIPWhitelists)
	group.POST("/ip-whitelists", deps.SecurityHandler.CreateIPWhitelist)
	group.GET("/ip-whitelists/:id", deps.SecurityHandler.GetIPWhitelist)
	group.PUT("/ip-whitelists/:id", deps.SecurityHandler.UpdateIPWhitelist)
	group.DELETE("/ip-whitelists/:id", deps.SecurityHandler.DeleteIPWhitelist)

	group.GET("/sensitive-words", deps.SecurityHandler.ListSensitiveWords)
	group.POST("/sensitive-words", deps.SecurityHandler.CreateSensitiveWord)
	group.GET("/sensitive-words/:id", deps.SecurityHandler.GetSensitiveWord)
	group.PUT("/sensitive-words/:id", deps.SecurityHandler.UpdateSensitiveWord)
	group.DELETE("/sensitive-words/:id", deps.SecurityHandler.DeleteSensitiveWord)

	group.GET("/sensitive-field-rules", deps.SecurityHandler.ListSensitiveFieldRules)
	group.POST("/sensitive-field-rules", deps.SecurityHandler.CreateSensitiveFieldRule)
	group.GET("/sensitive-field-rules/:id", deps.SecurityHandler.GetSensitiveFieldRule)
	group.PUT("/sensitive-field-rules/:id", deps.SecurityHandler.UpdateSensitiveFieldRule)
	group.DELETE("/sensitive-field-rules/:id", deps.SecurityHandler.DeleteSensitiveFieldRule)
}

func registerAuditRoutes(group *gin.RouterGroup, deps Dependencies) {
	group.GET("", deps.AuditHandler.List)
	group.GET("/:id", deps.AuditHandler.Get)
}
