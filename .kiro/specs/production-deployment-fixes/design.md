# Design Document

## Overview

This design document outlines the comprehensive architectural improvements and fixes needed to transform the AgentScan Security Scanner from its current state with critical issues into a production-ready, enterprise-grade security scanning platform. The design addresses code duplication, security vulnerabilities, performance bottlenecks, inconsistent error handling, deployment configuration issues, and architectural concerns identified in the senior software engineer review.

## Architecture

### Current Architecture Issues

The current architecture suffers from several critical problems:
- Mixed concerns across layers (business logic in API handlers)
- Inconsistent repository patterns and interfaces
- No clear separation between domain and infrastructure
- Duplicate code patterns throughout the API layer
- Inconsistent error handling strategies

### Target Clean Architecture

```
cmd/
├── api/                    # Application entry points
├── orchestrator/          # Orchestrator service entry
internal/
├── domain/               # Business Logic Layer (NEW)
│   ├── entities/         # Core business entities
│   ├── services/         # Business services
│   ├── repositories/     # Repository interfaces
│   └── errors/          # Domain-specific errors
├── application/         # Application Layer (NEW)
│   ├── commands/        # Command handlers (CQRS)
│   ├── queries/         # Query handlers (CQRS)
│   ├── dto/            # Data transfer objects
│   └── services/       # Application services
├── infrastructure/     # Infrastructure Layer (NEW)
│   ├── database/       # Database implementations
│   ├── external/       # External service clients
│   ├── cache/         # Caching implementations
│   └── queue/         # Queue implementations
├── adapters/          # Interface Adapters (NEW)
│   ├── http/          # HTTP handlers (thin layer)
│   ├── middleware/    # Common middleware
│   └── validators/    # Input validation
└── shared/           # Shared utilities (NEW)
    ├── config/       # Configuration management
    ├── logging/      # Structured logging
    └── monitoring/   # Observability
```

### Dependency Flow

```
HTTP Handlers → Application Services → Domain Services → Repository Interfaces
                     ↓                      ↓                    ↓
                Application DTOs    Domain Entities    Infrastructure Implementations
```

## Components and Interfaces

### 1. Standardized Repository Pattern

```go
// internal/domain/repositories/base.go
type BaseRepository[T any, ID comparable] interface {
    Create(ctx context.Context, entity T) error
    GetByID(ctx context.Context, id ID) (T, error)
    Update(ctx context.Context, entity T) error
    Delete(ctx context.Context, id ID) error
    List(ctx context.Context, filter Filter, pagination Pagination) ([]T, int64, error)
    Exists(ctx context.Context, id ID) (bool, error)
}

// internal/domain/repositories/user.go
type UserRepository interface {
    BaseRepository[*entities.User, uuid.UUID]
    GetByEmail(ctx context.Context, email string) (*entities.User, error)
    GetBySupabaseID(ctx context.Context, supabaseID string) (*entities.User, error)
}

// internal/domain/repositories/scan_job.go
type ScanJobRepository interface {
    BaseRepository[*entities.ScanJob, uuid.UUID]
    ListByRepository(ctx context.Context, repoID uuid.UUID, filter Filter, pagination Pagination) ([]*entities.ScanJob, int64, error)
    GetWithDetails(ctx context.Context, id uuid.UUID) (*entities.ScanJobWithDetails, error)
    UpdateStatus(ctx context.Context, id uuid.UUID, status string) error
}
```

### 2. Unified Error Handling System

```go
// pkg/errors/domain.go
type DomainError struct {
    Type       ErrorType              `json:"type"`
    Code       string                 `json:"code"`
    Message    string                 `json:"message"`
    Details    map[string]interface{} `json:"details,omitempty"`
    Cause      error                  `json:"-"`
    Timestamp  time.Time              `json:"timestamp"`
    RequestID  string                 `json:"request_id,omitempty"`
    UserID     *uuid.UUID             `json:"user_id,omitempty"`
}

// internal/adapters/http/middleware/error.go
func ErrorHandlerMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        c.Next()
        if len(c.Errors) > 0 {
            err := c.Errors.Last().Err
            handleError(c, err)
        }
    }
}

// internal/adapters/http/responses/standardized.go
type APIResponse struct {
    Success   bool                   `json:"success"`
    Data      interface{}            `json:"data,omitempty"`
    Error     *APIError             `json:"error,omitempty"`
    Meta      *ResponseMeta         `json:"meta,omitempty"`
    RequestID string                `json:"request_id"`
    Timestamp time.Time             `json:"timestamp"`
}
```

### 3. Common Middleware and Utilities

```go
// internal/adapters/http/middleware/auth.go
func RequireAuth() gin.HandlerFunc {
    return func(c *gin.Context) {
        userID, exists := extractUserID(c)
        if !exists {
            c.Error(errors.NewAuthenticationError("authentication required"))
            c.Abort()
            return
        }
        c.Set("authenticated_user_id", userID)
        c.Next()
    }
}

// internal/adapters/http/middleware/pagination.go
func PaginationMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        pagination := parsePagination(c)
        c.Set("pagination", pagination)
        c.Next()
    }
}

// internal/adapters/http/middleware/validation.go
func ValidateUUID(param string) gin.HandlerFunc {
    return func(c *gin.Context) {
        id, err := parseUUIDParam(c, param)
        if err != nil {
            c.Error(errors.NewValidationError("invalid UUID format"))
            c.Abort()
            return
        }
        c.Set(param+"_uuid", id)
        c.Next()
    }
}
```

### 4. Database Optimization Layer

```go
// internal/infrastructure/database/optimized_queries.go
type OptimizedQueries struct {
    db *sqlx.DB
}

// Solve N+1 query problem with JOIN queries
func (q *OptimizedQueries) GetScansWithDetails(ctx context.Context, filter ScanFilter, pagination Pagination) ([]*ScanWithDetails, int64, error) {
    query := `
        SELECT 
            sj.*,
            r.name as repository_name,
            r.url as repository_url,
            r.language as repository_language,
            u.email as triggered_by_email,
            COUNT(f.id) as findings_count,
            EXTRACT(EPOCH FROM (COALESCE(sj.completed_at, NOW()) - sj.started_at)) as duration_seconds
        FROM scan_jobs sj
        JOIN repositories r ON sj.repository_id = r.id
        LEFT JOIN users u ON sj.user_id = u.id
        LEFT JOIN findings f ON sj.id = f.scan_job_id
        WHERE ($1::uuid IS NULL OR sj.repository_id = $1)
          AND ($2::text IS NULL OR sj.status = $2)
          AND ($3::uuid IS NULL OR sj.user_id = $3)
        GROUP BY sj.id, r.id, u.id
        ORDER BY sj.created_at DESC
        LIMIT $4 OFFSET $5
    `
    // Single query instead of N+1 queries
}

// Database indexes for performance
func (q *OptimizedQueries) CreateIndexes(ctx context.Context) error {
    indexes := []string{
        "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scan_jobs_repository_status ON scan_jobs(repository_id, status)",
        "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_findings_scan_job_severity ON findings(scan_job_id, severity)",
        "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_repositories_org_active ON repositories(organization_id, is_active)",
        "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_scan_jobs_user_created ON scan_jobs(user_id, created_at DESC)",
    }
    // Execute index creation
}
```

### 5. Security Enhancement Layer

```go
// internal/adapters/http/middleware/security.go
func SecurityMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Security headers
        c.Header("X-Content-Type-Options", "nosniff")
        c.Header("X-Frame-Options", "DENY")
        c.Header("X-XSS-Protection", "1; mode=block")
        c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
        c.Header("Content-Security-Policy", "default-src 'self'")
        c.Next()
    }
}

func RateLimitMiddleware(requests int, window time.Duration) gin.HandlerFunc {
    // Implement rate limiting using Redis
}

// internal/adapters/http/validators/input.go
type InputValidator struct {
    validator *validator.Validate
}

func (v *InputValidator) ValidateAndSanitize(input interface{}) error {
    // Comprehensive input validation and sanitization
    if err := v.validator.Struct(input); err != nil {
        return errors.NewValidationError("input validation failed").WithDetails(extractValidationErrors(err))
    }
    return nil
}
```

## Data Models

### Enhanced Entity Definitions

```go
// internal/domain/entities/user.go
type User struct {
    ID         uuid.UUID  `json:"id"`
    Email      string     `json:"email" validate:"required,email"`
    Name       string     `json:"name" validate:"required,min=1,max=255"`
    AvatarURL  string     `json:"avatar_url" validate:"omitempty,url"`
    SupabaseID *string    `json:"supabase_id,omitempty"`
    Role       UserRole   `json:"role" validate:"required"`
    CreatedAt  time.Time  `json:"created_at"`
    UpdatedAt  time.Time  `json:"updated_at"`
}

// internal/domain/entities/scan_job.go
type ScanJobWithDetails struct {
    ScanJob
    Repository    *Repository `json:"repository"`
    User          *User       `json:"user,omitempty"`
    FindingsCount int         `json:"findings_count"`
    Duration      *time.Duration `json:"duration,omitempty"`
}

// internal/domain/value_objects/pagination.go
type Pagination struct {
    Page       int   `json:"page" validate:"min=1"`
    PageSize   int   `json:"page_size" validate:"min=1,max=100"`
    Total      int64 `json:"total"`
    TotalPages int   `json:"total_pages"`
    HasNext    bool  `json:"has_next"`
    HasPrev    bool  `json:"has_prev"`
}
```

### Database Schema Optimizations

```sql
-- migrations/011_add_performance_indexes.up.sql
-- Critical indexes for performance
CREATE INDEX CONCURRENTLY idx_scan_jobs_repository_status ON scan_jobs(repository_id, status);
CREATE INDEX CONCURRENTLY idx_scan_jobs_user_created ON scan_jobs(user_id, created_at DESC);
CREATE INDEX CONCURRENTLY idx_findings_scan_job_severity ON findings(scan_job_id, severity);
CREATE INDEX CONCURRENTLY idx_repositories_org_active ON repositories(organization_id, is_active);

-- Optimized view for dashboard statistics
CREATE VIEW dashboard_stats AS
SELECT 
    COUNT(DISTINCT sj.id) as total_scans,
    COUNT(DISTINCT r.id) as total_repositories,
    COUNT(CASE WHEN f.severity = 'critical' THEN 1 END) as critical_findings,
    COUNT(CASE WHEN f.severity = 'high' THEN 1 END) as high_findings,
    COUNT(CASE WHEN f.severity = 'medium' THEN 1 END) as medium_findings,
    COUNT(CASE WHEN f.severity = 'low' THEN 1 END) as low_findings,
    r.organization_id
FROM repositories r
LEFT JOIN scan_jobs sj ON r.id = sj.repository_id
LEFT JOIN findings f ON sj.id = f.scan_job_id
WHERE r.is_active = true
GROUP BY r.organization_id;
```

## Error Handling

### Comprehensive Error Strategy

```go
// pkg/errors/types.go
type ErrorType string

const (
    ErrorTypeValidation     ErrorType = "validation"
    ErrorTypeAuthentication ErrorType = "authentication"
    ErrorTypeAuthorization  ErrorType = "authorization"
    ErrorTypeNotFound       ErrorType = "not_found"
    ErrorTypeConflict       ErrorType = "conflict"
    ErrorTypeRateLimit      ErrorType = "rate_limit"
    ErrorTypeInternal       ErrorType = "internal"
    ErrorTypeExternal       ErrorType = "external"
    ErrorTypeTimeout        ErrorType = "timeout"
    ErrorTypeSecurity       ErrorType = "security"
)

// internal/adapters/http/handlers/base.go
type BaseHandler struct {
    logger    *zap.Logger
    validator *InputValidator
}

func (h *BaseHandler) HandleError(c *gin.Context, err error) {
    requestID := c.GetString("request_id")
    
    // Log error with context
    h.logger.Error("request error",
        zap.String("request_id", requestID),
        zap.String("method", c.Request.Method),
        zap.String("path", c.Request.URL.Path),
        zap.Error(err),
    )
    
    // Convert to standardized error response
    c.Error(err)
}
```

### Frontend Error Handling Integration

```typescript
// web/frontend/src/utils/errorHandler.ts
interface StandardizedError {
    type: string;
    code: string;
    message: string;
    details?: Record<string, any>;
    request_id?: string;
    timestamp: string;
}

class ErrorHandler {
    handleApiError(error: StandardizedError, context?: string): EnhancedError {
        // Map backend error types to frontend error handling
        const severity = this.mapErrorTypeToSeverity(error.type);
        const userMessage = this.generateUserFriendlyMessage(error);
        
        return {
            code: error.code,
            message: error.message,
            userMessage,
            severity,
            details: error.details,
            retryable: this.isRetryable(error.type),
            actionable: this.isActionable(error.type),
        };
    }
}
```

## Testing Strategy

### Comprehensive Testing Approach

```go
// tests/integration/api_test.go
func TestAPIEndpoints(t *testing.T) {
    // Test all API endpoints with standardized patterns
    testCases := []struct {
        name           string
        method         string
        path           string
        body           interface{}
        expectedStatus int
        expectedError  string
    }{
        {
            name:           "create repository with valid data",
            method:         "POST",
            path:           "/api/v1/repositories",
            body:           validRepositoryRequest,
            expectedStatus: 201,
        },
        {
            name:           "create repository with invalid data",
            method:         "POST",
            path:           "/api/v1/repositories",
            body:           invalidRepositoryRequest,
            expectedStatus: 400,
            expectedError:  "VALIDATION_ERROR",
        },
    }
}

// tests/unit/repositories_test.go
func TestRepositoryPattern(t *testing.T) {
    // Test repository implementations
    repo := setupTestRepository(t)
    
    t.Run("CRUD operations", func(t *testing.T) {
        // Test Create, Read, Update, Delete operations
    })
    
    t.Run("error handling", func(t *testing.T) {
        // Test error scenarios
    })
}
```

### Performance Testing

```go
// tests/performance/load_test.go
func TestDatabasePerformance(t *testing.T) {
    // Test N+1 query resolution
    // Test pagination performance
    // Test index effectiveness
}
```

## Deployment Strategy

### Consolidated Deployment Configuration

```yaml
# deployment/production/docker-compose.yml
version: '3.8'
services:
  api:
    build:
      context: .
      dockerfile: deployment/production/Dockerfile
    environment:
      - ENV=production
      - DB_HOST=postgres
      - REDIS_HOST=redis
    depends_on:
      - postgres
      - redis
    
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: agentscan
      POSTGRES_USER: agentscan
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    
  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data
```

```dockerfile
# deployment/production/Dockerfile
FROM golang:1.23-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o agentscan ./cmd/api

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/agentscan .
EXPOSE 8080
CMD ["./agentscan"]
```

### Environment-Specific Configurations

```bash
# deployment/scripts/deploy-production.sh
#!/bin/bash
set -euo pipefail

# Single deployment script for production
echo "Deploying AgentScan to production..."

# Validate environment
./scripts/validate-env.sh production

# Build and deploy
docker-compose -f deployment/production/docker-compose.yml up -d

# Run health checks
./scripts/health-check.sh

echo "Deployment completed successfully!"
```

## Security Considerations

### Input Validation and Sanitization

```go
// internal/adapters/http/validators/security.go
type SecurityValidator struct {
    sanitizer *bluemonday.Policy
}

func (v *SecurityValidator) SanitizeInput(input string) string {
    return v.sanitizer.Sanitize(input)
}

func (v *SecurityValidator) ValidateSQL(query string) error {
    // Validate SQL queries for injection attempts
    if containsSQLInjection(query) {
        return errors.NewSecurityError("potential SQL injection detected")
    }
    return nil
}
```

### Authentication and Authorization

```go
// internal/domain/services/auth.go
type AuthService struct {
    supabaseClient *supabase.Client
    jwtValidator   *JWTValidator
}

func (s *AuthService) ValidateToken(ctx context.Context, token string) (*User, error) {
    // Validate Supabase JWT token
    claims, err := s.jwtValidator.ValidateToken(token)
    if err != nil {
        return nil, errors.NewAuthenticationError("invalid token")
    }
    
    // Get user from database
    user, err := s.userRepo.GetBySupabaseID(ctx, claims.Subject)
    if err != nil {
        return nil, errors.NewAuthenticationError("user not found")
    }
    
    return user, nil
}
```

## Performance Optimizations

### Caching Strategy

```go
// internal/infrastructure/cache/redis.go
type CacheService struct {
    client *redis.Client
}

func (c *CacheService) GetOrSet(ctx context.Context, key string, ttl time.Duration, fn func() (interface{}, error)) (interface{}, error) {
    // Try to get from cache first
    cached, err := c.client.Get(ctx, key).Result()
    if err == nil {
        return cached, nil
    }
    
    // If not in cache, execute function and cache result
    result, err := fn()
    if err != nil {
        return nil, err
    }
    
    // Cache the result
    c.client.Set(ctx, key, result, ttl)
    return result, nil
}
```

### Database Connection Pooling

```go
// internal/infrastructure/database/connection.go
func NewOptimizedDB(config DatabaseConfig) (*sqlx.DB, error) {
    db, err := sqlx.Connect("postgres", config.URL)
    if err != nil {
        return nil, err
    }
    
    // Optimize connection pool
    db.SetMaxOpenConns(config.MaxOpenConns)
    db.SetMaxIdleConns(config.MaxIdleConns)
    db.SetConnMaxLifetime(config.ConnMaxLifetime)
    db.SetConnMaxIdleTime(config.ConnMaxIdleTime)
    
    return db, nil
}
```

## Monitoring and Observability

### Structured Logging

```go
// internal/shared/logging/logger.go
type Logger struct {
    *zap.Logger
}

func (l *Logger) LogAPICall(ctx context.Context, method, path string, duration time.Duration, statusCode int) {
    l.Info("api_call",
        zap.String("request_id", getRequestID(ctx)),
        zap.String("method", method),
        zap.String("path", path),
        zap.Duration("duration", duration),
        zap.Int("status_code", statusCode),
        zap.String("user_id", getUserID(ctx)),
    )
}
```

### Metrics Collection

```go
// internal/shared/monitoring/metrics.go
type Metrics struct {
    requestDuration *prometheus.HistogramVec
    requestCount    *prometheus.CounterVec
    errorCount      *prometheus.CounterVec
}

func (m *Metrics) RecordRequest(method, path string, duration time.Duration, statusCode int) {
    m.requestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
    m.requestCount.WithLabelValues(method, path, strconv.Itoa(statusCode)).Inc()
    
    if statusCode >= 400 {
        m.errorCount.WithLabelValues(method, path, strconv.Itoa(statusCode)).Inc()
    }
}
```

This comprehensive design addresses all the critical issues identified in the codebase review and provides a clear path to transform AgentScan into a production-ready, enterprise-grade security scanning platform.