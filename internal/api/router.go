package api

import (
	"github.com/gin-gonic/gin"

	"github.com/agentscan/agentscan/internal/database"
	"github.com/agentscan/agentscan/internal/orchestrator"
	"github.com/agentscan/agentscan/internal/queue"
	"github.com/agentscan/agentscan/pkg/config"
)

// Router creates and configures the API router
func NewRouter(cfg *config.Config, db *database.DB, redis *queue.RedisClient, repos *database.Repositories, orch orchestrator.OrchestrationService, q *queue.Queue) *gin.Engine {
	// Set Gin mode based on environment
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Add middleware
	router.Use(RequestIDMiddleware())
	router.Use(LoggingMiddleware())
	router.Use(ErrorHandlingMiddleware())
	router.Use(CORSMiddleware())
	router.Use(SecurityHeadersMiddleware())
	router.Use(RateLimitMiddleware())

	// Health check endpoint (no auth required)
	healthHandler := NewHealthHandler(db, redis)
	router.GET("/health", gin.WrapH(healthHandler))

	// API version info (no auth required)
	router.GET("/api/v1", func(c *gin.Context) {
		SuccessResponse(c, map[string]interface{}{
			"name":    "AgentScan API",
			"version": "1.0.0",
			"status":  "ok",
		})
	})

	// Create handlers
	authHandler := NewAuthHandler(cfg, repos)
	scanHandler := NewScanHandler(repos, orch, q)
	findingHandler := NewFindingHandler(repos)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Authentication routes (no auth required)
		auth := v1.Group("/auth")
		{
			auth.GET("/github/url", authHandler.GetAuthURL)
			auth.POST("/github/callback", authHandler.LoginWithGitHub)
			auth.POST("/logout", authHandler.Logout)
		}

		// Protected routes (require authentication)
		protected := v1.Group("")
		protected.Use(AuthMiddleware(cfg))
		{
			// User routes
			user := protected.Group("/user")
			{
				user.GET("/me", authHandler.GetCurrentUserInfo)
				user.POST("/refresh", authHandler.RefreshToken)
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
				findings.GET("", findingHandler.ListFindings)
				findings.GET("/stats", findingHandler.GetFindingStats)
				findings.GET("/export", findingHandler.ExportFindings)
				findings.GET("/:id", findingHandler.GetFinding)
				findings.PATCH("/:id/status", findingHandler.UpdateFindingStatus)
				findings.PATCH("/bulk/status", findingHandler.BulkUpdateFindings)
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
					// TODO: Implement GitHub webhook handler
					SuccessResponse(c, map[string]string{
						"message": "GitHub webhook received",
					})
				})

				// GitLab webhook handler
				webhooks.POST("/gitlab", func(c *gin.Context) {
					// TODO: Implement GitLab webhook handler
					SuccessResponse(c, map[string]string{
						"message": "GitLab webhook received",
					})
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
				agents.POST("/results", func(c *gin.Context) {
					// TODO: Implement agent result submission
					SuccessResponse(c, map[string]string{
						"message": "Results received",
					})
				})
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
	return NewRouter(cfg, db, redis, repos, orch, q)
}