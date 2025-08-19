# Observe MCP Integration for Debugging

This document describes the Observe MCP (Model Context Protocol) integration for comprehensive debugging and monitoring of the AgentScan application.

## Overview

The Observe MCP integration provides:

- **Real-time Debugging**: Live monitoring of API calls, errors, and performance
- **Distributed Tracing**: Track requests across frontend and backend
- **Error Tracking**: Comprehensive error logging with context
- **Performance Monitoring**: API response times and system metrics
- **User Action Tracking**: Monitor user behavior and interactions
- **Custom Dashboards**: Create monitoring dashboards for specific metrics

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   Backend       │    │   Observe MCP   │
│                 │    │                 │    │                 │
│  ObserveLogger  ├────┤  ObserveLogger  ├────┤   Dashboard     │
│                 │    │                 │    │                 │
│  API Client     │    │  Middleware     │    │   Analytics     │
│                 │    │                 │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Setup

### 1. Configure MCP Server

The Observe MCP server is configured in `.kiro/settings/mcp.json`:

```json
{
  "mcpServers": {
    "observe": {
      "command": "npx",
      "args": [
        "mcp-remote@latest",
        "https://agentscan.observeinc.com/v1/ai/mcp",
        "--header",
        "Authorization:${OBSERVE_AUTH_HEADER}"
      ],
      "env": {
        "OBSERVE_AUTH_HEADER": "Bearer agentscan-prod ${OBSERVE_API_TOKEN}"
      },
      "disabled": false,
      "autoApprove": [
        "log_event",
        "create_trace",
        "log_error",
        "log_api_call",
        "query_logs",
        "create_dashboard"
      ]
    }
  }
}
```

### 2. Environment Configuration

#### Backend (.env)
```bash
OBSERVE_ENABLED=true
OBSERVE_ENDPOINT=https://agentscan.observeinc.com/v1
OBSERVE_API_KEY=your-observe-api-key-here
OBSERVE_PROJECT_ID=agentscan-backend
OBSERVE_ENVIRONMENT=development
```

#### Frontend (.env.development)
```bash
VITE_OBSERVE_ENABLED=true
VITE_OBSERVE_ENDPOINT=https://agentscan.observeinc.com/v1
VITE_OBSERVE_API_KEY=your-observe-api-key-here
VITE_OBSERVE_PROJECT_ID=agentscan-frontend-dev
```

### 3. Automated Setup

Use the setup script for easy configuration:

```bash
# Set your Observe API token
export OBSERVE_API_TOKEN=your-observe-api-token-here

# Run the setup script
./scripts/setup-observe-mcp.sh
```

## Usage

### Frontend Logging

The frontend automatically logs:

- **API Calls**: All HTTP requests with timing and response data
- **Errors**: JavaScript errors with stack traces and context
- **User Actions**: Button clicks, form submissions, navigation
- **Performance**: Page load times, component render times

```typescript
import { observeLogger } from './services/observeLogger'

// Manual logging
observeLogger.logEvent('info', 'User performed action', {
  action: 'scan_created',
  repository: 'my-repo'
})

// Error logging
observeLogger.logError(error, {
  component: 'ScanForm',
  userId: 'user-123'
})

// User action tracking
observeLogger.logUserAction('button_click', {
  button: 'create_scan',
  page: 'dashboard'
})
```

### Backend Logging

The backend automatically logs:

- **HTTP Requests**: All API endpoints with timing and status codes
- **Database Operations**: Query performance and errors
- **Scan Progress**: Real-time scan status updates
- **System Events**: Application startup, configuration changes

```go
import "github.com/NikhilSetiya/agentscan-security-scanner/internal/observability"

// Create logger
observeLogger := observability.NewObserveLogger(config, logger)

// Log events
observeLogger.LogEvent(ctx, "scan_started", "info", "Scan initiated", map[string]interface{}{
    "scan_id": scanID,
    "repository": repoName,
})

// Create traces
trace := observeLogger.CreateTrace(ctx, "process_scan")
defer observeLogger.EndTrace(ctx, trace, success, metadata)

// Log errors
observeLogger.LogError(ctx, err, map[string]interface{}{
    "operation": "database_query",
    "table": "scans",
})
```

### Middleware Integration

The backend includes automatic request logging middleware:

```go
// Add to your Gin router
observeMiddleware := observability.NewObserveMiddleware(observeLogger)
router.Use(observeMiddleware.LogRequest())
```

## Debugging Features

### 1. Real-time API Monitoring

Monitor all API calls in real-time:

- Request/response payloads
- Response times and status codes
- Error rates and patterns
- Authentication failures

### 2. Distributed Tracing

Track requests across the entire stack:

- Frontend user action → API call → Backend processing → Database query
- Identify bottlenecks and performance issues
- Correlate errors across services

### 3. Error Analysis

Comprehensive error tracking:

- Stack traces with source code context
- Error frequency and patterns
- User impact analysis
- Automatic error grouping

### 4. Performance Insights

Monitor application performance:

- API response times
- Database query performance
- Frontend rendering times
- Resource utilization

### 5. User Behavior Analytics

Track user interactions:

- Feature usage patterns
- User journey analysis
- Conversion funnel tracking
- A/B testing support

## Custom Dashboards

Create custom monitoring dashboards:

```typescript
// Create a dashboard for scan monitoring
await observeLogger.createDashboard('Scan Monitoring', [
  {
    name: 'Active Scans',
    query: 'event_type:scan_progress AND stage:running',
    visualization: 'number'
  },
  {
    name: 'Scan Success Rate',
    query: 'event_type:scan_completed',
    visualization: 'line'
  },
  {
    name: 'Error Rate by Endpoint',
    query: 'event_type:api_call AND status_code:>=400',
    visualization: 'bar'
  }
])
```

## Query Examples

### Find API Errors
```
event_type:api_call AND status_code:>=400
```

### Monitor Scan Performance
```
event_type:scan_progress AND progress:100
```

### Track User Actions
```
event_type:user_action AND action:scan_created
```

### Database Query Performance
```
event_type:trace_end AND operation:database_query AND duration_ms:>1000
```

### Authentication Issues
```
event_type:error AND message:*authentication*
```

## Troubleshooting

### Common Issues

1. **MCP Connection Failed**
   ```bash
   # Check MCP server status
   npx mcp-remote@latest https://agentscan.observeinc.com/v1/ai/mcp --test
   
   # Verify API token
   echo $OBSERVE_API_TOKEN
   ```

2. **No Logs Appearing**
   - Check `OBSERVE_ENABLED=true` in environment
   - Verify API key is correct
   - Check network connectivity to Observe endpoint

3. **High Latency**
   - Observe logging is asynchronous and shouldn't impact performance
   - Check network connection to Observe
   - Consider reducing log verbosity in production

### Debug Commands

```bash
# Test MCP connection
npx mcp-remote@latest https://agentscan.observeinc.com/v1/ai/mcp --header "Authorization:Bearer agentscan-prod $OBSERVE_API_TOKEN" --test

# Check environment configuration
env | grep OBSERVE

# View application logs
tail -f logs/application.log | grep OBSERVE
```

## Security Considerations

### Data Sanitization

All sensitive data is automatically sanitized:

- Passwords, tokens, and API keys are redacted
- Personal information is filtered
- Request/response bodies are sanitized

### Access Control

- API tokens are environment-specific
- Production and development use separate projects
- Service role authentication for backend

### Network Security

- All communication uses HTTPS/WSS
- API tokens are transmitted securely
- No sensitive data in query parameters

## Performance Impact

The Observe integration is designed for minimal performance impact:

- **Asynchronous Logging**: All logging is non-blocking
- **Intelligent Sampling**: High-frequency events are sampled
- **Local Caching**: Events are batched and sent periodically
- **Graceful Degradation**: Application continues if Observe is unavailable

## Best Practices

### Development

- Enable verbose logging for debugging
- Use traces for complex operations
- Log user actions for UX analysis
- Monitor API performance continuously

### Production

- Reduce log verbosity for performance
- Set up alerts for critical errors
- Monitor key business metrics
- Regular dashboard reviews

### Security

- Never log sensitive information
- Use environment-specific API keys
- Regular token rotation
- Monitor access patterns

## Integration with Kiro IDE

The Observe MCP integration works seamlessly with Kiro IDE:

1. **Real-time Debugging**: View logs and traces in real-time
2. **Error Navigation**: Jump directly to error locations in code
3. **Performance Profiling**: Identify slow operations
4. **User Session Replay**: Understand user behavior patterns

## Monitoring Checklist

- [ ] MCP server connection established
- [ ] Frontend logging enabled
- [ ] Backend middleware configured
- [ ] Custom dashboards created
- [ ] Alerts configured for critical errors
- [ ] Performance baselines established
- [ ] Security sanitization verified
- [ ] Team access configured

This comprehensive monitoring setup provides deep insights into application behavior, enabling rapid debugging and continuous performance optimization.