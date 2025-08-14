"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.ConfigurationManager = void 0;
const vscode = __importStar(require("vscode"));
class ConfigurationManager {
    constructor() {
        this.config = vscode.workspace.getConfiguration('agentscan');
    }
    getServerUrl() {
        return this.config.get('serverUrl', 'http://localhost:8080');
    }
    getApiKey() {
        return this.config.get('apiKey', '');
    }
    isRealTimeScanningEnabled() {
        return this.config.get('enableRealTimeScanning', true);
    }
    getScanDebounceMs() {
        return this.config.get('scanDebounceMs', 1000);
    }
    getEnabledLanguages() {
        return this.config.get('enabledLanguages', ['javascript', 'typescript', 'python', 'go', 'java']);
    }
    getSeverityThreshold() {
        return this.config.get('severityThreshold', 'medium');
    }
    isInlineAnnotationsEnabled() {
        return this.config.get('showInlineAnnotations', true);
    }
    isWebSocketEnabled() {
        return this.config.get('enableWebSocket', true);
    }
    isLanguageSupported(languageId) {
        const enabledLanguages = this.getEnabledLanguages();
        return enabledLanguages.includes(languageId);
    }
    shouldShowFinding(severity) {
        const threshold = this.getSeverityThreshold();
        const severityLevels = { low: 1, medium: 2, high: 3 };
        const findingSeverity = severityLevels[severity] || 1;
        const thresholdLevel = severityLevels[threshold];
        return findingSeverity >= thresholdLevel;
    }
    getWebSocketUrl() {
        const serverUrl = this.getServerUrl();
        const wsUrl = serverUrl.replace(/^https?:\/\//, 'ws://').replace(/^http:\/\//, 'ws://').replace(/^https:\/\//, 'wss://');
        return `${wsUrl}/ws`;
    }
    get(key, defaultValue) {
        return this.config.get(key, defaultValue);
    }
    async updateConfiguration(key, value, target) {
        await this.config.update(key, value, target || vscode.ConfigurationTarget.Workspace);
        this.refresh();
    }
    refresh() {
        this.config = vscode.workspace.getConfiguration('agentscan');
    }
    // Validation methods
    validateConfiguration() {
        const errors = [];
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
    isValidUrl(url) {
        try {
            new URL(url);
            return true;
        }
        catch {
            return false;
        }
    }
    // Get configuration for display in UI
    getConfigurationSummary() {
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
exports.ConfigurationManager = ConfigurationManager;
//# sourceMappingURL=ConfigurationManager.js.map