#!/bin/bash

# AgentScan Observe MCP Setup Script
# This script helps configure Observe MCP for debugging and monitoring

set -e

echo "🔍 AgentScan Observe MCP Setup"
echo "================================"

# Check if required environment variables are set
if [ -z "$OBSERVE_API_TOKEN" ]; then
    echo "❌ OBSERVE_API_TOKEN environment variable is required"
    echo "Please set it with your Observe API token:"
    echo "export OBSERVE_API_TOKEN=your-observe-api-token-here"
    exit 1
fi

# Update MCP configuration
echo "📝 Updating MCP configuration..."

# Create the MCP config with proper Observe settings
cat > .kiro/settings/mcp.json << EOF
{
  "mcpServers": {
    "react-bits-mcp": {
      "command": "npx",
      "args": ["mcp-remote", "https://react-bits-mcp.davidhzdev.workers.dev/sse"],
      "disabled": false
    },
    "observe": {
      "command": "npx",
      "args": [
        "mcp-remote@latest",
        "https://agentscan.observeinc.com/v1/ai/mcp",
        "--header",
        "Authorization:\${OBSERVE_AUTH_HEADER}"
      ],
      "env": {
        "OBSERVE_AUTH_HEADER": "Bearer agentscan-prod $OBSERVE_API_TOKEN"
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
EOF

echo "✅ MCP configuration updated"

# Update environment files
echo "📝 Updating environment configuration..."

# Update .env if it exists
if [ -f ".env" ]; then
    # Add Observe configuration if not already present
    if ! grep -q "OBSERVE_ENABLED" .env; then
        echo "" >> .env
        echo "# Observe MCP Configuration" >> .env
        echo "OBSERVE_ENABLED=true" >> .env
        echo "OBSERVE_ENDPOINT=https://agentscan.observeinc.com/v1" >> .env
        echo "OBSERVE_API_KEY=$OBSERVE_API_TOKEN" >> .env
        echo "OBSERVE_PROJECT_ID=agentscan-backend" >> .env
        echo "OBSERVE_ENVIRONMENT=development" >> .env
        echo "✅ Updated .env file"
    else
        echo "⚠️  Observe configuration already exists in .env"
    fi
fi

# Update frontend environment files
if [ -f "web/frontend/.env.development" ]; then
    if ! grep -q "VITE_OBSERVE_ENABLED" web/frontend/.env.development; then
        echo "" >> web/frontend/.env.development
        echo "# Observe MCP Configuration" >> web/frontend/.env.development
        echo "VITE_OBSERVE_ENABLED=true" >> web/frontend/.env.development
        echo "VITE_OBSERVE_ENDPOINT=https://agentscan.observeinc.com/v1" >> web/frontend/.env.development
        echo "VITE_OBSERVE_API_KEY=$OBSERVE_API_TOKEN" >> web/frontend/.env.development
        echo "VITE_OBSERVE_PROJECT_ID=agentscan-frontend-dev" >> web/frontend/.env.development
        echo "✅ Updated frontend development environment"
    else
        echo "⚠️  Observe configuration already exists in frontend .env.development"
    fi
fi

# Test MCP connection
echo "🧪 Testing MCP connection..."

# Check if npx is available
if ! command -v npx &> /dev/null; then
    echo "❌ npx is not installed. Please install Node.js and npm first."
    exit 1
fi

# Test the MCP remote connection
echo "Testing Observe MCP connection..."
timeout 10s npx mcp-remote@latest https://agentscan.observeinc.com/v1/ai/mcp --header "Authorization:Bearer agentscan-prod $OBSERVE_API_TOKEN" --test || {
    echo "⚠️  MCP connection test failed or timed out"
    echo "This might be normal if the Observe endpoint is not yet configured"
}

echo ""
echo "🎉 Observe MCP setup complete!"
echo ""
echo "Configuration Summary:"
echo "======================"
echo "• MCP Server: Configured in .kiro/settings/mcp.json"
echo "• Backend: Observe logging enabled in environment"
echo "• Frontend: Observe logging enabled in development"
echo "• API Token: Configured (hidden for security)"
echo ""
echo "Next steps:"
echo "1. Restart your development environment"
echo "2. Check logs for Observe MCP connection status"
echo "3. Use the debugging features in your IDE"
echo "4. Monitor application logs in Observe dashboard"
echo ""
echo "Debugging commands:"
echo "• View MCP logs: Check Kiro IDE MCP panel"
echo "• Test API calls: Use the integrated debugging tools"
echo "• Monitor performance: Check Observe dashboard"