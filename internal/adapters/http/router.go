package http

import (
	"github.com/gin-gonic/gin"
	
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/adapters/http/handlers"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/adapters/http/middleware"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/services"
)

// Router sets up HTTP routes for the application
type Router struct {
	appService *services.ApplicationService
	
	// Handlers
	userHandler       *handlers.UserHandler
	repoHandler       *handlers.RepositoryHandler
	scanJobHandler    *handlers.ScanJobHandler
}

// NewRouter creates a new HTTP router
func NewRouter(appService *services.ApplicationService) *Router {
	return &Router{
		appService:     appService,
		userHandler:    handlers.NewUserHandler(appService),
		repoHandler:    handlers.NewRepositoryHandler(appService),
		scanJobHandler: handlers.NewScanJobHandler(appService),
	}
}

// SetupRoutes configures all HTTP routes
func (r *Router) SetupRoutes() *gin.Engine {
	router := gin.New()
	
	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.RequestID())
	
	// Health check
	router.GET("/health", r.healthCheck)
	
	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Authentication middleware for protected routes
		protected := v1.Group("")
		protected.Use(middleware.AuthRequired())
		
		// User routes
		userRoutes := protected.Group("/users")
		{
			userRoutes.POST("", r.userHandler.CreateUser)
			userRoutes.GET("/:id", r.userHandler.GetUser)
			userRoutes.GET("/email/:email", r.userHandler.GetUserByEmail)
			userRoutes.GET("", r.userHandler.ListUsers)
			userRoutes.PUT("/:id/profile", r.userHandler.UpdateUserProfile)
			userRoutes.PUT("/:id/role", middleware.RequireRole("admin"), r.userHandler.UpdateUserRole)
			userRoutes.POST("/:id/deactivate", middleware.RequireRole("admin"), r.userHandler.DeactivateUser)
			userRoutes.POST("/:id/activate", middleware.RequireRole("admin"), r.userHandler.ActivateUser)
			userRoutes.GET("/:id/scan-jobs", r.scanJobHandler.ListScanJobsByUser)
		}
		
		// Repository routes
		repoRoutes := protected.Group("/repositories")
		{
			repoRoutes.POST("", r.repoHandler.CreateRepository)
			repoRoutes.GET("/:id", r.repoHandler.GetRepository)
			repoRoutes.GET("/by-url", r.repoHandler.GetRepositoryByURL)
			repoRoutes.GET("", r.repoHandler.ListRepositories)
			repoRoutes.GET("/active", r.repoHandler.ListActiveRepositories)
			repoRoutes.PUT("/:id", r.repoHandler.UpdateRepository)
			repoRoutes.PUT("/:id/default-branch", r.repoHandler.SetDefaultBranch)
			repoRoutes.POST("/:id/languages", r.repoHandler.AddLanguage)
			repoRoutes.PUT("/:id/settings", r.repoHandler.UpdateRepositorySettings)
			repoRoutes.GET("/:id/settings", r.repoHandler.GetRepositorySettings)
			repoRoutes.POST("/:id/deactivate", middleware.RequireRole("admin"), r.repoHandler.DeactivateRepository)
			repoRoutes.POST("/:id/activate", middleware.RequireRole("admin"), r.repoHandler.ActivateRepository)
			repoRoutes.GET("/:id/scan-jobs", r.scanJobHandler.ListScanJobsByRepository)
		}
		
		// Scan job routes
		scanJobRoutes := protected.Group("/scan-jobs")
		{
			scanJobRoutes.POST("", r.scanJobHandler.CreateScanJob)
			scanJobRoutes.GET("/:id", r.scanJobHandler.GetScanJob)
			scanJobRoutes.GET("/:id/details", r.scanJobHandler.GetScanJobWithDetails)
			scanJobRoutes.GET("", r.scanJobHandler.ListScanJobs)
			scanJobRoutes.GET("/queued", r.scanJobHandler.GetQueuedJobs)
			scanJobRoutes.GET("/running", r.scanJobHandler.GetRunningJobs)
			scanJobRoutes.POST("/:id/start", r.scanJobHandler.StartScanJob)
			scanJobRoutes.POST("/:id/complete", r.scanJobHandler.CompleteScanJob)
			scanJobRoutes.POST("/:id/fail", r.scanJobHandler.FailScanJob)
			scanJobRoutes.POST("/:id/cancel", r.scanJobHandler.CancelScanJob)
			scanJobRoutes.POST("/:id/retry", r.scanJobHandler.RetryScanJob)
			scanJobRoutes.POST("/:id/completed-agents", r.scanJobHandler.AddCompletedAgent)
			scanJobRoutes.PUT("/:id/metadata", r.scanJobHandler.UpdateScanJobMetadata)
		}
	}
	
	return router
}

// healthCheck handles health check requests
func (r *Router) healthCheck(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "healthy",
		"service": "agentscan-security-scanner",
	})
}"