package queries

import (
	"context"

	"github.com/google/uuid"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/application/dto"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/repositories"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/services"
)

// GetScanJobByIDQuery represents a query to get scan job by ID
type GetScanJobByIDQuery struct {
	ScanJobID uuid.UUID
}

// GetScanJobWithDetailsQuery represents a query to get scan job with details
type GetScanJobWithDetailsQuery struct {
	ScanJobID uuid.UUID
}

// ListScanJobsQuery represents a query to list scan jobs
type ListScanJobsQuery struct {
	Filter     repositories.Filter
	Pagination repositories.Pagination
}

// ListScanJobsByRepositoryQuery represents a query to list scan jobs by repository
type ListScanJobsByRepositoryQuery struct {
	RepositoryID uuid.UUID
	Filter       repositories.Filter
	Pagination   repositories.Pagination
}

// ListScanJobsByUserQuery represents a query to list scan jobs by user
type ListScanJobsByUserQuery struct {
	UserID     uuid.UUID
	Filter     repositories.Filter
	Pagination repositories.Pagination
}

// GetQueuedJobsQuery represents a query to get queued jobs
type GetQueuedJobsQuery struct {
	Limit int
}

// GetRunningJobsQuery represents a query to get running jobs
type GetRunningJobsQuery struct{}

// ScanJobQueryHandler handles scan job-related queries
type ScanJobQueryHandler struct {
	scanService *services.ScanService
}

// NewScanJobQueryHandler creates a new scan job query handler
func NewScanJobQueryHandler(scanService *services.ScanService) *ScanJobQueryHandler {
	return &ScanJobQueryHandler{
		scanService: scanService,
	}
}

// GetScanJobByID handles the get scan job by ID query
func (h *ScanJobQueryHandler) GetScanJobByID(ctx context.Context, query GetScanJobByIDQuery) (*dto.ScanJobResponse, error) {
	scanJob, err := h.scanService.GetScanJobByID(ctx, query.ScanJobID)
	if err != nil {
		return nil, err
	}
	
	response := dto.ToScanJobResponse(scanJob)
	return &response, nil
}

// GetScanJobWithDetails handles the get scan job with details query
func (h *ScanJobQueryHandler) GetScanJobWithDetails(ctx context.Context, query GetScanJobWithDetailsQuery) (*dto.ScanJobWithDetailsResponse, error) {
	scanJobDetails, err := h.scanService.GetScanJobWithDetails(ctx, query.ScanJobID)
	if err != nil {
		return nil, err
	}
	
	response := dto.ToScanJobWithDetailsResponse(scanJobDetails)
	return &response, nil
}

// ListScanJobs handles the list scan jobs query
func (h *ScanJobQueryHandler) ListScanJobs(ctx context.Context, query ListScanJobsQuery) (*dto.ScanJobListResponse, error) {
	scanJobs, total, err := h.scanService.ListScanJobs(ctx, query.Filter, query.Pagination)
	if err != nil {
		return nil, err
	}
	
	pagination := dto.CreatePagination(query.Pagination.Page, query.Pagination.PageSize, total)
	response := dto.ToScanJobListResponse(scanJobs, pagination)
	return &response, nil
}

// ListScanJobsByRepository handles the list scan jobs by repository query
func (h *ScanJobQueryHandler) ListScanJobsByRepository(ctx context.Context, query ListScanJobsByRepositoryQuery) (*dto.ScanJobListResponse, error) {
	scanJobs, total, err := h.scanService.ListScanJobsByRepository(ctx, query.RepositoryID, query.Filter, query.Pagination)
	if err != nil {
		return nil, err
	}
	
	pagination := dto.CreatePagination(query.Pagination.Page, query.Pagination.PageSize, total)
	response := dto.ToScanJobListResponse(scanJobs, pagination)
	return &response, nil
}

// ListScanJobsByUser handles the list scan jobs by user query
func (h *ScanJobQueryHandler) ListScanJobsByUser(ctx context.Context, query ListScanJobsByUserQuery) (*dto.ScanJobListResponse, error) {
	scanJobs, total, err := h.scanService.ListScanJobsByUser(ctx, query.UserID, query.Filter, query.Pagination)
	if err != nil {
		return nil, err
	}
	
	pagination := dto.CreatePagination(query.Pagination.Page, query.Pagination.PageSize, total)
	response := dto.ToScanJobListResponse(scanJobs, pagination)
	return &response, nil
}

// GetQueuedJobs handles the get queued jobs query
func (h *ScanJobQueryHandler) GetQueuedJobs(ctx context.Context, query GetQueuedJobsQuery) (*dto.ScanJobListResponse, error) {
	scanJobs, err := h.scanService.GetQueuedJobs(ctx, query.Limit)
	if err != nil {
		return nil, err
	}
	
	pagination := dto.CreatePagination(1, len(scanJobs), int64(len(scanJobs)))
	response := dto.ToScanJobListResponse(scanJobs, pagination)
	return &response, nil
}

// GetRunningJobs handles the get running jobs query
func (h *ScanJobQueryHandler) GetRunningJobs(ctx context.Context, query GetRunningJobsQuery) (*dto.ScanJobListResponse, error) {
	scanJobs, err := h.scanService.GetRunningJobs(ctx)
	if err != nil {
		return nil, err
	}
	
	pagination := dto.CreatePagination(1, len(scanJobs), int64(len(scanJobs)))
	response := dto.ToScanJobListResponse(scanJobs, pagination)
	return &response, nil
}