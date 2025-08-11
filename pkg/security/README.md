# Security Package

This package provides comprehensive security hardening and compliance features for the AgentScan security scanner system.

## Components Implemented

### 1. Data Encryption (`encryption.go`)

Provides robust data encryption capabilities for protecting sensitive information at rest and in transit.

**Features:**
- AES-GCM encryption for data at rest
- PBKDF2 key derivation with salt
- Automatic sensitive field detection and encryption
- Secure password hashing and verification
- Cryptographically secure token generation

**Usage:**
```go
encSvc := NewEncryptionService("your-encryption-key")

// Encrypt sensitive data
encrypted, err := encSvc.Encrypt("sensitive-data")

// Encrypt sensitive fields in a map
data := map[string]interface{}{
    "username": "user",
    "password": "secret", // Will be encrypted
    "api_key":  "key123", // Will be encrypted
}
encryptedData, err := encSvc.EncryptSensitiveFields(data)

// Hash passwords securely
hash, err := encSvc.HashPassword("user-password")
valid, err := encSvc.VerifyPassword("user-password", hash)
```

### 2. Comprehensive Audit Logging (`audit.go`)

Implements comprehensive audit logging for compliance requirements with automatic encryption of sensitive data.

**Features:**
- Structured audit events with multiple event types
- Automatic encryption of sensitive audit data
- IP address and user agent tracking
- Request correlation ID support
- Compliance reporting capabilities
- 7-year retention period support

**Event Types:**
- Authentication events (login, logout, failures)
- Authorization events (access granted/denied)
- Data access events (read, write, delete, export)
- Scan events (started, completed, failed)
- Configuration changes
- Security violations

**Usage:**
```go
auditLogger := NewAuditLogger("agentscan", "1.0.0", encryptionSvc)

// Log authentication event
auditLogger.LogAuthenticationEvent(ctx, EventTypeLogin, userID, username, true, details)

// Log authorization event
auditLogger.LogAuthorizationEvent(ctx, userID, username, resource, action, granted, details)

// Log security violation
auditLogger.LogSecurityEvent(ctx, EventTypeSecurityViolation, "Suspicious activity", details)
```

### 3. Security Headers and CORS (`headers.go`)

Implements comprehensive HTTP security headers and CORS configuration to protect against common web vulnerabilities.

**Security Headers:**
- Content Security Policy (CSP) with strict directives
- HTTP Strict Transport Security (HSTS) with preload
- X-Frame-Options (clickjacking protection)
- X-Content-Type-Options (MIME sniffing protection)
- X-XSS-Protection (XSS filtering)
- Referrer-Policy (referrer information control)
- Permissions-Policy (feature policy)

**CORS Features:**
- Configurable allowed origins with wildcard support
- Method and header restrictions
- Credential handling
- Preflight request support

**Usage:**
```go
config := DefaultSecurityHeadersConfig()
router.Use(SecurityHeadersMiddleware(config))
router.Use(CORSMiddleware(config))
```

### 4. Rate Limiting and DDoS Protection (`ratelimit.go`)

Provides multi-layered rate limiting and DDoS protection with Redis-based distributed limiting.

**Rate Limiting Layers:**
- Global rate limits (system-wide)
- Per-IP rate limits
- Per-user rate limits (authenticated users)
- Endpoint-specific rate limits
- DDoS attack detection and mitigation

**Features:**
- Redis-based distributed rate limiting
- Local cache fallback when Redis unavailable
- IP whitelist and blacklist support
- Automatic DDoS detection and blocking
- Configurable rate limits per endpoint
- Comprehensive audit logging of violations

**Usage:**
```go
config := DefaultRateLimitConfig()
rateLimiter := NewRateLimiter(config, auditLogger)
router.Use(rateLimiter.RateLimitMiddleware())
```

### 5. Security Testing Suite (`testing.go`)

Automated security testing capabilities to validate security configurations and detect vulnerabilities.

**Security Tests:**
- TLS/SSL configuration validation
- Certificate expiration checking
- Security headers verification
- CORS configuration testing
- Authentication endpoint security
- JWT token validation
- SQL injection protection
- XSS protection validation
- CSRF protection testing
- Rate limiting verification
- Information disclosure detection
- Error handling security

**Usage:**
```go
testSuite := NewSecurityTestSuite("https://your-api.com")
results := testSuite.RunAllTests(ctx)

// Get failed tests
failedTests := testSuite.GetFailedTests()

// Get tests by severity
criticalIssues := testSuite.GetTestsBySeverity("critical")
```

### 6. Security Configuration Management (`config.go`)

Centralized security configuration management with environment-specific presets.

**Configuration Profiles:**
- **Default**: Balanced security for general use
- **Production**: Maximum security for production environments
- **Development**: Developer-friendly with relaxed restrictions

**Features:**
- Environment variable integration
- Configuration validation
- Security manager for component coordination
- Middleware integration helpers

**Usage:**
```go
// Production configuration
config := ProductionSecurityConfig()

// Development configuration
config := DevelopmentSecurityConfig()

// Create security manager
securityManager, err := NewSecurityManager(config, redisClient)

// Get all security middleware
middleware := securityManager.GetSecurityMiddleware()
for _, mw := range middleware {
    router.Use(mw)
}
```

## Integration with AgentScan

The security package integrates seamlessly with AgentScan's architecture:

1. **API Security**: All REST API endpoints are protected with security headers, CORS, and rate limiting
2. **Data Protection**: Sensitive data (API keys, tokens, passwords) is encrypted at rest
3. **Audit Compliance**: All security-relevant events are logged for compliance requirements
4. **DDoS Protection**: Multi-layered rate limiting protects against abuse and attacks
5. **Automated Testing**: Continuous security validation through automated testing

## Compliance Features

The package addresses multiple compliance requirements:

- **SOC 2**: Comprehensive audit logging and access controls
- **GDPR**: Data encryption and retention policies
- **HIPAA**: Encryption at rest and in transit
- **PCI DSS**: Secure data handling and access logging
- **ISO 27001**: Security controls and monitoring

## Requirements Satisfied

This implementation satisfies the following requirements:

- **7.4**: ✅ Data encryption at rest and in transit
- **7.5**: ✅ Comprehensive audit logging for compliance
- **Security Headers**: ✅ CORS configuration and security headers
- **Rate Limiting**: ✅ DDoS protection and rate limiting
- **Security Testing**: ✅ Automated vulnerability assessments

## Testing

The package includes comprehensive tests:

```bash
# Run all security tests
go test ./pkg/security/... -v

# Run specific test suites
go test ./pkg/security -run TestEncryption -v
go test ./pkg/security -run TestHeaders -v
go test ./pkg/security -run TestRateLimit -v
```

## Environment Variables

Key environment variables for security configuration:

```bash
# Encryption
ENCRYPTION_KEY=your-32-character-encryption-key

# TLS
TLS_CERT_FILE=/path/to/cert.pem
TLS_KEY_FILE=/path/to/key.pem
TLS_MIN_VERSION=1.2

# Session
SESSION_SECURE_COOKIES=true
SESSION_HTTP_ONLY=true
SESSION_SAME_SITE=strict
SESSION_TIMEOUT=24h

# Audit
AUDIT_LOG_ENABLED=true
AUDIT_LOG_ENCRYPT=true
AUDIT_LOG_RETENTION=8760h

# Rate Limiting
RATE_LIMIT_PER_IP_RPS=100
RATE_LIMIT_DDOS_THRESHOLD=500

# CORS
CORS_ALLOWED_ORIGINS=https://app.agentscan.dev,https://dashboard.agentscan.dev
CORS_ALLOW_CREDENTIALS=true
```

## Security Best Practices

The package implements industry security best practices:

1. **Defense in Depth**: Multiple security layers
2. **Principle of Least Privilege**: Minimal required permissions
3. **Secure by Default**: Secure default configurations
4. **Zero Trust**: Verify all requests and users
5. **Continuous Monitoring**: Comprehensive audit logging
6. **Automated Testing**: Regular security validation

This security package provides enterprise-grade security hardening that ensures AgentScan meets the highest security and compliance standards.