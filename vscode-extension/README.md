# AgentScan Security Extension

Real-time security scanning with multi-agent consensus for VS Code. Get instant feedback on security vulnerabilities with sub-2-second response times and intelligent caching.

## ✨ Features

### 🚀 **Real-time Security Scanning**
- **Lightning Fast**: Sub-2-second response times with intelligent caching
- **Live Feedback**: Scan as you type with debounced triggers
- **Smart Caching**: Avoid redundant scans with content-based caching

### 🎯 **Multi-Agent Consensus**
- **Reduce False Positives**: Intelligent consensus scoring across multiple security tools
- **High Confidence Results**: Only show findings validated by multiple agents
- **Tool Reliability**: Track and weight tool accuracy over time

### 💡 **Rich Developer Experience**
- **Inline Annotations**: See security issues directly in your code
- **Rich Hover Tooltips**: Detailed vulnerability information with fix suggestions
- **Code Actions**: Quick fixes for common security issues
- **Keyboard Navigation**: Navigate between findings with F8/Shift+F8

### 🔧 **Quick Actions**
- **Suppress Finding**: Mark false positives with reason
- **Mark as Fixed**: Track resolution status
- **Ignore Rule**: Disable specific security rules
- **Learn More**: Detailed vulnerability documentation

### 📊 **Security Health Dashboard**
- **Health Score**: Overall security posture visualization
- **Trend Analysis**: Track security improvements over time
- **Detailed Statistics**: Breakdown by severity and tool

### 🌐 **Robust Connectivity**
- **WebSocket Resilience**: Automatic reconnection with exponential backoff
- **Offline Mode**: Continue working when server is unavailable
- **Connection Quality**: Monitor and display connection health

### 📈 **Performance & Telemetry**
- **Performance Monitoring**: Track scan times and cache hit rates
- **Usage Analytics**: Understand extension usage patterns
- **Crash Reporting**: Automatic error reporting for reliability

## 🎮 Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+Shift+S` (`Cmd+Shift+S` on Mac) | Scan Current File |
| `F8` | Go to Next Finding |
| `Shift+F8` | Go to Previous Finding |
| `Ctrl+Shift+I` (`Cmd+Shift+I` on Mac) | Suppress Finding |
| `Ctrl+Shift+T` (`Cmd+Shift+T` on Mac) | Toggle Real-time Scanning |

## 📋 Requirements

- VS Code 1.74.0 or higher
- AgentScan server running locally or remotely
- Valid API key for AgentScan service

## ⚙️ Extension Settings

### Core Settings
* `agentscan.serverUrl`: AgentScan server URL (default: `http://localhost:8080`)
* `agentscan.apiKey`: API key for server authentication
* `agentscan.enableRealTimeScanning`: Enable real-time scanning (default: `true`)

### Performance Settings
* `agentscan.scanDebounceMs`: Debounce delay for real-time scanning (default: `1000`)
* `agentscan.maxConcurrentScans`: Maximum concurrent file scans (default: `3`)
* `agentscan.cacheEnabled`: Enable result caching (default: `true`)
* `agentscan.cacheMaxAge`: Cache expiration time in seconds (default: `300`)

### Display Settings
* `agentscan.enabledLanguages`: Languages to scan (default: `["javascript", "typescript", "python", "go", "java"]`)
* `agentscan.severityThreshold`: Minimum severity to show (default: `"medium"`)
* `agentscan.showInlineAnnotations`: Show inline annotations (default: `true`)
* `agentscan.showSecurityHealth`: Show security health in status bar (default: `true`)

### Advanced Settings
* `agentscan.enableWebSocket`: Enable WebSocket for real-time updates (default: `true`)
* `agentscan.enableTelemetry`: Enable usage analytics and crash reporting (default: `true`)

## 🎯 Commands

### Scanning Commands
* `AgentScan: Scan Current File` - Scan the currently active file
* `AgentScan: Scan Workspace` - Scan the entire workspace
* `AgentScan: Clear All Findings` - Clear all security findings

### Navigation Commands
* `AgentScan: Go to Next Finding` - Navigate to next security finding
* `AgentScan: Go to Previous Finding` - Navigate to previous security finding

### Management Commands
* `AgentScan: Suppress Finding` - Suppress a false positive
* `AgentScan: Mark as Fixed` - Mark finding as resolved
* `AgentScan: Ignore Rule` - Disable a specific security rule
* `AgentScan: Learn More` - Open detailed vulnerability information

### Utility Commands
* `AgentScan: Toggle Real-time Scanning` - Enable/disable live scanning
* `AgentScan: Show Security Health` - Open security dashboard
* `AgentScan: Open Settings` - Open AgentScan configuration

## 🚀 Getting Started

1. **Install the Extension**: Search for "AgentScan Security" in the VS Code marketplace
2. **Configure Server**: Set your AgentScan server URL in settings
3. **Add API Key**: Configure your authentication key
4. **Start Scanning**: Open a supported file and save to trigger your first scan

## 🔍 Supported Languages

- **JavaScript/TypeScript**: ESLint security rules, Semgrep patterns
- **Python**: Bandit security analysis, Semgrep patterns  
- **Go**: golangci-lint security rules, Semgrep patterns
- **Java**: SpotBugs security rules, Semgrep patterns
- **More**: Additional languages supported through Semgrep

## 🎨 UI/UX Design

The extension follows a "quiet luxury" design philosophy inspired by Linear, Vercel, and Superhuman:

- **Clean Interface**: Minimal, focused design that doesn't distract
- **Smooth Interactions**: Subtle animations and hover states
- **Consistent Typography**: Inter font family throughout
- **Semantic Colors**: Intuitive severity color coding
- **Accessible**: High contrast and keyboard navigation support

## 🔧 Troubleshooting

### Connection Issues
- Verify server URL and API key in settings
- Check if AgentScan server is running and accessible
- Review WebSocket connection status in status bar

### Performance Issues
- Adjust `scanDebounceMs` for slower systems
- Reduce `maxConcurrentScans` if experiencing lag
- Enable caching for faster repeated scans

### Scanning Issues
- Ensure file language is in `enabledLanguages` list
- Check `severityThreshold` setting
- Verify real-time scanning is enabled

## 📊 Privacy & Telemetry

This extension collects anonymous usage data to improve performance and reliability:

- **Performance Metrics**: Scan times, cache hit rates, error rates
- **Usage Patterns**: Feature usage, command frequency
- **Error Reports**: Crash logs and error messages (sanitized)

**No sensitive data** like code content, file paths, or personal information is collected. Telemetry can be disabled in settings.

## 🐛 Known Issues

- Large files (>1MB) may experience slower scan times
- WebSocket reconnection may take up to 30 seconds in poor network conditions
- Cache invalidation may occasionally miss dependency changes

## 📝 Release Notes

### 0.1.0 - Enhanced Developer Experience

**🎉 Major Features**
- ⚡ Sub-2-second scan response times with intelligent caching
- 🎯 Rich hover tooltips with vulnerability details and fix suggestions
- 🔧 Code actions for quick fixes (suppress, ignore, learn more)
- 📊 Status bar integration with security health indicators
- ⌨️ Keyboard shortcuts for common actions
- 🌐 Enhanced WebSocket resilience with offline mode
- 📦 VS Code Marketplace packaging with telemetry

**🚀 Performance Improvements**
- Smart content-based caching system
- Concurrent scan limiting
- Debounced live scanning
- Connection quality monitoring

**💡 Developer Experience**
- Navigation between findings (F8/Shift+F8)
- Security health dashboard
- Welcome flow for new users
- Comprehensive error handling

**🔧 Technical Enhancements**
- TypeScript strict mode compliance
- Comprehensive telemetry and crash reporting
- Robust offline mode support
- Enhanced configuration management

---

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](https://github.com/NikhilSetiya/agentscan-security-scanner/blob/main/CONTRIBUTING.md) for details.

## 📄 License

This extension is licensed under the [MIT License](https://github.com/NikhilSetiya/agentscan-security-scanner/blob/main/LICENSE).

## 🔗 Links

- [GitHub Repository](https://github.com/NikhilSetiya/agentscan-security-scanner)
- [Documentation](https://github.com/NikhilSetiya/agentscan-security-scanner/tree/main/docs)
- [Issue Tracker](https://github.com/NikhilSetiya/agentscan-security-scanner/issues)
- [Changelog](https://github.com/NikhilSetiya/agentscan-security-scanner/blob/main/CHANGELOG.md)