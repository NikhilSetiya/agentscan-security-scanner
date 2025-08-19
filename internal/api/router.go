package api

import (
	"github.com/gin-gonic/gin"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/findings"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/github"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/gitlab"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/orchestrator"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/queue"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
)

// Router creates and configures the API router
func NewRouter(cfg *config.Config, db *database.DB, redis *queue.RedisClient, repos *database.Repositories, orch orchestrator.OrchestrationService, q *queue.Queue, githubHandler *github.WebhookHandler, gitlabHandler *gitlab.WebhookHandler) *gin.Engine {
	// Set Gin mode based on environment
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Create services
	auditLogger := NewAuditLogger(repos)
	_ = NewRBACService(repos) // TODO: Use RBAC service in protected routes
	agentResultHandler := NewAgentResultHandler(repos, orch)

	// Add middleware
	router.Use(RequestIDMiddleware())
	router.Use(LoggingMiddleware())
	router.Use(ErrorHandlingMiddleware())
	router.Use(CORSMiddleware())
	router.Use(SecurityHeadersMiddleware())
	router.Use(auditLogger.AuditMiddleware())
	router.Use(RateLimitMiddleware(redis))

	// Health check endpoint (no auth required)
	healthHandler := NewHealthHandler(db, redis)
	router.GET("/health", gin.WrapH(healthHandler))

	// Root endpoint with API info
	router.GET("/", func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		c.String(200, `<!DOCTYPE html>
<html>
<head>
    <title>AgentScan Security Scanner</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .header { text-align: center; margin-bottom: 40px; }
        .endpoint { background: #f5f5f5; padding: 10px; margin: 10px 0; border-radius: 5px; }
        .method { color: #2563eb; font-weight: bold; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🛡️ AgentScan Security Scanner</h1>
        <p>Multi-agent security scanning platform with 80%% false positive reduction</p>
        <p><strong>Status:</strong> ✅ Running | <strong>Region:</strong> Mumbai, India</p>
    </div>
    
    <h2>Available Endpoints</h2>
    <div class="endpoint"><span class="method">GET</span> /health - Health check</div>
    <div class="endpoint"><span class="method">GET</span> /api/v1 - API information</div>
    <div class="endpoint"><span class="method">POST</span> /api/v1/auth/github/callback - GitHub OAuth</div>
    <div class="endpoint"><span class="method">POST</span> /api/v1/scans - Create security scan (requires auth)</div>
    <div class="endpoint"><span class="method">GET</span> /api/v1/scans - List scans (requires auth)</div>
    
    <h2>Quick Test</h2>
    <p>Try: <a href="/api/v1" target="_blank">/api/v1</a> or <a href="/health" target="_blank">/health</a></p>
    
    <h2>Frontend Dashboard</h2>
    <p>React dashboard available - needs separate deployment or static build.</p>
</body>
</html>`)
	})

	// API version info (no auth required)
	router.GET("/api/v1", func(c *gin.Context) {
		SuccessResponse(c, map[string]interface{}{
			"name":    "AgentScan API",
			"version": "1.0.0",
			"status":  "ok",
		})
	})

	// Create handlers
	authHandler := NewAuthHandler(cfg, repos, auditLogger)
	scanHandler := NewScanHandler(repos, orch, q)
	dashboardHandler := NewDashboardHandler(repos)
	repositoryHandler := NewRepositoryHandler(repos)
	
	// Create findings service and handler
	findingsExporter := findings.NewExportService("http://localhost:8080") // TODO: Use config
	findingsService := findings.NewService(db.DB, findingsExporter)
	findingsHandler := NewFindingsHandler(findingsService)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Authentication routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login) // Simple login for testing
			auth.GET("/github/url", authHandler.GetAuthURL)
			auth.POST("/github/callback", authHandler.LoginWithGitHub)
			auth.GET("/gitlab/url", authHandler.GetGitLabAuthURL)
			auth.POST("/gitlab/callback", authHandler.LoginWithGitLab)
			auth.POST("/logout", authHandler.Logout)
		}

		// Protected routes (require authentication)
		protected := v1.Group("")
		protected.Use(AuthMiddleware(cfg))
		{
			// Dashboard routes
			dashboard := protected.Group("/dashboard")
			{
				dashboard.GET("/stats", dashboardHandler.GetStats)
				dashboard.GET("/trends", dashboardHandler.GetScanTrends)
				dashboard.GET("/health", dashboardHandler.GetSystemHealth)
				dashboard.GET("/repositories/:id/stats", dashboardHandler.GetRepositoryStats)
			}

			// User routes
			user := protected.Group("/user")
			{
				user.GET("/me", authHandler.GetCurrentUserInfo)
				user.PUT("/me", authHandler.UpdateUserProfile)
				user.DELETE("/me", authHandler.DeleteUserAccount)
				user.POST("/refresh", authHandler.RefreshToken)
			}

			// Repository routes
			repositories := protected.Group("/repositories")
			{
				repositories.GET("", repositoryHandler.ListRepositories)
				repositories.POST("", repositoryHandler.CreateRepository)
				repositories.GET("/:id", repositoryHandler.GetRepository)
				repositories.PUT("/:id", repositoryHandler.UpdateRepository)
				repositories.DELETE("/:id", repositoryHandler.DeleteRepository)
				repositories.GET("/:id/scans", repositoryHandler.GetRepositoryScans)
			}

			// Scan routes
			scans := protected.Group("/scans")
			{
				scans.POST("", scanHandler.CreateScan)
				scans.GET("", scanHandler.ListScans)
				scans.GET("/metrics", scanHandler.GetScanMetrics)
				scans.GET("/:id", scanHandler.GetScan)
				scans.GET("/:id/status", scanHandler.GetScanStatus)
				scans.GET("/:id/results", scanHandler.GetScanResults)
				scans.POST("/:id/cancel", scanHandler.CancelScan)
				scans.POST("/:id/retry", scanHandler.RetryFailedScan)
				scans.PATCH("/:id/status", scanHandler.UpdateScanStatus) // Internal use
			}

			// Finding routes
			findings := protected.Group("/findings")
			{
				findings.GET("", findingsHandler.ListFindings)
				findings.GET("/:id", findingsHandler.GetFinding)
				findings.PATCH("/:id/status", findingsHandler.UpdateFindingStatus)
				findings.POST("/:id/suppress", findingsHandler.SuppressFinding)
				findings.PATCH("/bulk/status", findingsHandler.BulkUpdateFindings)
				findings.POST("/export", findingsHandler.ExportFindings)
			}

			// Finding suppressions routes
			suppressions := protected.Group("/suppressions")
			{
				suppressions.GET("", findingsHandler.GetSuppressions)
				suppressions.DELETE("/:id", findingsHandler.DeleteSuppression)
			}

			// Finding stats routes
			stats := protected.Group("/stats")
			{
				stats.GET("/findings/:scan_job_id", findingsHandler.GetFindingStats)
			}

			// User feedback routes
			feedback := protected.Group("/feedback")
			{
				feedback.GET("", findingsHandler.GetUserFeedback)
			}
		}

		// Internal routes (for service-to-service communication)
		// These should be protected by API keys or internal network access
		internal := v1.Group("/internal")
		{
			// Webhook endpoints for CI/CD integrations
			webhooks := internal.Group("/webhooks")
			{
				// GitHub webhook handler
				webhooks.POST("/github", func(c *gin.Context) {
					if githubHandler != nil {
						githubHandler.HandleWebhook(c.Writer, c.Request)
					} else {
						SuccessResponse(c, map[string]string{
							"message": "GitHub webhook received (handler not configured)",
						})
					}
				})

				// GitLab webhook handler
				webhooks.POST("/gitlab", func(c *gin.Context) {
					if gitlabHandler != nil {
						gitlabHandler.HandleWebhook(c.Writer, c.Request)
					} else {
						SuccessResponse(c, map[string]string{
							"message": "GitLab webhook received (handler not configured)",
						})
					}
				})
			}

			// Agent callback endpoints
			agents := internal.Group("/agents")
			{
				// Agent health check
				agents.GET("/health", func(c *gin.Context) {
					SuccessResponse(c, map[string]string{
						"status": "ok",
					})
				})

				// Agent result submission
				agents.POST("/results", agentResultHandler.SubmitResults)
			}
		}
	}

	// Catch-all route for undefined endpoints
	router.NoRoute(func(c *gin.Context) {
		NotFoundResponse(c, "Endpoint not found")
	})

	return router
}

// SetupRoutes is a convenience function to set up routes with all dependencies
func SetupRoutes(cfg *config.Config, db *database.DB, redis *queue.RedisClient, repos *database.Repositories, orch orchestrator.OrchestrationService, q *queue.Queue) *gin.Engine {
	// Initialize GitHub service if configured
	var githubHandler *github.WebhookHandler
	if cfg.GitHub.AppID != 0 && cfg.GitHub.PrivateKey != "" {
		githubService := github.NewService(cfg, repos)
		githubHandler = github.NewWebhookHandler(repos, orch.(*orchestrator.Service), githubService)
	}
	
	// Initialize GitLab service if configured
	var gitlabHandler *gitlab.WebhookHandler
	if cfg.Auth.GitLabClientID != "" && cfg.Auth.GitLabSecret != "" {
		gitlabService := gitlab.NewService(cfg, repos)
		gitlabHandler = gitlab.NewWebhookHandler(repos, orch.(*orchestrator.Service), gitlabService)
	}
	
	return NewRouter(cfg, db, redis, repos, orch, q, githubHandler, gitlabHandler)
}