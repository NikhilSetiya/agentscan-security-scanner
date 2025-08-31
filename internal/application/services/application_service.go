package services

import (
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/commands"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/queries"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/services"
)

// ApplicationService aggregates all command and query handlers
type ApplicationService struct {
	// Command Handlers
	UserCommands       *commands.UserCommandHandler
	RepositoryCommands *commands.RepositoryCommandHandler
	ScanJobCommands    *commands.ScanJobCommandHandler
	
	// Query Handlers
	UserQueries       *queries.UserQueryHandler
	RepositoryQueries *queries.RepositoryQueryHandler
	ScanJobQueries    *queries.ScanJobQueryHandler
}

// NewApplicationService creates a new application service with all handlers
func NewApplicationService(
	userService *services.UserService,
	repositoryService *services.RepositoryService,
	scanService *services.ScanService,
) *ApplicationService {
	return &ApplicationService{
		// Initialize command handlers
		UserCommands:       commands.NewUserCommandHandler(userService),
		RepositoryCommands: commands.NewRepositoryCommandHandler(repositoryService),
		ScanJobCommands:    commands.NewScanJobCommandHandler(scanService),
		
		// Initialize query handlers
		UserQueries:       queries.NewUserQueryHandler(userService),
		RepositoryQueries: queries.NewRepositoryQueryHandler(repositoryService),
		ScanJobQueries:    queries.NewScanJobQueryHandler(scanService),
	}
}