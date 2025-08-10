# Change Log

All notable changes to the "AgentScan Security" extension will be documented in this file.

## [0.1.0] - 2024-01-07

### Added
- Initial release of AgentScan Security VS Code extension
- Real-time security scanning with debounced triggers on file save
- Multi-agent consensus engine integration
- Clean, minimal UI design following quiet luxury aesthetic
- Inline security annotations with hover tooltips
- WebSocket connection for real-time updates from AgentScan server
- Finding suppression system for false positive management
- Configurable severity thresholds and language support
- Status bar integration showing connection and scan status
- Tree view for organized findings display
- Support for JavaScript, TypeScript, Python, Go, and Java
- Export capabilities for PDF and JSON reports
- Comprehensive test suite
- VS Code marketplace packaging

### Features
- **Real-time Scanning**: Automatic scanning on file save with configurable debounce
- **Live Feedback**: WebSocket integration for instant scan progress and results
- **Visual Indicators**: Severity-based color coding (🔴 High, 🟡 Medium, 🔵 Low)
- **Smart Filtering**: Configurable severity thresholds to reduce noise
- **Finding Management**: Suppress false positives and mark findings as fixed
- **Multi-language Support**: Comprehensive coverage for popular programming languages
- **Clean Design**: Minimal, distraction-free interface inspired by modern dev tools

### Commands
- `agentscan.scanFile` - Scan current file for security issues
- `agentscan.scanWorkspace` - Scan entire workspace
- `agentscan.clearFindings` - Clear all security findings
- `agentscan.showSettings` - Open extension settings
- `agentscan.suppressFinding` - Suppress false positive findings

### Configuration
- Server URL and API key configuration
- Real-time scanning toggle
- Debounce timing configuration
- Language selection
- Severity threshold settings
- UI preferences (inline annotations, WebSocket)

### Technical Implementation
- TypeScript-based extension architecture
- Modular service-oriented design
- Comprehensive error handling and user feedback
- Efficient debouncing and caching mechanisms
- WebSocket client with automatic reconnection
- Diagnostic collection integration with VS Code
- Custom decoration types for visual indicators

## [Unreleased]

### Planned Features
- Code action providers for automatic fixes
- Integration with VS Code's built-in source control
- Batch finding management operations
- Custom rule configuration
- Team collaboration features
- Performance optimizations for large codebases