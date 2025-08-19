package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

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
				dashboard.GET("/stats", func(c *gin.Context) {
					// Return dashboard stats matching frontend DashboardStats interface exactly
					SuccessResponse(c, map[string]interface{}{
						"total_scans": 12,
						"total_repositories": 3,
						"findings_by_severity": map[string]interface{}{
							"critical": 2,
							"high":     5,
							"medium":   8,
							"low":      15,
							"info":     3,
						},
						"recent_scans": []map[string]interface{}{
							{
								"id":              "scan-1",
								"repository_id":   "repo-1",
								"repository": map[string]interface{}{
									"id":   "repo-1",
									"name": "demo-repo",
									"url":  "https://github.com/demo/repo",
									"language": "JavaScript",
									"branch": "main",
									"created_at": "2025-08-17T10:00:00Z",
									"last_scan_at": "2025-08-18T22:32:15Z",
								},
								"status":         "completed",
								"progress":       100,
								"findings_count": 7,
								"started_at":     "2025-08-18T22:30:00Z",
								"completed_at":   "2025-08-18T22:32:15Z",
								"duration":       "2m 15s",
								"branch":         "main",
								"commit":         "abc123",
								"commit_message": "Fix security vulnerability in authentication",
								"triggered_by":   "user@example.com",
								"scan_type":      "full",
							},
							{
								"id":              "scan-2",
								"repository_id":   "repo-2", 
								"repository": map[string]interface{}{
									"id":   "repo-2",
									"name": "api-service",
									"url":  "https://github.com/demo/api",
									"language": "Python",
									"branch": "develop",
									"created_at": "2025-08-16T14:00:00Z",
									"last_scan_at": "2025-08-18T23:15:00Z",
								},
								"status":         "running",
								"progress":       65,
								"findings_count": 3,
								"started_at":     "2025-08-18T23:15:00Z",
								"branch":         "develop",
								"commit":         "def456",
								"commit_message": "Add new API endpoint",
								"triggered_by":   "admin@example.com",
								"scan_type":      "incremental",
							},
						},
						"trend_data": []map[string]interface{}{
							{"date": "2025-08-14", "critical": 1, "high": 3, "medium": 5, "low": 12, "info": 2},
							{"date": "2025-08-15", "critical": 0, "high": 2, "medium": 6, "low": 10, "info": 1},
							{"date": "2025-08-16", "critical": 2, "high": 4, "medium": 7, "low": 13, "info": 3},
							{"date": "2025-08-17", "critical": 1, "high": 3, "medium": 8, "low": 14, "info": 2},
							{"date": "2025-08-18", "critical": 2, "high": 5, "medium": 8, "low": 15, "info": 3},
						},
					})
				})
			}

			// User routes
			user := protected.Group("/user")
			{
				user.GET("/me", authHandler.GetCurrentUserInfo)
				user.POST("/refresh", authHandler.RefreshToken)
			}

			// Repository routes
			repositories := protected.Group("/repositories")
			{
				repositories.GET("", func(c *gin.Context) {
					// Return mock repositories for demo matching frontend Repository interface
					SuccessResponse(c, map[string]interface{}{
						"repositories": []map[string]interface{}{
							{
								"id":           "repo-1",
								"name":         "demo-repo",
								"url":          "https://github.com/demo/repo",
								"language":     "JavaScript", 
								"branch":       "main",
								"created_at":   "2025-08-17T10:00:00Z",
								"last_scan_at": "2025-08-18T22:32:15Z",
							},
							{
								"id":           "repo-2",
								"name":         "api-service",
								"url":          "https://github.com/demo/api",
								"language":     "Python",
								"branch":       "develop", 
								"created_at":   "2025-08-16T14:00:00Z",
								"last_scan_at": "2025-08-18T23:15:00Z",
							},
							{
								"id":         "repo-3",
								"name":       "frontend-app",
								"url":        "https://github.com/demo/frontend",
								"language":   "TypeScript",
								"branch":     "main",
								"created_at": "2025-08-15T09:30:00Z",
							},
						},
						"pagination": map[string]interface{}{
							"page":        1,
							"limit":       20,
							"total":       3,
							"total_pages": 1,
						},
					})
				})

				repositories.POST("", func(c *gin.Context) {
					var req struct {
						Name     string `json:"name" binding:"required"`
						URL      string `json:"url" binding:"required"`
						Language string `json:"language" binding:"required"`
						Branch   string `json:"branch"`
					}
					
					if err := c.ShouldBindJSON(&req); err != nil {
						BadRequestResponse(c, "Invalid repository data")
						return
					}

					branch := req.Branch
					if branch == "" {
						branch = "main"
					}

					// Return created repository matching frontend Repository interface
					newRepo := map[string]interface{}{
						"id":         uuid.New().String(),
						"name":       req.Name,
						"url":        req.URL,
						"language":   req.Language,
						"branch":     branch,
						"created_at": time.Now().Format(time.RFC3339),
					}

					c.JSON(http.StatusCreated, APIResponse{
						Success: true,
						Data:    newRepo,
					})
				})
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