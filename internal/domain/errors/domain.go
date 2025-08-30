package errors

import (
	"fmt"
	"github.com/google/uuid"
	"github.com/agentscan/agentscan/pkg/errors"
)

// Domain-specific error constructors that provide more context

// Repository Domain Errors

func NewRepositoryNotFoundError(id uuid.UUID) *errors.AppError {
	return errors.NewNotFoundError("repository").
		WithDetails(map[string]interface{}{
			"repository_id": id.String(),
		})
}

func NewRepositoryAlreadyExistsError(name, url string) *errors.AppError {
	return errors.NewConflictError("repository already exists").
		WithDetails(map[string]interface{}{
			"name": name,
			"url":  url,
		})
}

func NewRepositoryAccessDeniedError(id uuid.UUID, userID uuid.UUID) *errors.AppError {
	return errors.NewForbiddenError("access denied to repository").
		WithDetails(map[string]interface{}{
			"repository_id": id.String(),
			"user_id":       userID.String(),
		})
}

func NewInvalidRepositoryURLError(url string) *errors.AppError {
	return errors.NewValidationError("invalid repository URL").
		WithDetails(map[string]interface{}{
			"url": url,
		})
}

// Scan Job Domain Errors

func NewScanJobNotFoundError(id uuid.UUID) *errors.AppError {
	return errors.NewNotFoundError("scan job").
		WithDetails(map[string]interface{}{
			"scan_job_id": id.String(),
		})
}

func NewScanJobAlreadyRunningError(id uuid.UUID) *errors.AppError {
	return errors.NewConflictError("scan job is already running").
		WithDetails(map[string]interface{}{
			"scan_job_id": id.String(),
			"status":      "running",
		})
}

func NewScanJobCancellationError(id uuid.UUID, reason string) *errors.AppError {
	return errors.NewInternalError("failed to cancel scan job").
		WithDetails(map[string]interface{}{
			"scan_job_id": id.String(),
			"reason":      reason,
		})
}

func NewInvalidScanConfigurationError(details map[string]interface{}) *errors.AppError {
	return errors.NewValidationError("invalid scan configuration").
		WithDetails(details)
}

func NewScanTimeoutError(id uuid.UUID, duration string) *errors.AppError {
	return errors.NewTimeoutError("scan job").
		WithDetails(map[string]interface{}{
			"scan_job_id": id.String(),
			"duration":    duration,
		})
}

// Agent Domain Errors

func NewAgentNotAvailableError(agentName string) *errors.AppError {
	return errors.NewServiceUnavailableError(fmt.Sprintf("agent %s", agentName)).
		WithDetails(map[string]interface{}{
			"agent_name": agentName,
		})
}

func NewAgentExecutionError(agentName, phase, message string) *errors.AppError {
	return errors.NewAgentError(agentName, fmt.Sprintf("execution failed in %s phase: %s", phase, message)).
		WithDetails(map[string]interface{}{
			"agent_name": agentName,
			"phase":      phase,
			"error":      message,
		})
}

func NewAgentTimeoutError(agentName string, timeout string) *errors.AppError {
	return errors.NewTimeoutError(fmt.Sprintf("agent %s", agentName)).
		WithDetails(map[string]interface{}{
			"agent_name": agentName,
			"timeout":    timeout,
		})
}

func NewInvalidAgentResponseError(agentName string, response interface{}) *errors.AppError {
	return errors.NewValidationError("invalid agent response format").
		WithDetails(map[string]interface{}{
			"agent_name": agentName,
			"response":   response,
		})
}

// Finding Domain Errors

func NewFindingNotFoundError(id uuid.UUID) *errors.AppError {
	return errors.NewNotFoundError("finding").
		WithDetails(map[string]interface{}{
			"finding_id": id.String(),
		})
}

func NewInvalidFindingStatusError(currentStatus, requestedStatus string) *errors.AppError {
	return errors.NewValidationError("invalid finding status transition").
		WithDetails(map[string]interface{}{
			"current_status":   currentStatus,
			"requested_status": requestedStatus,
		})
}

func NewFindingSuppressionError(id uuid.UUID, reason string) *errors.AppError {
	return errors.NewValidationError("cannot suppress finding").
		WithDetails(map[string]interface{}{
			"finding_id": id.String(),
			"reason":     reason,
		})
}

// User Domain Errors

func NewUserNotFoundError(identifier string) *errors.AppError {
	return errors.NewNotFoundError("user").
		WithDetails(map[string]interface{}{
			"identifier": identifier,
		})
}

func NewUserAlreadyExistsError(email string) *errors.AppError {
	return errors.NewConflictError("user already exists").
		WithDetails(map[string]interface{}{
			"email": email,
		})
}

func NewInvalidCredentialsError() *errors.AppError {
	return errors.NewUnauthorizedError("invalid credentials")
}

func NewAccountDisabledError(userID uuid.UUID) *errors.AppError {
	return errors.NewForbiddenError("account is disabled").
		WithDetails(map[string]interface{}{
			"user_id": userID.String(),
		})
}

func NewInsufficientPermissionsError(userID uuid.UUID, requiredPermission string) *errors.AppError {
	return errors.NewForbiddenError("insufficient permissions").
		WithDetails(map[string]interface{}{
			"user_id":             userID.String(),
			"required_permission": requiredPermission,
		})
}

// Organization Domain Errors

func NewOrganizationNotFoundError(id uuid.UUID) *errors.AppError {
	return errors.NewNotFoundError("organization").
		WithDetails(map[string]interface{}{
			"organization_id": id.String(),
		})
}

func NewOrganizationAccessDeniedError(orgID, userID uuid.UUID) *errors.AppError {
	return errors.NewForbiddenError("access denied to organization").
		WithDetails(map[string]interface{}{
			"organization_id": orgID.String(),
			"user_id":         userID.String(),
		})
}

// Integration Domain Errors

func NewGitHubIntegrationError(operation, message string) *errors.AppError {
	return errors.NewExternalError("GitHub", fmt.Sprintf("%s failed: %s", operation, message)).
		WithDetails(map[string]interface{}{
			"operation": operation,
			"provider":  "github",
		})
}

func NewGitLabIntegrationError(operation, message string) *errors.AppError {
	return errors.NewExternalError("GitLab", fmt.Sprintf("%s failed: %s", operation, message)).
		WithDetails(map[string]interface{}{
			"operation": operation,
			"provider":  "gitlab",
		})
}

func NewSupabaseIntegrationError(operation, message string) *errors.AppError {
	return errors.NewExternalError("Supabase", fmt.Sprintf("%s failed: %s", operation, message)).
		WithDetails(map[string]interface{}{
			"operation": operation,
			"provider":  "supabase",
		})
}

// Orchestration Domain Errors

func NewOrchestrationError(phase, message string) *errors.AppError {
	return errors.NewInternalError(fmt.Sprintf("orchestration failed in %s: %s", phase, message)).
		WithDetails(map[string]interface{}{
			"phase": phase,
		})
}

func NewConsensusFailedError(agentCount int, agreementThreshold float64) *errors.AppError {
	return errors.NewConsensusError("agents failed to reach consensus").
		WithDetails(map[string]interface{}{
			"agent_count":          agentCount,
			"agreement_threshold":  agreementThreshold,
		})
}

func NewResourceExhaustionError(resource, limit string) *errors.AppError {
	return errors.NewServiceUnavailableError("system").
		WithDetails(map[string]interface{}{
			"resource": resource,
			"limit":    limit,
		})
}

// Queue Domain Errors

func NewQueueFullError(queueName string, capacity int) *errors.AppError {
	return errors.NewServiceUnavailableError("queue").
		WithDetails(map[string]interface{}{
			"queue_name": queueName,
			"capacity":   capacity,
		})
}

func NewJobProcessingError(jobID, message string) *errors.AppError {
	return errors.NewInternalError("job processing failed").
		WithDetails(map[string]interface{}{
			"job_id": jobID,
			"error":  message,
		})
}

// Configuration Domain Errors

func NewConfigurationError(key, message string) *errors.AppError {
	return errors.NewInternalError(fmt.Sprintf("configuration error for %s: %s", key, message)).
		WithDetails(map[string]interface{}{
			"config_key": key,
		})
}

func NewMissingConfigurationError(key string) *errors.AppError {
	return errors.NewInternalError(fmt.Sprintf("missing required configuration: %s", key)).
		WithDetails(map[string]interface{}{
			"config_key": key,
		})
}

// Validation Domain Errors

func NewFieldValidationError(field, rule, value string) *errors.AppError {
	return errors.NewValidationError(fmt.Sprintf("field %s failed validation rule %s", field, rule)).
		WithDetails(map[string]interface{}{
			"field": field,
			"rule":  rule,
			"value": value,
		})
}

func NewBusinessRuleViolationError(rule, message string) *errors.AppError {
	return errors.NewValidationError(fmt.Sprintf("business rule violation: %s", message)).
		WithDetails(map[string]interface{}{
			"rule": rule,
		})
}

// Rate Limiting Domain Errors

func NewRateLimitExceededError(resource string, limit int, window string) *errors.AppError {
	return errors.NewRateLimitError(fmt.Sprintf("rate limit exceeded for %s", resource)).
		WithDetails(map[string]interface{}{
			"resource": resource,
			"limit":    limit,
			"window":   window,
		})
}