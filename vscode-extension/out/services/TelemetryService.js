"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.TelemetryService = void 0;
class TelemetryService {
    constructor(context, enabled = true) {
        this.reporter = null;
        this.enabled = true;
        this.extensionId = 'agentscan.agentscan-security';
        this.enabled = enabled;
        this.extensionVersion = context.extension?.packageJSON?.version || '0.1.0';
        if (this.enabled) {
            this.initializeTelemetry();
        }
    }
    initializeTelemetry() {
        try {
            // Mock implementation - in production you would use actual TelemetryReporter
            this.reporter = {
                sendTelemetryEvent: (eventName, properties, measurements) => {
                    console.log('Telemetry Event:', eventName, properties, measurements);
                },
                sendTelemetryErrorEvent: (eventName, properties, measurements) => {
                    console.log('Telemetry Error:', eventName, properties, measurements);
                },
                dispose: () => {
                    console.log('Telemetry disposed');
                }
            };
        }
        catch (error) {
            console.error('Failed to initialize telemetry:', error);
            this.enabled = false;
        }
    }
    /**
     * Track scan completion metrics
     */
    trackScanCompleted(metrics) {
        if (!this.enabled || !this.reporter)
            return;
        try {
            this.reporter.sendTelemetryEvent('scan.completed', {
                scanType: metrics.scanType,
                language: metrics.language || 'unknown',
                cacheHit: metrics.cacheHit ? 'true' : 'false',
                error: metrics.error || 'none'
            }, {
                duration: metrics.duration,
                findingsCount: metrics.findingsCount,
                highSeverityCount: metrics.highSeverityCount,
                fileSize: metrics.fileSize || 0
            });
        }
        catch (error) {
            console.error('Failed to track scan metrics:', error);
        }
    }
    /**
     * Track scan failures
     */
    trackScanFailed(scanType, error, duration) {
        if (!this.enabled || !this.reporter)
            return;
        try {
            this.reporter.sendTelemetryEvent('scan.failed', {
                scanType,
                error: this.sanitizeError(error)
            }, {
                duration
            });
        }
        catch (err) {
            console.error('Failed to track scan failure:', err);
        }
    }
    /**
     * Track performance metrics
     */
    trackPerformance(metrics) {
        if (!this.enabled || !this.reporter)
            return;
        try {
            this.reporter.sendTelemetryEvent('performance', {
                operation: metrics.operation,
                success: metrics.success ? 'true' : 'false',
                error: metrics.error ? this.sanitizeError(metrics.error) : 'none'
            }, {
                duration: metrics.duration
            });
        }
        catch (error) {
            console.error('Failed to track performance metrics:', error);
        }
    }
    /**
     * Track user actions
     */
    trackUserAction(action, properties) {
        if (!this.enabled || !this.reporter)
            return;
        try {
            this.reporter.sendTelemetryEvent('user.action', {
                action,
                ...properties
            });
        }
        catch (error) {
            console.error('Failed to track user action:', error);
        }
    }
    /**
     * Track extension activation
     */
    trackActivation() {
        if (!this.enabled || !this.reporter)
            return;
        try {
            this.reporter.sendTelemetryEvent('extension.activated', {
                version: this.extensionVersion,
                platform: process.platform,
                nodeVersion: process.version
            });
        }
        catch (error) {
            console.error('Failed to track activation:', error);
        }
    }
    /**
     * Track configuration changes
     */
    trackConfigurationChange(setting, oldValue, newValue) {
        if (!this.enabled || !this.reporter)
            return;
        try {
            this.reporter.sendTelemetryEvent('configuration.changed', {
                setting,
                oldValue: String(oldValue),
                newValue: String(newValue)
            });
        }
        catch (error) {
            console.error('Failed to track configuration change:', error);
        }
    }
    /**
     * Track errors and exceptions
     */
    trackError(error, context) {
        if (!this.enabled || !this.reporter)
            return;
        try {
            this.reporter.sendTelemetryErrorEvent('error', {
                name: error.name,
                message: this.sanitizeError(error.message),
                stack: this.sanitizeStackTrace(error.stack),
                context: context || 'unknown'
            });
        }
        catch (err) {
            console.error('Failed to track error:', err);
        }
    }
    /**
     * Track WebSocket connection events
     */
    trackWebSocketEvent(event, details) {
        if (!this.enabled || !this.reporter)
            return;
        try {
            this.reporter.sendTelemetryEvent('websocket', {
                event,
                details: details || 'none'
            });
        }
        catch (error) {
            console.error('Failed to track WebSocket event:', error);
        }
    }
    /**
     * Track cache performance
     */
    trackCacheEvent(event, filePath) {
        if (!this.enabled || !this.reporter)
            return;
        try {
            this.reporter.sendTelemetryEvent('cache', {
                event,
                hasFilePath: filePath ? 'true' : 'false'
            });
        }
        catch (error) {
            console.error('Failed to track cache event:', error);
        }
    }
    /**
     * Update telemetry enabled state
     */
    setEnabled(enabled) {
        this.enabled = enabled;
        if (!enabled && this.reporter) {
            this.reporter.dispose();
            this.reporter = null;
        }
        else if (enabled && !this.reporter) {
            this.initializeTelemetry();
        }
    }
    /**
     * Sanitize error messages to remove sensitive information
     */
    sanitizeError(error) {
        // Remove file paths, URLs, and other potentially sensitive information
        return error
            .replace(/\/[^\s]+/g, '[PATH]')
            .replace(/https?:\/\/[^\s]+/g, '[URL]')
            .replace(/\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b/g, '[IP]')
            .replace(/[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}/g, '[EMAIL]')
            .substring(0, 500); // Limit length
    }
    /**
     * Sanitize stack traces
     */
    sanitizeStackTrace(stack) {
        if (!stack)
            return '';
        return stack
            .split('\n')
            .slice(0, 10) // Limit to first 10 lines
            .map(line => this.sanitizeError(line))
            .join('\n');
    }
    dispose() {
        if (this.reporter) {
            this.reporter.dispose();
            this.reporter = null;
        }
    }
}
exports.TelemetryService = TelemetryService;
//# sourceMappingURL=TelemetryService.js.map