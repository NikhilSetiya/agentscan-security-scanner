import * as vscode from 'vscode';

// Mock TelemetryReporter for now - in production you would install @vscode/extension-telemetry
interface TelemetryReporter {
    sendTelemetryEvent(eventName: string, properties?: { [key: string]: string }, measurements?: { [key: string]: number }): void;
    sendTelemetryErrorEvent(eventName: string, properties?: { [key: string]: string }, measurements?: { [key: string]: number }): void;
    dispose(): void;
}

interface ScanMetrics {
    scanType: 'file' | 'workspace';
    duration: number;
    findingsCount: number;
    highSeverityCount: number;
    language?: string;
    fileSize?: number;
    cacheHit?: boolean;
    error?: string;
}

interface PerformanceMetrics {
    operation: string;
    duration: number;
    success: boolean;
    error?: string;
}

export class TelemetryService {
    private reporter: TelemetryReporter | null = null;
    private enabled: boolean = true;
    private readonly extensionId = 'agentscan.agentscan-security';
    private readonly extensionVersion: string;

    constructor(context: vscode.ExtensionContext, enabled: boolean = true) {
        this.enabled = enabled;
        this.extensionVersion = context.extension?.packageJSON?.version || '0.1.0';
        
        if (this.enabled) {
            this.initializeTelemetry();
        }
    }

    private initializeTelemetry(): void {
        try {
            // Mock implementation - in production you would use actual TelemetryReporter
            this.reporter = {
                sendTelemetryEvent: (eventName: string, properties?: { [key: string]: string }, measurements?: { [key: string]: number }) => {
                    console.log('Telemetry Event:', eventName, properties, measurements);
                },
                sendTelemetryErrorEvent: (eventName: string, properties?: { [key: string]: string }, measurements?: { [key: string]: number }) => {
                    console.log('Telemetry Error:', eventName, properties, measurements);
                },
                dispose: () => {
                    console.log('Telemetry disposed');
                }
            };
        } catch (error) {
            console.error('Failed to initialize telemetry:', error);
            this.enabled = false;
        }
    }

    /**
     * Track scan completion metrics
     */
    trackScanCompleted(metrics: ScanMetrics): void {
        if (!this.enabled || !this.reporter) return;

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
        } catch (error) {
            console.error('Failed to track scan metrics:', error);
        }
    }

    /**
     * Track scan failures
     */
    trackScanFailed(scanType: 'file' | 'workspace', error: string, duration: number): void {
        if (!this.enabled || !this.reporter) return;

        try {
            this.reporter.sendTelemetryEvent('scan.failed', {
                scanType,
                error: this.sanitizeError(error)
            }, {
                duration
            });
        } catch (err) {
            console.error('Failed to track scan failure:', err);
        }
    }

    /**
     * Track performance metrics
     */
    trackPerformance(metrics: PerformanceMetrics): void {
        if (!this.enabled || !this.reporter) return;

        try {
            this.reporter.sendTelemetryEvent('performance', {
                operation: metrics.operation,
                success: metrics.success ? 'true' : 'false',
                error: metrics.error ? this.sanitizeError(metrics.error) : 'none'
            }, {
                duration: metrics.duration
            });
        } catch (error) {
            console.error('Failed to track performance metrics:', error);
        }
    }

    /**
     * Track user actions
     */
    trackUserAction(action: string, properties?: { [key: string]: string }): void {
        if (!this.enabled || !this.reporter) return;

        try {
            this.reporter.sendTelemetryEvent('user.action', {
                action,
                ...properties
            });
        } catch (error) {
            console.error('Failed to track user action:', error);
        }
    }

    /**
     * Track extension activation
     */
    trackActivation(): void {
        if (!this.enabled || !this.reporter) return;

        try {
            this.reporter.sendTelemetryEvent('extension.activated', {
                version: this.extensionVersion,
                platform: process.platform,
                nodeVersion: process.version
            });
        } catch (error) {
            console.error('Failed to track activation:', error);
        }
    }

    /**
     * Track configuration changes
     */
    trackConfigurationChange(setting: string, oldValue: any, newValue: any): void {
        if (!this.enabled || !this.reporter) return;

        try {
            this.reporter.sendTelemetryEvent('configuration.changed', {
                setting,
                oldValue: String(oldValue),
                newValue: String(newValue)
            });
        } catch (error) {
            console.error('Failed to track configuration change:', error);
        }
    }

    /**
     * Track errors and exceptions
     */
    trackError(error: Error, context?: string): void {
        if (!this.enabled || !this.reporter) return;

        try {
            this.reporter.sendTelemetryErrorEvent('error', {
                name: error.name,
                message: this.sanitizeError(error.message),
                stack: this.sanitizeStackTrace(error.stack),
                context: context || 'unknown'
            });
        } catch (err) {
            console.error('Failed to track error:', err);
        }
    }

    /**
     * Track WebSocket connection events
     */
    trackWebSocketEvent(event: 'connected' | 'disconnected' | 'error' | 'reconnected', details?: string): void {
        if (!this.enabled || !this.reporter) return;

        try {
            this.reporter.sendTelemetryEvent('websocket', {
                event,
                details: details || 'none'
            });
        } catch (error) {
            console.error('Failed to track WebSocket event:', error);
        }
    }

    /**
     * Track cache performance
     */
    trackCacheEvent(event: 'hit' | 'miss' | 'invalidated', filePath?: string): void {
        if (!this.enabled || !this.reporter) return;

        try {
            this.reporter.sendTelemetryEvent('cache', {
                event,
                hasFilePath: filePath ? 'true' : 'false'
            });
        } catch (error) {
            console.error('Failed to track cache event:', error);
        }
    }

    /**
     * Update telemetry enabled state
     */
    setEnabled(enabled: boolean): void {
        this.enabled = enabled;
        
        if (!enabled && this.reporter) {
            this.reporter.dispose();
            this.reporter = null;
        } else if (enabled && !this.reporter) {
            this.initializeTelemetry();
        }
    }

    /**
     * Sanitize error messages to remove sensitive information
     */
    private sanitizeError(error: string): string {
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
    private sanitizeStackTrace(stack?: string): string {
        if (!stack) return '';
        
        return stack
            .split('\n')
            .slice(0, 10) // Limit to first 10 lines
            .map(line => this.sanitizeError(line))
            .join('\n');
    }

    dispose(): void {
        if (this.reporter) {
            this.reporter.dispose();
            this.reporter = null;
        }
    }
}