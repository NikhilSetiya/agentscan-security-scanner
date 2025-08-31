package commands

import (
	"context"

	"github.com/google/uuid"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/dto"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/services"
)

// CreateScanJobCommand represents a command to create a scan job
type CreateScanJobCommand struct {
	RepositoryID    uuid.UUID
	UserID          *uuid.UUID
	Branch          string
	CommitSHA       string
	ScanType        entities.ScanType
	Priority        entities.Priority
	AgentsRequested []string
}

// StartScanJobCommand represents a command to start a scan job
type StartScanJobCommand struct {
	ScanJobID uuid.UUID
}

// CompleteScanJobCommand represents a command to complete a scan job
type CompleteScanJobCommand struct {
	ScanJobID uuid.UUID
}

// FailScanJobCommand represents a command to fail a scan job
type FailScanJobCommand struct {
	ScanJobID    uuid.UUID
	ErrorMessage string
}

// CancelScanJobCommand represents a command to cancel a scan job
type CancelScanJobCommand struct {
	ScanJobID uuid.UUID
	UserID    uuid.UUID
}

// RetryScanJobCommand represents a command to retry a scan job
type RetryScanJobCommand struct {
	ScanJobID uuid.UUID
	UserID    uuid.UUID
}

// AddCompletedAgentCommand represents a command to add a completed agent
type AddCompletedAgentCommand struct {
	ScanJobID uuid.UUID
	Agent     string
}

// UpdateMetadataCommand represents a command to update scan job metadata
type UpdateMetadataCommand struct {
	ScanJobID uuid.UUID
	Metadata  map[string]interface{}
}

// ScanJobCommandHandler handles scan job-related commands
type ScanJobCommandHandler struct {
	scanService *services.ScanService
}

// NewScanJobCommandHandler creates a new scan job command handler
func NewScanJobCommandHandler(scanService *services.ScanService) *ScanJobCommandHandler {
	return &ScanJobCommandHandler{
		scanService: scanService,
	}
}

// CreateScanJob handles the create scan job command
func (h *ScanJobCommandHandler) CreateScanJob(ctx context.Context, cmd CreateScanJobCommand) (*dto.ScanJobResponse, error) {
	scanJob, err := h.scanService.CreateScanJob(
		ctx,
		cmd.RepositoryID,
		cmd.UserID,
		cmd.Branch,
		cmd.CommitSHA,
		cmd.ScanType,
		cmd.Priority,
		cmd.AgentsRequested,
	)
	if err != nil {
		return nil, err
	}
	
	response := dto.ToScanJobResponse(scanJob)
	return &response, nil
}

// StartScanJob handles the start scan job command
func (h *ScanJobCommandHandler) StartScanJob(ctx context.Context, cmd StartScanJobCommand) error {
	return h.scanService.StartScanJob(ctx, cmd.ScanJobID)
}

// CompleteScanJob handles the complete scan job command
func (h *ScanJobCommandHandler) CompleteScanJob(ctx context.Context, cmd CompleteScanJobCommand) error {
	return h.scanService.CompleteScanJob(ctx, cmd.ScanJobID)
}

// FailScanJob handles the fail scan job command
func (h *ScanJobCommandHandler) FailScanJob(ctx context.Context, cmd FailScanJobCommand) error {
	return h.scanService.FailScanJob(ctx, cmd.ScanJobID, cmd.ErrorMessage)
}

// CancelScanJob handles the cancel scan job command
func (h *ScanJobCommandHandler) CancelScanJob(ctx context.Context, cmd CancelScanJobCommand) error {
	return h.scanService.CancelScanJob(ctx, cmd.ScanJobID, cmd.UserID)
}

// RetryScanJob handles the retry scan job command
func (h *ScanJobCommandHandler) RetryScanJob(ctx context.Context, cmd RetryScanJobCommand) (*dto.ScanJobResponse, error) {
	scanJob, err := h.scanService.RetryScanJob(ctx, cmd.ScanJobID, cmd.UserID)
	if err != nil {
		return nil, err
	}
	
	response := dto.ToScanJobResponse(scanJob)
	return &response, nil
}

// AddCompletedAgent handles the add completed agent command
func (h *ScanJobCommandHandler) AddCompletedAgent(ctx context.Context, cmd AddCompletedAgentCommand) error {
	return h.scanService.AddCompletedAgent(ctx, cmd.ScanJobID, cmd.Agent)
}

// UpdateMetadata handles the update metadata command
func (h *ScanJobCommandHandler) UpdateMetadata(ctx context.Context, cmd UpdateMetadataCommand) error {
	return h.scanService.UpdateScanJobMetadata(ctx, cmd.ScanJobID, cmd.Metadata)
}