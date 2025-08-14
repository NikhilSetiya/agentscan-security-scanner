import * as vscode from 'vscode';

export class ConfigurationManager {
    private config: vscode.WorkspaceConfiguration;

    constructor() {
        this.config = vscode.workspace.getConfiguration('agentscan');
    }

    getServerUrl(): string {
        return this.config.get<string>('serverUrl', 'http://localhost:8080');
    }

    getApiKey(): string {
        return this.config.get<string>('apiKey', '');
    }

    isRealTimeScanningEnabled(): boolean {
        return this.config.get<boolean>('enableRealTimeScanning', true);
    }

    getScanDebounceMs(): number {
        return this.config.get<number>('scanDebounceMs', 1000);
    }

    getEnabledLanguages(): string[] {
        return this.config.get<string[]>('enabledLanguages', ['javascript', 'typescript', 'python', 'go', 'java']);
    }

    getSeverityThreshold(): 'low' | 'medium' | 'high' {
        return this.config.get<'low' | 'medium' | 'high'>('severityThreshold', 'medium');
    }

    isInlineAnnotationsEnabled(): boolean {
        return this.config.get<boolean>('showInlineAnnotations', true);
    }

    isWebSocketEnabled(): boolean {
        return this.config.get<boolean>('enableWebSocket', true);
    }

    isLanguageSupported(languageId: string): boolean {
        const enabledLanguages = this.getEnabledLanguages();
        return enabledLanguages.includes(languageId);
    }

    shouldShowFinding(severity: string): boolean {
        const threshold = this.getSeverityThreshold();
        const severityLevels = { low: 1, medium: 2, high: 3 };
        
        const findingSeverity = severityLevels[severity as keyof typeof severityLevels] || 1;
        const thresholdLevel = severityLevels[threshold];
        
        return findingSeverity >= thresholdLevel;
    }

    getWebSocketUrl(): string {
        const serverUrl = this.getServerUrl();
        const wsUrl = serverUrl.replace(/^https?:\/\//, 'ws://').replace(/^http:\/\//, 'ws://').replace(/^https:\/\//, 'wss://');
        return `${wsUrl}/ws`;
    }

    get<T>(key: string, defaultValue: T): T {
        return this.config.get<T>(key, defaultValue);
    }

    async updateConfiguration(key: string, value: any, target?: vscode.ConfigurationTarget) {
        await this.config.update(key, value, target || vscode.ConfigurationTarget.Workspace);
        this.refresh();
    }

    refresh() {
        this.config = vscode.workspace.getConfiguration('agentscan');
    }

    // Validation methods
    validateConfiguration(): { isValid: boolean; errors: string[] } {
        const errors: string[] = [];

        const serverUrl = this.getServerUrl();
        if (!serverUrl || !this.isValidUrl(serverUrl)) {
            errors.push('Invalid server URL. Please provide a valid HTTP/HTTPS URL.');
        }

        const apiKey = this.getApiKey();
        if (!apiKey || apiKey.trim().length === 0) {
            errors.push('API key is required. Please configure your AgentScan API key.');
        }

        const debounceMs = this.getScanDebounceMs();
        if (debounceMs < 100 || debounceMs > 10000) {
            errors.push('Scan debounce must be between 100ms and 10000ms.');
        }

        const enabledLanguages = this.getEnabledLanguages();
        if (!Array.isArray(enabledLanguages) || enabledLanguages.length === 0) {
            errors.push('At least one language must be enabled for scanning.');
        }

        return {
            isValid: errors.length === 0,
            errors
        };
    }

    private isValidUrl(url: string): boolean {
        try {
            new URL(url);
            return true;
        } catch {
            return false;
        }
    }

    // Get configuration for display in UI
    getConfigurationSummary(): { [key: string]: any } {
        return {
            serverUrl: this.getServerUrl(),
            hasApiKey: this.getApiKey().length > 0,
            realTimeScanning: this.isRealTimeScanningEnabled(),
            debounceMs: this.getScanDebounceMs(),
            enabledLanguages: this.getEnabledLanguages(),
            severityThreshold: this.getSeverityThreshold(),
            inlineAnnotations: this.isInlineAnnotationsEnabled(),
            webSocketEnabled: this.isWebSocketEnabled()
        };
    }
}