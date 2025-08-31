# Implementation Plan

## Overview

This implementation plan systematically addresses all critical issues identified in the senior software engineer codebase review. The tasks are organized to fix security vulnerabilities first, then architectural issues, performance problems, and finally deployment consolidation. Each task builds incrementally on previous work to ensure a stable transformation to production-ready code.

## Tasks

- [x] 1. Security Vulnerability Remediation (Critical Priority)

  - Create comprehensive input validation and sanitization framework
  - Implement parameterized queries to prevent SQL injection
  - Add security middleware for headers and rate limiting
  - _Requirements: 13.1, 13.2, 13.3, 13.4, 13.6_

- [x] 1.1 Implement Input Validation Framework

  - Create `internal/adapters/http/validators/input.go` with comprehensive validation
  - Add sanitization for all user inputs using bluemonday
  - Implement validation middleware for all API endpoints
  - Add validation tags to all request DTOs
  - _Requirements: 13.2_

- [x] 1.2 Fix SQL Injection Vulnerabilities

  - Replace string concatenation in `internal/database/repositories.go` with parameterized queries
  - Update all WHERE clause building to use proper parameter binding
  - Add SQL query validation to prevent injection attempts
  - Create secure query builder utilities
  - _Requirements: 13.1_

- [x] 1.3 Implement Security Middleware

  - Create `internal/adapters/http/middleware/security.go` with security headers
  - Add rate limiting middleware using Redis
  - Implement CORS middleware with proper origin validation
  - Add request logging middleware with security context
  - _Requirements: 13.6, 13.4_

- [x] 1.4 Add Authentication Consistency

  - Create unified authentication middleware in `internal/adapters/http/middleware/auth.go`
  - Ensure all protected endpoints use consistent authentication
  - Implement proper JWT token validation with Supabase
  - Add role-based access control (RBAC) framework
  - _Requirements: 13.5_

- [x] 2. Error Handling Standardization

  - Create unified error handling system across all layers
  - Implement standardized API response format
  - Add proper error logging with correlation IDs
  - _Requirements: 16.1, 16.2, 16.3, 16.4_

- [x] 2.1 Create Unified Error System

  - Update `pkg/errors/errors.go` to use `map[string]interface{}` for details
  - Create `internal/domain/errors/domain.go` with enhanced error types
  - Add correlation ID and request context to all errors
  - Implement error wrapping and unwrapping utilities
  - _Requirements: 16.1, 16.5_

- [x] 2.2 Standardize API Response Format

  - Update `internal/api/responses.go` to use consistent response wrapper
  - Ensure all API endpoints return standardized format
  - Add proper error code mapping from domain errors to HTTP status
  - Implement response metadata with pagination and timestamps
  - _Requirements: 16.2, 4.1, 4.2_

- [x] 2.3 Implement Error Logging and Monitoring

  - Create `internal/shared/logging/logger.go` with structured logging
  - Add correlation IDs to all requests using middleware
  - Implement error tracking and alerting
  - Add request/response logging with proper sanitization
  - _Requirements: 16.4, 16.7_

- [x] 2.4 Frontend Error Handling Integration

  - Update `web/frontend/src/utils/errorHandler.ts` to handle new error format
  - Ensure consistent error display across all components
  - Add proper error boundaries and fallback UI
  - Implement retry mechanisms for transient errors
  - _Requirements: 16.2, 6.5_

- [x] 3. Code Duplication Elimination and API Refactoring

  - Extract common patterns into reusable middleware and utilities
  - Refactor API handlers to eliminate 40%+ code duplication
  - Create standardized pagination and parameter parsing utilities
  - _Requirements: 12.1, 12.2, 12.3, 12.4, 12.7_

- [x] 3.1 Create Common Middleware

  - Extract user authentication pattern into `internal/adapters/http/middleware/auth.go`
  - Create pagination middleware in `internal/adapters/http/middleware/pagination.go`
  - Add UUID parameter validation middleware
  - Implement request context middleware for correlation IDs
  - _Requirements: 12.2, 12.4_

- [x] 3.2 Create Utility Functions

  - Extract pagination parsing into `internal/shared/utils/pagination.go`
  - Create UUID parameter parsing utilities
  - Add common response helper functions
  - Implement request validation utilities
  - _Requirements: 12.3, 12.5_

- [x] 3.3 Refactor API Handlers

  - Update `internal/api/scans.go` to use common middleware and utilities
  - Refactor `internal/api/repositories.go` to eliminate duplicate patterns
  - Update `internal/api/dashboard.go` to use standardized patterns
  - Remove duplicate code from all API handlers
  - _Requirements: 12.1, 12.6, 12.7_

- [x] 3.4 Create Base Handler Pattern

  - Create `internal/adapters/http/handlers/base.go` with common functionality
  - Implement standardized error handling in base handler
  - Add common validation and response methods
  - Update all handlers to extend base handler
  - _Requirements: 12.5, 12.6_

- [ ] 4. Database Performance Optimization

  - Add critical database indexes for performance
  - Resolve N+1 query problems with JOIN queries
  - Implement efficient pagination strategies
  - _Requirements: 14.1, 14.2, 14.3, 14.4, 14.6_

- [x] 4.1 Add Database Indexes

  - Create migration `migrations/011_add_performance_indexes.up.sql`
  - Add indexes on `scan_jobs(repository_id, status)` for frequent queries
  - Add indexes on `findings(scan_job_id, severity)` for scan results
  - Add indexes on `repositories(organization_id, is_active)` for listing
  - Add composite indexes on `scan_jobs(user_id, created_at DESC)` for user scans
  - _Requirements: 14.1_

- [x] 4.2 Resolve N+1 Query Problems

  - Create `internal/infrastructure/database/optimized_queries.go`
  - Replace N+1 queries in `ListScans` with single JOIN query
  - Optimize dashboard statistics with aggregate queries
  - Implement eager loading for scan job details
  - _Requirements: 14.2, 14.6_

- [x] 4.3 Optimize Pagination Implementation

  - Create efficient pagination using database LIMIT/OFFSET
  - Implement cursor-based pagination for large datasets
  - Add pagination metadata to all list responses
  - Optimize count queries for pagination
  - _Requirements: 14.3_

- [x] 4.4 Create Database Views for Complex Queries

  - Create `dashboard_stats` view for dashboard statistics
  - Create `scan_jobs_with_details` view for scan listing
  - Add materialized views for heavy analytical queries
  - Implement view refresh strategies
  - _Requirements: 14.5_

- [ ] 5. Repository Pattern Standardization

  - Implement consistent repository interfaces across all entities
  - Create generic base repository with common CRUD operations
  - Standardize error handling in repository layer
  - _Requirements: 17.1, 17.2, 17.4_

- [x] 5.1 Create Generic Repository Interface

  - Create `internal/domain/repositories/base.go` with generic repository interface
  - Define standard CRUD operations with proper error handling
  - Add filtering and pagination support to base interface
  - Implement repository interface for each entity type
  - _Requirements: 17.2, 17.4_

- [x] 5.2 Standardize Repository Implementations

  - Update `internal/database/repositories.go` to implement standardized interfaces
  - Ensure consistent error handling across all repositories
  - Add proper transaction support to repository operations
  - Implement repository health checks and monitoring
  - _Requirements: 17.1, 17.4_

- [x] 5.3 Create Repository Factory Pattern

  - Create `internal/infrastructure/database/factory.go` for repository creation
  - Implement dependency injection for repository interfaces
  - Add repository configuration and connection management
  - Create repository testing utilities and mocks
  - _Requirements: 17.3_

- [x] 6. Clean Architecture Implementation

  - Separate business logic from infrastructure concerns
  - Implement proper layer boundaries and dependency inversion
  - Create application services and domain services
  - _Requirements: 17.1, 17.3, 17.5, 17.6_

- [x] 6.1 Create Domain Layer

  - Create `internal/domain/entities/` with core business entities
  - Implement `internal/domain/services/` for business logic
  - Add `internal/domain/repositories/` for repository interfaces
  - Create domain-specific error types and validation
  - _Requirements: 17.1, 17.6_

- [x] 6.2 Implement Application Layer

  - Create `internal/application/commands/` for command handlers (CQRS)
  - Implement `internal/application/queries/` for query handlers
  - Add `internal/application/dto/` for data transfer objects
  - Create application services for orchestrating domain operations
  - _Requirements: 17.3, 17.5_

- [x] 6.3 Refactor Infrastructure Layer

  - Move database implementations to `internal/infrastructure/database/`
  - Create `internal/infrastructure/external/` for external service clients
  - Add `internal/infrastructure/cache/` for caching implementations
  - Implement proper dependency injection and configuration
  - _Requirements: 17.1, 17.3_

- [x] 6.4 Create Adapter Layer

  - Move HTTP handlers to `internal/adapters/http/handlers/`
  - Create thin adapter layer that delegates to application services
  - Implement proper request/response mapping
  - Add adapter-specific validation and error handling
  - _Requirements: 17.4, 17.5_

- [x] 7. Frontend Architecture Improvements

  - Standardize API client patterns and error handling
  - Implement consistent state management
  - Add proper loading states and error boundaries
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 6.1, 6.2, 6.5_

- [x] 7.1 Standardize API Client Patterns

  - Update `web/frontend/src/services/api.ts` to handle new response format
  - Ensure consistent error handling across all API calls
  - Implement proper retry mechanisms and timeout handling
  - Add request/response interceptors for logging and monitoring
  - _Requirements: 3.1, 3.2_

- [x] 7.2 Improve Hook-based State Management

  - Update `web/frontend/src/hooks/useApi.ts` for consistent patterns
  - Implement proper loading states and error handling
  - Add caching and invalidation strategies
  - Create specialized hooks for different data types
  - _Requirements: 3.3, 3.4_

- [x] 7.3 Enhance Error Handling and UX

  - Update error handler to work with new backend error format
  - Implement proper error boundaries and fallback UI
  - Add user-friendly error messages and recovery options
  - Create loading skeletons and empty states
  - _Requirements: 6.1, 6.2, 6.5_

- [x] 7.4 Add Real-time Updates and Monitoring

  - Implement WebSocket connections for real-time scan updates
  - Add proper connection state management
  - Create observability integration with backend monitoring
  - Add performance monitoring and error tracking
  - _Requirements: 11.1, 11.2, 11.3_

- [ ] 8. Deployment Configuration Consolidation

  - Consolidate multiple deployment configurations into single strategy
  - Create environment-specific deployment scripts
  - Implement proper health checks and monitoring
  - _Requirements: 15.1, 15.2, 15.3, 15.4, 15.6_

- [x] 8.1 Consolidate Docker Configurations

  - Create single `deployment/production/Dockerfile` with multi-stage build
  - Remove conflicting `Dockerfile.api` and `Dockerfile.orchestrator`
  - Implement consistent base images and security practices
  - Add proper health check endpoints and monitoring
  - _Requirements: 15.1, 15.2_

- [x] 8.2 Standardize Deployment Scripts

  - Create `deployment/scripts/deploy-production.sh` as single deployment script
  - Remove conflicting deployment scripts and configurations
  - Add environment validation and prerequisite checks
  - Implement proper rollback and recovery procedures
  - _Requirements: 15.4, 15.7_

- [x] 8.3 Environment Configuration Management

  - Create environment-specific configuration files
  - Implement proper secrets management with Supabase
  - Add configuration validation and error handling
  - Create configuration documentation and examples
  - _Requirements: 15.3, 8.1, 8.2, 8.3_

- [x] 8.4 Health Checks and Monitoring

  - Implement comprehensive health check endpoints
  - Add monitoring and alerting for all services
  - Create deployment verification and smoke tests
  - Add performance monitoring and logging
  - _Requirements: 15.5, 15.6_

- [-] 9. Supabase Authentication Integration

  - Integrate Supabase authentication with backend
  - Implement proper token validation and user management
  - Add secure session handling and token refresh
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 2.7, 2.8_

- [x] 9.1 Backend Supabase Integration

  - Create `internal/infrastructure/auth/supabase.go` for Supabase client
  - Implement JWT token validation with Supabase
  - Add user creation and management through Supabase
  - Create authentication middleware using Supabase tokens
  - _Requirements: 2.1, 2.2, 2.4_

- [x] 9.2 Frontend Supabase Integration

  - Update frontend to use Supabase client for authentication
  - Implement login, signup, and password reset flows
  - Add proper session management and token refresh
  - Create authentication context and protected routes
  - _Requirements: 2.3, 2.5, 2.6, 2.7, 2.8_

- [x] 9.3 User Management and Profiles

  - Create user profile management with Supabase
  - Implement user role and permission management
  - Add user settings and preferences
  - Create user onboarding and verification flows
  - _Requirements: 2.1, 2.7, 2.8_

- [x] 10. Production Environment Configuration

  - Configure production settings for security and performance
  - Implement proper logging and monitoring
  - Add environment-specific optimizations
  - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 7.1, 7.2, 7.3, 7.4, 7.5_

- [x] 10.1 Production Security Configuration

  - Configure HTTPS enforcement and security headers
  - Implement proper CORS configuration for production domains
  - Add rate limiting and DDoS protection
  - Configure secure cookie and session settings
  - _Requirements: 7.1, 7.3, 7.4, 7.5_

- [x] 10.2 Performance and Monitoring Configuration

  - Configure production logging with structured format
  - Add performance monitoring and metrics collection
  - Implement health checks and alerting
  - Configure database connection pooling and optimization
  - _Requirements: 5.1, 5.5, 7.2_

- [x] 10.3 Environment Variable Management

  - Create comprehensive environment variable documentation
  - Implement environment validation and fail-fast behavior
  - Add secrets management with Supabase vault
  - Create environment-specific configuration templates
  - _Requirements: 5.2, 5.3, 8.4_

- [ ] 11. Real Data Integration and Testing

  - Replace dummy data with real database integration
  - Implement comprehensive testing for all components
  - Add integration tests for critical workflows
  - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5_

- [ ] 11.1 Database Integration Testing

  - Create integration tests for all repository implementations
  - Test database performance with realistic data volumes
  - Verify index effectiveness and query optimization
  - Add database migration testing and rollback procedures
  - _Requirements: 11.1, 11.2_

- [ ] 11.2 API Integration Testing

  - Create end-to-end tests for all API endpoints
  - Test authentication and authorization flows
  - Verify error handling and response formats
  - Add performance testing for critical endpoints
  - _Requirements: 11.3, 11.4_

- [ ] 11.3 Frontend Integration Testing

  - Test frontend with real backend API integration
  - Verify data display and user interactions
  - Test error handling and loading states
  - Add accessibility and responsive design testing
  - _Requirements: 11.5, 10.6, 10.7_

- [ ] 12. Final Production Deployment and Verification

  - Deploy all components to production environment
  - Verify all functionality works with real data
  - Perform comprehensive smoke testing
  - _Requirements: All requirements verification_

- [ ] 12.1 Production Deployment

  - Deploy backend to Fly.io with new configuration
  - Deploy frontend to Vercel with updated settings
  - Configure production database and Redis
  - Set up monitoring and alerting systems
  - _Requirements: 1.1, 5.4, 15.6_

- [ ] 12.2 Smoke Testing and Verification

  - Test all critical user workflows end-to-end
  - Verify authentication and authorization
  - Test scan creation and management functionality
  - Validate performance and security measures
  - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 9.7_

- [ ] 12.3 Documentation and Handover
  - Create comprehensive deployment documentation
  - Document all configuration and environment variables
  - Create troubleshooting guides and runbooks
  - Add monitoring and alerting documentation
  - _Requirements: 5.5, 15.5, 15.6_
