# AgentScan Master Development Instructions

## Development Philosophy: Think → Build → Test → Commit

This document provides the overarching framework for building AgentScan. Every feature, component, and integration must follow this cycle with multiple rounds of reasoning to minimize errors and technical debt.

## Core Principles

### 1. Ultra-Think Before Building
- **Multiple Reasoning Rounds**: Every decision requires at least 3 rounds of analysis
- **Consider Edge Cases**: Think through failure modes, scale issues, and user experience
- **Document Reasoning**: All decisions must be documented in scratchpad.md
- **Challenge Assumptions**: Question every requirement and design choice

### 2. Incremental Development
- **Small, Testable Units**: Build the smallest possible working component first
- **Vertical Slices**: Complete end-to-end functionality before adding breadth
- **Fail Fast**: Identify problems early through rapid prototyping
- **Continuous Integration**: Every commit should maintain system stability

### 3. Quality Gates
- **No Code Without Tests**: Every function must have corresponding tests
- **Performance Benchmarks**: Measure and document performance characteristics
- **Security Review**: Every component must pass security analysis
- **Documentation**: Code must be self-documenting with clear interfaces

## Development Cycle Framework

### THINK Phase (30% of development time)

#### Round 1: Problem Analysis
1. **What exactly are we solving?**
   - Define the specific problem in user terms
   - Identify success criteria and failure modes
   - Consider alternative approaches

2. **Who are the stakeholders?**
   - Primary users and their workflows
   - Secondary users and edge cases
   - System administrators and operators

3. **What are the constraints?**
   - Performance requirements
   - Security requirements
   - Integration requirements
   - Resource limitations

#### Round 2: Solution Design
1. **Architecture Decisions**
   - Component boundaries and interfaces
   - Data flow and state management
   - Error handling and recovery
   - Scalability considerations

2. **Technology Choices**
   - Language and framework selection
   - Database and storage decisions
   - Third-party dependencies
   - Deployment and infrastructure

3. **Risk Assessment**
   - Technical risks and mitigation strategies
   - Business risks and contingencies
   - Security vulnerabilities and protections
   - Performance bottlenecks and optimizations

#### Round 3: Implementation Planning
1. **Break Down Into Tasks**
   - Identify minimal viable components
   - Define clear interfaces and contracts
   - Plan testing strategy
   - Estimate effort and dependencies

2. **Validate Assumptions**
   - Prototype critical components
   - Test integration points
   - Verify performance characteristics
   - Confirm security model

3. **Document Decisions**
   - Update scratchpad.md with reasoning
   - Create architectural decision records
   - Define acceptance criteria
   - Plan rollback strategies

### BUILD Phase (40% of development time)

#### Code Quality Standards
1. **Interface-First Development**
   - Define clear, minimal interfaces
   - Use dependency injection for testability
   - Implement proper error handling
   - Follow language-specific best practices

2. **Security by Design**
   - Input validation and sanitization
   - Proper authentication and authorization
   - Secure data handling and storage
   - Regular security dependency updates

3. **Performance Considerations**
   - Efficient algorithms and data structures
   - Proper resource management
   - Caching strategies where appropriate
   - Monitoring and observability hooks

#### Implementation Guidelines
1. **Start with the Interface**
   ```go
   // Define interface first
   type SecurityAgent interface {
       Scan(ctx context.Context, config ScanConfig) (*ScanResult, error)
       HealthCheck(ctx context.Context) error
       GetConfig() AgentConfig
   }
   ```

2. **Implement Core Logic**
   - Focus on the happy path first
   - Add error handling incrementally
   - Use proper logging and metrics
   - Handle context cancellation

3. **Add Configuration**
   - Environment-based configuration
   - Validation of configuration values
   - Sensible defaults
   - Configuration documentation

### TEST Phase (20% of development time)

#### Testing Strategy
1. **Unit Tests (70% of test effort)**
   - Test all public interfaces
   - Test error conditions
   - Test edge cases and boundary conditions
   - Achieve >90% code coverage

2. **Integration Tests (20% of test effort)**
   - Test component interactions
   - Test external service integrations
   - Test database operations
   - Test concurrent operations

3. **End-to-End Tests (10% of test effort)**
   - Test complete user workflows
   - Test system under load
   - Test failure scenarios
   - Test security boundaries

#### Test Implementation
1. **Test-Driven Development**
   ```go
   func TestSemgrepAgent_Scan(t *testing.T) {
       tests := []struct {
           name     string
           config   ScanConfig
           expected *ScanResult
           wantErr  bool
       }{
           // Test cases here
       }
       
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               // Test implementation
           })
       }
   }
   ```

2. **Mock External Dependencies**
   - Use interfaces for all external dependencies
   - Create mock implementations for testing
   - Test both success and failure scenarios
   - Verify interaction patterns

3. **Performance Testing**
   - Benchmark critical paths
   - Test memory usage patterns
   - Test concurrent access
   - Monitor resource consumption

### COMMIT Phase (10% of development time)

#### Pre-Commit Checklist
1. **Code Quality**
   - [ ] All tests pass
   - [ ] Code coverage meets standards
   - [ ] No security vulnerabilities detected
   - [ ] Performance benchmarks within acceptable range

2. **Documentation**
   - [ ] Public interfaces documented
   - [ ] README updated if needed
   - [ ] Architecture decisions recorded
   - [ ] Configuration options documented

3. **Integration**
   - [ ] No breaking changes to existing APIs
   - [ ] Database migrations tested
   - [ ] Deployment scripts updated
   - [ ] Monitoring and alerting configured

#### Commit Message Format
```
type(scope): brief description

Detailed explanation of what changed and why.

- Specific change 1
- Specific change 2

Closes #issue-number
```

## Component-Specific Guidelines

### Agent Development
1. **Agent Interface Compliance**
   - Implement all required interface methods
   - Handle context cancellation properly
   - Return standardized result format
   - Provide meaningful error messages

2. **Container Integration**
   - Use official tool Docker images when available
   - Implement proper resource limits
   - Handle container lifecycle management
   - Provide health check endpoints

3. **Result Standardization**
   - Map tool-specific output to common format
   - Preserve original tool information
   - Include confidence scores
   - Handle partial results gracefully

### API Development
1. **RESTful Design**
   - Use appropriate HTTP methods
   - Implement proper status codes
   - Provide consistent error responses
   - Include rate limiting

2. **Authentication & Authorization**
   - Implement JWT-based authentication
   - Use role-based access control
   - Audit all access attempts
   - Handle token expiration gracefully

3. **Performance**
   - Implement request/response caching
   - Use database connection pooling
   - Implement proper pagination
   - Monitor response times

### Database Operations
1. **Schema Design**
   - Use appropriate data types
   - Implement proper indexing
   - Design for query patterns
   - Plan for data growth

2. **Migration Strategy**
   - All schema changes via migrations
   - Test migrations on production-like data
   - Implement rollback procedures
   - Document breaking changes

3. **Query Optimization**
   - Use prepared statements
   - Implement query result caching
   - Monitor slow queries
   - Optimize based on usage patterns

## Error Handling Strategy

### Error Categories
1. **User Errors** (4xx)
   - Invalid input data
   - Authentication failures
   - Authorization violations
   - Resource not found

2. **System Errors** (5xx)
   - Database connection failures
   - External service timeouts
   - Resource exhaustion
   - Unexpected exceptions

3. **Agent Errors**
   - Tool execution failures
   - Container startup issues
   - Result parsing errors
   - Timeout conditions

### Error Response Format
```json
{
  "error": {
    "code": "SCAN_FAILED",
    "message": "Semgrep agent failed to complete scan",
    "details": {
      "agent": "semgrep",
      "exit_code": 1,
      "stderr": "Error message from tool"
    },
    "request_id": "req_123456789"
  }
}
```

## Monitoring and Observability

### Metrics to Track
1. **Business Metrics**
   - Scan completion rate
   - False positive rate
   - User engagement
   - Feature adoption

2. **Technical Metrics**
   - Response times
   - Error rates
   - Resource utilization
   - Queue depths

3. **Security Metrics**
   - Authentication failures
   - Authorization violations
   - Suspicious activity
   - Vulnerability detections

### Logging Standards
1. **Structured Logging**
   ```json
   {
     "timestamp": "2024-01-15T10:30:00Z",
     "level": "INFO",
     "component": "orchestrator",
     "request_id": "req_123456789",
     "message": "Scan completed successfully",
     "metadata": {
       "scan_id": "scan_987654321",
       "duration_ms": 45000,
       "agents_used": ["semgrep", "eslint"],
       "findings_count": 12
     }
   }
   ```

2. **Log Levels**
   - ERROR: System errors requiring immediate attention
   - WARN: Potential issues that should be monitored
   - INFO: Important business events
   - DEBUG: Detailed execution information

## Security Requirements

### Data Protection
1. **Encryption**
   - All data encrypted at rest
   - TLS 1.3 for data in transit
   - Proper key management
   - Regular key rotation

2. **Access Control**
   - Principle of least privilege
   - Regular access reviews
   - Multi-factor authentication
   - Session management

3. **Audit Trail**
   - Log all access attempts
   - Track data modifications
   - Monitor privileged operations
   - Retain logs per compliance requirements

### Secure Development
1. **Code Security**
   - Regular dependency updates
   - Static code analysis
   - Security code reviews
   - Penetration testing

2. **Infrastructure Security**
   - Container security scanning
   - Network segmentation
   - Regular security patches
   - Intrusion detection

## Performance Requirements

### Response Time Targets
- API responses: < 200ms (95th percentile)
- Full repository scan: < 5 minutes (100k LOC)
- Incremental scan: < 30 seconds
- IDE feedback: < 2 seconds

### Scalability Targets
- Support 1000+ concurrent scans
- Handle 10,000+ developers
- Process 1M+ LOC repositories
- 99.9% uptime SLA

### Resource Optimization
1. **Memory Management**
   - Proper garbage collection
   - Memory leak detection
   - Resource pooling
   - Efficient data structures

2. **CPU Optimization**
   - Parallel processing where appropriate
   - Efficient algorithms
   - Proper caching
   - Load balancing

## Deployment Strategy

### Environment Management
1. **Development**
   - Local Docker Compose setup
   - Automated testing pipeline
   - Feature branch deployments
   - Performance profiling

2. **Staging**
   - Production-like environment
   - Integration testing
   - Security testing
   - Performance testing

3. **Production**
   - Blue-green deployments
   - Automated rollback capability
   - Health checks and monitoring
   - Disaster recovery procedures

### Configuration Management
1. **Environment Variables**
   - Separate config per environment
   - Secure secret management
   - Configuration validation
   - Default value handling

2. **Feature Flags**
   - Gradual feature rollouts
   - A/B testing capability
   - Emergency feature toggles
   - User-based feature access

This master instruction document should be referenced for every development decision and updated as the project evolves. The goal is to build a robust, scalable, and maintainable security scanning platform that developers actually want to use.