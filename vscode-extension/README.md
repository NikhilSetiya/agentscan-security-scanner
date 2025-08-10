# AgentScan Security - VS Code Extension

Real-time security scanning with multi-agent consensus directly in your VS Code editor.

## Features

### 🔍 Real-time Security Scanning
- **Instant Feedback**: Get security findings as you type with debounced scanning
- **File-level Scanning**: Scan individual files on save or on-demand
- **Workspace Scanning**: Comprehensive security analysis of your entire project

### 🤖 Multi-Agent Consensus
- **Multiple Tools**: Leverages Semgrep, ESLint Security, Bandit, and more
- **Consensus Scoring**: Reduces false positives through intelligent agreement analysis
- **High Confidence**: Only shows findings that multiple tools agree on

### 🎨 Clean, Minimal UI
- **Inline Annotations**: Security issues highlighted directly in your code
- **Hover Tooltips**: Detailed information without cluttering your workspace
- **Quiet Luxury Design**: Inspired by Linear, Vercel, and Superhuman aesthetics
- **Severity-based Colors**: Visual hierarchy with high/medium/low severity indicators

### ⚡ Live Updates
- **WebSocket Connection**: Real-time updates from the AgentScan server
- **Progress Tracking**: See scan progress in real-time
- **Status Bar Integration**: Always know the connection and scan status

### 🛠️ Advanced Features
- **Finding Suppression**: Mark false positives to improve accuracy over time
- **Configurable Thresholds**: Set minimum severity levels to reduce noise
- **Language Support**: JavaScript, TypeScript, Python, Go, Java, and more
- **Export Capabilities**: Generate PDF and JSON reports of findings

## Installation

1. Install the extension from the VS Code Marketplace
2. Configure your AgentScan server URL and API key in settings
3. Start scanning your code for security issues!

## Configuration

Open VS Code settings and search for "AgentScan" to configure:

### Required Settings
- **Server URL**: Your AgentScan server endpoint (default: `http://localhost:8080`)
- **API Key**: Your authentication token for the AgentScan API

### Optional Settings
- **Enable Real-time Scanning**: Scan files automatically on save (default: `true`)
- **Scan Debounce**: Delay before triggering scans in milliseconds (default: `1000`)
- **Enabled Languages**: Languages to scan (default: `["javascript", "typescript", "python", "go", "java"]`)
- **Severity Threshold**: Minimum severity to show findings (default: `"medium"`)
- **Inline Annotations**: Show security annotations in editor (default: `true`)
- **WebSocket Connection**: Enable real-time updates (default: `true`)

## Usage

### Scanning Files
- **Automatic**: Files are scanned automatically when saved (if enabled)
- **Manual**: Use `Ctrl+Shift+P` → "AgentScan: Scan Current File"
- **Workspace**: Use `Ctrl+Shift+P` → "AgentScan: Scan Workspace"

### Viewing Findings
- **Inline**: Security issues are highlighted directly in your code
- **Problems Panel**: View all findings in the VS Code Problems panel
- **Tree View**: Use the AgentScan sidebar panel for organized findings view
- **Hover**: Hover over highlighted code for detailed information

### Managing Findings
- **Suppress**: Right-click on a finding to suppress false positives
- **Mark as Fixed**: Update finding status when issues are resolved
- **Export**: Generate reports for sharing with your team

### Commands
- `AgentScan: Scan Current File` - Scan the currently open file
- `AgentScan: Scan Workspace` - Scan the entire workspace
- `AgentScan: Clear All Findings` - Clear all security findings
- `AgentScan: Open Settings` - Open AgentScan configuration
- `AgentScan: Suppress Finding` - Suppress a false positive

## Security Findings

### Severity Levels
- **🔴 High**: Critical security vulnerabilities requiring immediate attention
- **🟡 Medium**: Important security issues that should be addressed
- **🔵 Low**: Minor security concerns or best practice violations

### Supported Vulnerability Types
- SQL Injection
- Cross-Site Scripting (XSS)
- Command Injection
- Path Traversal
- Insecure Cryptography
- Hardcoded Secrets
- Dependency Vulnerabilities
- Configuration Issues

## Requirements

- VS Code 1.74.0 or higher
- AgentScan server running and accessible
- Valid API key for authentication

## Extension Settings

This extension contributes the following settings:

* `agentscan.serverUrl`: AgentScan server URL
* `agentscan.apiKey`: API key for authentication
* `agentscan.enableRealTimeScanning`: Enable real-time scanning on file save
* `agentscan.scanDebounceMs`: Debounce delay for real-time scanning
* `agentscan.enabledLanguages`: Languages to scan for security issues
* `agentscan.severityThreshold`: Minimum severity level to show findings
* `agentscan.showInlineAnnotations`: Show inline security annotations
* `agentscan.enableWebSocket`: Enable WebSocket for real-time updates

## Known Issues

- Large files (>10MB) may take longer to scan
- WebSocket connection may occasionally disconnect and reconnect
- Some findings may require manual review for accuracy

## Release Notes

### 0.1.0

Initial release of AgentScan Security extension:
- Real-time security scanning
- Multi-agent consensus engine
- Clean, minimal UI design
- WebSocket integration for live updates
- Finding suppression and management
- Comprehensive language support

## Contributing

Found a bug or have a feature request? Please open an issue on our [GitHub repository](https://github.com/NikhilSetiya/agentscan-security-scanner).

## License

This extension is licensed under the MIT License. See the LICENSE file for details.

---

**Enjoy secure coding with AgentScan! 🛡️**