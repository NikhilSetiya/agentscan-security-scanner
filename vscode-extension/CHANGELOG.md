# Change Log

All notable changes to the AgentScan Security extension will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2024-01-15

### 🎉 Added - Enhanced Developer Experience

#### Real-time Performance
- **Sub-2-second scanning**: Intelligent caching system for lightning-fast results
- **Smart content hashing**: Avoid redundant scans with SHA-256 content fingerprinting
- **Concurrent scan limiting**: Configurable limits to prevent system overload
- **Performance telemetry**: Track scan times, cache hit rates, and system performance

#### Rich Developer Interface
- **Enhanced hover tooltips**: Detailed vulnerability information with severity badges
- **Code actions provider**: Quick fixes for suppress, mark fixed, ignore rule, and learn more
- **Keyboard navigation**: F8/Shift+F8 to navigate between findings
- **Status bar integration**: Real-time security health and scan progress indicators

#### Advanced WebSocket Features
- **Connection resilience**: Automatic reconnection with exponential backoff
- **Offline mode support**: Continue working when server is unavailable
- **Connection quality monitoring**: Track and display connection health metrics
- **Message queuing**: Reliable message delivery with automatic retry

#### Security Health Dashboard
- **Health score calculation**: Visual security posture assessment
- **Statistics overview**: Breakdown by severity, tools, and file types
- **Cache performance**: Monitor cache efficiency and storage usage
- **Connection diagnostics**: Real-time connection status and quality metrics

#### Developer Experience Enhancements
- **Welcome flow**: First-time user onboarding with configuration guidance
- **Comprehensive error handling**: Graceful degradation and user-friendly error messages
- **Telemetry and crash reporting**: Anonymous usage analytics for reliability improvements
- **Configuration validation**: Real-time validation of server URLs and API keys

### 🚀 Improved - Core Functionality

#### Scanning Engine
- **Debounced live scanning**: Configurable delays for optimal performance
- **Language detection**: Automatic file type recognition and appropriate tool selection
- **Incremental updates**: Only scan changed content for faster feedback
- **Error recovery**: Robust handling of scan failures and timeouts

#### User Interface
- **Quiet luxury design**: Clean, minimal interface inspired by Linear and Vercel
- **Semantic color coding**: Intuitive severity-based visual hierarchy
- **Smooth animations**: Subtle hover states and transitions
- **Accessibility improvements**: High contrast support and keyboard navigation

#### Configuration Management
- **Extended settings**: New options for caching, performance, and telemetry
- **Dynamic updates**: Live configuration changes without restart
- **Validation feedback**: Real-time validation with helpful error messages
- **Workspace-specific settings**: Per-project configuration support

### 🔧 Technical Improvements

#### Architecture
- **TypeScript strict mode**: Enhanced type safety and code quality
- **Modular service architecture**: Separation of concerns with dedicated service classes
- **Event-driven design**: Reactive updates and efficient resource management
- **Memory optimization**: Proper cleanup and disposal patterns

#### Performance
- **Caching layer**: Content-based caching with configurable expiration
- **Connection pooling**: Efficient WebSocket connection management
- **Resource monitoring**: Track memory usage and performance metrics
- **Lazy loading**: On-demand initialization of heavy components

#### Reliability
- **Comprehensive error handling**: Graceful failure modes and recovery
- **Telemetry integration**: Anonymous crash reporting and usage analytics
- **Health monitoring**: System health checks and diagnostics
- **Offline resilience**: Robust offline mode with automatic reconnection

### 📋 Commands Added

#### Navigation
- `agentscan.nextFinding` - Navigate to next security finding (F8)
- `agentscan.previousFinding` - Navigate to previous security finding (Shift+F8)

#### Quick Actions
- `agentscan.markAsFixed` - Mark finding as resolved
- `agentscan.ignoreRule` - Disable specific security rule
- `agentscan.learnMore` - Open detailed vulnerability documentation

#### Utilities
- `agentscan.toggleRealTimeScanning` - Toggle live scanning (Ctrl+Shift+T)
- `agentscan.showSecurityHealth` - Open security health dashboard

### ⚙️ Settings Added

#### Performance
- `agentscan.cacheEnabled` - Enable intelligent result caching (default: true)
- `agentscan.cacheMaxAge` - Cache expiration time in seconds (default: 300)
- `agentscan.maxConcurrentScans` - Maximum concurrent file scans (default: 3)

#### Features
- `agentscan.enableTelemetry` - Enable usage analytics and crash reporting (default: true)
- `agentscan.showSecurityHealth` - Show security health in status bar (default: true)

### 🎮 Keyboard Shortcuts Added

- `Ctrl+Shift+S` (`Cmd+Shift+S`) - Scan Current File
- `F8` - Go to Next Finding
- `Shift+F8` - Go to Previous Finding
- `Ctrl+Shift+I` (`Cmd+Shift+I`) - Suppress Finding
- `Ctrl+Shift+T` (`Cmd+Shift+T`) - Toggle Real-time Scanning

### 🐛 Fixed

#### Stability
- Fixed memory leaks in WebSocket connection handling
- Resolved race conditions in concurrent scanning
- Improved error handling for malformed server responses
- Fixed cache invalidation edge cases

#### User Interface
- Corrected hover tooltip positioning and content
- Fixed status bar update timing issues
- Resolved decoration flickering during rapid edits
- Improved responsiveness of real-time scanning

#### Performance
- Optimized scan debouncing for better responsiveness
- Reduced memory usage in large workspaces
- Improved startup time with lazy initialization
- Fixed cache size management and cleanup

### 🔒 Security

#### Data Protection
- Sanitized error messages to prevent information leakage
- Implemented secure credential storage
- Added input validation for all user-provided data
- Enhanced API key handling and storage

#### Privacy
- Anonymous telemetry with no sensitive data collection
- Configurable telemetry with easy opt-out
- Local caching with secure content hashing
- No code content transmitted in telemetry

### 📚 Documentation

#### User Guides
- Comprehensive README with feature overview
- Detailed configuration guide
- Troubleshooting section with common issues
- Keyboard shortcuts reference

#### Developer Documentation
- API documentation for extension points
- Architecture overview and design decisions
- Contributing guidelines and development setup
- Code examples and best practices

---

## [Unreleased]

### Planned Features
- **Multi-workspace support**: Scan across multiple workspace folders
- **Custom rule configuration**: User-defined security rules and patterns
- **Integration plugins**: Support for additional security tools
- **Team collaboration**: Shared findings and suppression lists
- **Advanced reporting**: Detailed security reports and trend analysis

### Known Issues
- Large files (>1MB) may experience slower scan times
- WebSocket reconnection may take up to 30 seconds in poor network conditions
- Cache invalidation may occasionally miss dependency changes

---

## Version History

- **0.1.0** - Enhanced Developer Experience (Current)
- **0.0.1** - Initial prototype release (Internal)

For more details about each release, see the [GitHub Releases](https://github.com/NikhilSetiya/agentscan-security-scanner/releases) page.