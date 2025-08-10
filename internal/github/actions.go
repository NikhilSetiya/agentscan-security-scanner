package github

import (
	"fmt"
	"strings"
)

// GenerateWorkflowYAML generates a GitHub Actions workflow for AgentScan
func GenerateWorkflowYAML(repoName string, options WorkflowOptions) string {
	var workflow strings.Builder

	workflow.WriteString("name: AgentScan Security\n\n")
	workflow.WriteString("on:\n")
	
	// Configure triggers
	if options.OnPush {
		workflow.WriteString("  push:\n")
		if len(options.PushBranches) > 0 {
			workflow.WriteString("    branches:\n")
			for _, branch := range options.PushBranches {
				workflow.WriteString(fmt.Sprintf("      - %s\n", branch))
			}
		}
	}
	
	if options.OnPullRequest {
		workflow.WriteString("  pull_request:\n")
		if len(options.PRBranches) > 0 {
			workflow.WriteString("    branches:\n")
			for _, branch := range options.PRBranches {
				workflow.WriteString(fmt.Sprintf("      - %s\n", branch))
			}
		}
	}
	
	if options.OnSchedule != "" {
		workflow.WriteString("  schedule:\n")
		workflow.WriteString(fmt.Sprintf("    - cron: '%s'\n", options.OnSchedule))
	}

	workflow.WriteString("\njobs:\n")
	workflow.WriteString("  agentscan:\n")
	workflow.WriteString("    name: AgentScan Security Analysis\n")
	workflow.WriteString("    runs-on: ubuntu-latest\n")
	
	if len(options.Permissions) > 0 {
		workflow.WriteString("    permissions:\n")
		for permission, level := range options.Permissions {
			workflow.WriteString(fmt.Sprintf("      %s: %s\n", permission, level))
		}
	} else {
		// Default permissions for security scanning
		workflow.WriteString("    permissions:\n")
		workflow.WriteString("      contents: read\n")
		workflow.WriteString("      security-events: write\n")
		workflow.WriteString("      pull-requests: write\n")
		workflow.WriteString("      checks: write\n")
	}

	workflow.WriteString("\n    steps:\n")
	workflow.WriteString("      - name: Checkout code\n")
	workflow.WriteString("        uses: actions/checkout@v4\n")
	workflow.WriteString("        with:\n")
	workflow.WriteString("          fetch-depth: 0\n")

	// Add language-specific setup steps
	if options.SetupNode {
		workflow.WriteString("\n      - name: Setup Node.js\n")
		workflow.WriteString("        uses: actions/setup-node@v4\n")
		workflow.WriteString("        with:\n")
		workflow.WriteString("          node-version: '18'\n")
		workflow.WriteString("          cache: 'npm'\n")
		workflow.WriteString("\n      - name: Install dependencies\n")
		workflow.WriteString("        run: npm ci\n")
	}

	if options.SetupPython {
		workflow.WriteString("\n      - name: Setup Python\n")
		workflow.WriteString("        uses: actions/setup-python@v4\n")
		workflow.WriteString("        with:\n")
		workflow.WriteString("          python-version: '3.11'\n")
		workflow.WriteString("          cache: 'pip'\n")
		workflow.WriteString("\n      - name: Install dependencies\n")
		workflow.WriteString("        run: |\n")
		workflow.WriteString("          python -m pip install --upgrade pip\n")
		workflow.WriteString("          if [ -f requirements.txt ]; then pip install -r requirements.txt; fi\n")
	}

	if options.SetupGo {
		workflow.WriteString("\n      - name: Setup Go\n")
		workflow.WriteString("        uses: actions/setup-go@v4\n")
		workflow.WriteString("        with:\n")
		workflow.WriteString("          go-version: '1.21'\n")
		workflow.WriteString("          cache: true\n")
		workflow.WriteString("\n      - name: Download dependencies\n")
		workflow.WriteString("        run: go mod download\n")
	}

	// AgentScan CLI step
	workflow.WriteString("\n      - name: Run AgentScan\n")
	workflow.WriteString("        uses: agentscan/agentscan-action@v1\n")
	workflow.WriteString("        with:\n")
	workflow.WriteString(fmt.Sprintf("          api-url: %s\n", options.APIUrl))
	
	if options.APIToken != "" {
		workflow.WriteString("          api-token: ${{ secrets.AGENTSCAN_TOKEN }}\n")
	}
	
	if options.FailOnHigh {
		workflow.WriteString("          fail-on-severity: high\n")
	}
	
	if options.FailOnMedium {
		workflow.WriteString("          fail-on-severity: medium\n")
	}
	
	if len(options.ExcludePaths) > 0 {
		workflow.WriteString("          exclude-paths: |\n")
		for _, path := range options.ExcludePaths {
			workflow.WriteString(fmt.Sprintf("            %s\n", path))
		}
	}
	
	if len(options.IncludeTools) > 0 {
		workflow.WriteString("          include-tools: " + strings.Join(options.IncludeTools, ",") + "\n")
	}
	
	if len(options.ExcludeTools) > 0 {
		workflow.WriteString("          exclude-tools: " + strings.Join(options.ExcludeTools, ",") + "\n")
	}

	// Upload results
	if options.UploadSARIF {
		workflow.WriteString("\n      - name: Upload SARIF results\n")
		workflow.WriteString("        uses: github/codeql-action/upload-sarif@v2\n")
		workflow.WriteString("        if: always()\n")
		workflow.WriteString("        with:\n")
		workflow.WriteString("          sarif_file: agentscan-results.sarif\n")
	}

	if options.UploadArtifacts {
		workflow.WriteString("\n      - name: Upload scan results\n")
		workflow.WriteString("        uses: actions/upload-artifact@v3\n")
		workflow.WriteString("        if: always()\n")
		workflow.WriteString("        with:\n")
		workflow.WriteString("          name: agentscan-results\n")
		workflow.WriteString("          path: |\n")
		workflow.WriteString("            agentscan-results.json\n")
		workflow.WriteString("            agentscan-results.sarif\n")
		workflow.WriteString("            agentscan-report.pdf\n")
	}

	return workflow.String()
}

// WorkflowOptions configures the generated GitHub Actions workflow
type WorkflowOptions struct {
	// Triggers
	OnPush        bool     `json:"on_push"`
	OnPullRequest bool     `json:"on_pull_request"`
	OnSchedule    string   `json:"on_schedule,omitempty"` // Cron expression
	PushBranches  []string `json:"push_branches,omitempty"`
	PRBranches    []string `json:"pr_branches,omitempty"`

	// Permissions
	Permissions map[string]string `json:"permissions,omitempty"`

	// Language setup
	SetupNode   bool `json:"setup_node"`
	SetupPython bool `json:"setup_python"`
	SetupGo     bool `json:"setup_go"`

	// AgentScan configuration
	APIUrl       string   `json:"api_url"`
	APIToken     string   `json:"api_token,omitempty"`
	FailOnHigh   bool     `json:"fail_on_high"`
	FailOnMedium bool     `json:"fail_on_medium"`
	ExcludePaths []string `json:"exclude_paths,omitempty"`
	IncludeTools []string `json:"include_tools,omitempty"`
	ExcludeTools []string `json:"exclude_tools,omitempty"`

	// Output options
	UploadSARIF     bool `json:"upload_sarif"`
	UploadArtifacts bool `json:"upload_artifacts"`
}

// DefaultWorkflowOptions returns sensible defaults for a GitHub Actions workflow
func DefaultWorkflowOptions() WorkflowOptions {
	return WorkflowOptions{
		OnPush:        true,
		OnPullRequest: true,
		PushBranches:  []string{"main", "master", "develop"},
		PRBranches:    []string{"main", "master"},
		APIUrl:        "https://api.agentscan.dev",
		FailOnHigh:    true,
		FailOnMedium:  false,
		UploadSARIF:   true,
		UploadArtifacts: true,
		ExcludePaths: []string{
			"node_modules/**",
			"vendor/**",
			"*.min.js",
			"*.test.js",
			"test/**",
			"tests/**",
			"__tests__/**",
		},
	}
}

// GenerateActionYAML generates the action.yml file for the AgentScan GitHub Action
func GenerateActionYAML() string {
	return `name: 'AgentScan Security Scanner'
description: 'Multi-agent security scanning with intelligent consensus'
author: 'AgentScan'

branding:
  icon: 'shield'
  color: 'blue'

inputs:
  api-url:
    description: 'AgentScan API URL'
    required: false
    default: 'https://api.agentscan.dev'
  
  api-token:
    description: 'AgentScan API token'
    required: false
  
  fail-on-severity:
    description: 'Fail the build on findings of this severity or higher (low, medium, high)'
    required: false
    default: 'high'
  
  exclude-paths:
    description: 'Paths to exclude from scanning (newline separated)'
    required: false
  
  include-tools:
    description: 'Comma-separated list of tools to include'
    required: false
  
  exclude-tools:
    description: 'Comma-separated list of tools to exclude'
    required: false
  
  output-format:
    description: 'Output format (json, sarif, pdf)'
    required: false
    default: 'json,sarif'

outputs:
  results-file:
    description: 'Path to the results file'
  
  findings-count:
    description: 'Total number of findings'
  
  high-severity-count:
    description: 'Number of high severity findings'
  
  medium-severity-count:
    description: 'Number of medium severity findings'
  
  low-severity-count:
    description: 'Number of low severity findings'

runs:
  using: 'docker'
  image: 'Dockerfile'
  args:
    - ${{ inputs.api-url }}
    - ${{ inputs.api-token }}
    - ${{ inputs.fail-on-severity }}
    - ${{ inputs.exclude-paths }}
    - ${{ inputs.include-tools }}
    - ${{ inputs.exclude-tools }}
    - ${{ inputs.output-format }}
`
}

// GenerateActionDockerfile generates the Dockerfile for the AgentScan GitHub Action
func GenerateActionDockerfile() string {
	return `FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o agentscan-cli ./cmd/cli

FROM alpine:latest

RUN apk --no-cache add ca-certificates git
WORKDIR /root/

COPY --from=builder /app/agentscan-cli .
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

ENTRYPOINT ["./entrypoint.sh"]
`
}

// GenerateActionEntrypoint generates the entrypoint script for the GitHub Action
func GenerateActionEntrypoint() string {
	return `#!/bin/sh

set -e

API_URL="$1"
API_TOKEN="$2"
FAIL_ON_SEVERITY="$3"
EXCLUDE_PATHS="$4"
INCLUDE_TOOLS="$5"
EXCLUDE_TOOLS="$6"
OUTPUT_FORMAT="$7"

echo "🔒 Starting AgentScan security analysis..."

# Build CLI arguments
ARGS="--api-url=$API_URL"

if [ -n "$API_TOKEN" ]; then
    ARGS="$ARGS --api-token=$API_TOKEN"
fi

if [ -n "$FAIL_ON_SEVERITY" ]; then
    ARGS="$ARGS --fail-on-severity=$FAIL_ON_SEVERITY"
fi

if [ -n "$EXCLUDE_PATHS" ]; then
    echo "$EXCLUDE_PATHS" | while IFS= read -r path; do
        if [ -n "$path" ]; then
            ARGS="$ARGS --exclude-path=$path"
        fi
    done
fi

if [ -n "$INCLUDE_TOOLS" ]; then
    ARGS="$ARGS --include-tools=$INCLUDE_TOOLS"
fi

if [ -n "$EXCLUDE_TOOLS" ]; then
    ARGS="$ARGS --exclude-tools=$EXCLUDE_TOOLS"
fi

if [ -n "$OUTPUT_FORMAT" ]; then
    ARGS="$ARGS --output-format=$OUTPUT_FORMAT"
fi

# Run AgentScan CLI
echo "Running: ./agentscan-cli scan $ARGS"
./agentscan-cli scan $ARGS

# Set outputs
if [ -f "agentscan-results.json" ]; then
    echo "results-file=agentscan-results.json" >> $GITHUB_OUTPUT
    
    # Extract counts from JSON results
    if command -v jq >/dev/null 2>&1; then
        TOTAL_COUNT=$(jq '.findings | length' agentscan-results.json)
        HIGH_COUNT=$(jq '.findings | map(select(.severity == "high")) | length' agentscan-results.json)
        MEDIUM_COUNT=$(jq '.findings | map(select(.severity == "medium")) | length' agentscan-results.json)
        LOW_COUNT=$(jq '.findings | map(select(.severity == "low")) | length' agentscan-results.json)
        
        echo "findings-count=$TOTAL_COUNT" >> $GITHUB_OUTPUT
        echo "high-severity-count=$HIGH_COUNT" >> $GITHUB_OUTPUT
        echo "medium-severity-count=$MEDIUM_COUNT" >> $GITHUB_OUTPUT
        echo "low-severity-count=$LOW_COUNT" >> $GITHUB_OUTPUT
        
        echo "📊 Scan complete: $TOTAL_COUNT findings ($HIGH_COUNT high, $MEDIUM_COUNT medium, $LOW_COUNT low)"
    fi
fi

echo "✅ AgentScan analysis complete"
`
}