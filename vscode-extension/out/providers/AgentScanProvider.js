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
exports.AgentScanProvider = void 0;
const vscode = __importStar(require("vscode"));
const path = __importStar(require("path"));
class AgentScanProvider {
    constructor(apiClient, diagnosticsManager, config, cacheManager, telemetryService) {
        this.activeScanJobs = new Map(); // file path -> scan job ID
        this.scanDebounceTimers = new Map();
        this.concurrentScans = 0;
        this.maxConcurrentScans = 3;
        this.apiClient = apiClient;
        this.diagnosticsManager = diagnosticsManager;
        this.config = config;
        this.cacheManager = cacheManager;
        this.telemetryService = telemetryService;
        this.maxConcurrentScans = config.get('maxConcurrentScans', 3);
    }
    async scanFile(document, isLiveMode = false) {
        // Check if language is supported
        if (!this.config.isLanguageSupported(document.languageId)) {
            return;
        }
        // Validate configuration
        const validation = this.config.validateConfiguration();
        if (!validation.isValid) {
            vscode.window.showErrorMessage(`AgentScan configuration error: ${validation.errors.join(', ')}`);
            return;
        }
        const filePath = document.fileName;
        const content = document.getText();
        // Clear existing debounce timer
        const existingTimer = this.scanDebounceTimers.get(filePath);
        if (existingTimer) {
            clearTimeout(existingTimer);
        }
        // Set up debounced scan
        const debounceMs = isLiveMode ? 500 : this.config.getScanDebounceMs();
        const timer = setTimeout(async () => {
            await this.performFileScan(document, content, isLiveMode);
            this.scanDebounceTimers.delete(filePath);
        }, debounceMs);
        this.scanDebounceTimers.set(filePath, timer);
    }
    async performFileScan(document, content, isLiveMode) {
        const filePath = document.fileName;
        const relativePath = this.getRelativePath(filePath);
        const startTime = Date.now();
        // Check concurrent scan limit
        if (this.concurrentScans >= this.maxConcurrentScans) {
            if (!isLiveMode) {
                vscode.window.showWarningMessage('AgentScan: Too many concurrent scans. Please wait...');
            }
            return;
        }
        try {
            // Check cache first
            const cachedFindings = this.cacheManager.getCachedFindings(filePath, content);
            if (cachedFindings && this.config.get('cacheEnabled', true)) {
                this.diagnosticsManager.updateFindings(document.uri, cachedFindings);
                this.telemetryService.trackCacheEvent('hit', filePath);
                this.telemetryService.trackScanCompleted({
                    scanType: 'file',
                    duration: Date.now() - startTime,
                    findingsCount: cachedFindings.length,
                    highSeverityCount: cachedFindings.filter(f => f.severity === 'high').length,
                    language: document.languageId,
                    fileSize: content.length,
                    cacheHit: true
                });
                return;
            }
            this.telemetryService.trackCacheEvent('miss', filePath);
            // Cancel existing scan for this file
            const existingScanId = this.activeScanJobs.get(filePath);
            if (existingScanId) {
                console.log(`Cancelling existing scan ${existingScanId} for ${relativePath}`);
            }
            this.concurrentScans++;
            // Show progress
            if (!isLiveMode) {
                await vscode.window.withProgress({
                    location: vscode.ProgressLocation.Notification,
                    title: `Scanning ${path.basename(filePath)}...`,
                    cancellable: true
                }, async (progress, token) => {
                    return this.executeScan(document, content, progress, token, startTime);
                });
            }
            else {
                // Silent scan for live mode
                await this.executeScan(document, content, undefined, undefined, startTime);
            }
        }
        catch (error) {
            console.error('File scan failed:', error);
            this.telemetryService.trackScanFailed('file', error instanceof Error ? error.message : String(error), Date.now() - startTime);
            if (!isLiveMode) {
                vscode.window.showErrorMessage(`AgentScan: Failed to scan file - ${error}`);
            }
        }
        finally {
            this.concurrentScans--;
        }
    }
    async executeScan(document, content, progress, token, startTime) {
        const filePath = document.fileName;
        const relativePath = this.getRelativePath(filePath);
        progress?.report({ message: 'Starting scan...', increment: 10 });
        // Start scan
        const scanResult = await this.apiClient.scanFile(relativePath, content);
        this.activeScanJobs.set(filePath, scanResult.id);
        progress?.report({ message: 'Scan in progress...', increment: 30 });
        // Poll for results
        let attempts = 0;
        const maxAttempts = 60; // 60 seconds timeout
        while (attempts < maxAttempts) {
            if (token?.isCancellationRequested) {
                return;
            }
            const status = await this.apiClient.getScanStatus(scanResult.id);
            if (status.status === 'completed') {
                progress?.report({ message: 'Processing results...', increment: 50 });
                const findings = await this.apiClient.getScanResults(scanResult.id);
                const filteredFindings = this.filterFindings(findings);
                // Update diagnostics
                this.diagnosticsManager.updateFindings(document.uri, filteredFindings);
                // Cache the results
                if (this.config.get('cacheEnabled', true)) {
                    this.cacheManager.cacheFindings(filePath, content, filteredFindings);
                }
                // Track scan completion
                if (startTime) {
                    this.telemetryService.trackScanCompleted({
                        scanType: 'file',
                        duration: Date.now() - startTime,
                        findingsCount: filteredFindings.length,
                        highSeverityCount: filteredFindings.filter(f => f.severity === 'high').length,
                        language: document.languageId,
                        fileSize: content.length,
                        cacheHit: false
                    });
                }
                progress?.report({ message: 'Scan completed', increment: 100 });
                break;
            }
            else if (status.status === 'failed') {
                throw new Error(status.errorMessage || 'Scan failed');
            }
            else if (status.status === 'cancelled') {
                return;
            }
            // Wait before next poll
            await new Promise(resolve => setTimeout(resolve, 1000));
            attempts++;
            progress?.report({
                message: `Scanning... (${Math.min(90, 30 + attempts * 2)}%)`,
                increment: 1
            });
        }
        if (attempts >= maxAttempts) {
            throw new Error('Scan timeout - please try again');
        }
        this.activeScanJobs.delete(filePath);
    }
    async scanWorkspace() {
        const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
        if (!workspaceFolder) {
            vscode.window.showWarningMessage('No workspace folder open');
            return;
        }
        // Validate configuration
        const validation = this.config.validateConfiguration();
        if (!validation.isValid) {
            vscode.window.showErrorMessage(`AgentScan configuration error: ${validation.errors.join(', ')}`);
            return;
        }
        try {
            await vscode.window.withProgress({
                location: vscode.ProgressLocation.Notification,
                title: 'Scanning workspace...',
                cancellable: true
            }, async (progress, token) => {
                progress.report({ message: 'Starting workspace scan...', increment: 10 });
                const scanResult = await this.apiClient.scanWorkspace(workspaceFolder.uri.fsPath);
                progress.report({ message: 'Scan in progress...', increment: 30 });
                // Poll for results
                let attempts = 0;
                const maxAttempts = 300; // 5 minutes timeout for workspace scans
                while (attempts < maxAttempts) {
                    if (token.isCancellationRequested) {
                        return;
                    }
                    const status = await this.apiClient.getScanStatus(scanResult.id);
                    if (status.status === 'completed') {
                        progress.report({ message: 'Processing results...', increment: 80 });
                        const findings = await this.apiClient.getScanResults(scanResult.id);
                        const filteredFindings = this.filterFindings(findings);
                        // Group findings by file
                        const findingsByFile = this.groupFindingsByFile(filteredFindings);
                        // Update diagnostics for each file
                        for (const [filePath, fileFindings] of findingsByFile.entries()) {
                            const uri = vscode.Uri.file(filePath);
                            this.diagnosticsManager.updateFindings(uri, fileFindings);
                        }
                        progress.report({ message: 'Workspace scan completed', increment: 100 });
                        // Show summary
                        this.showWorkspaceScanSummary(filteredFindings);
                        break;
                    }
                    else if (status.status === 'failed') {
                        throw new Error(status.errorMessage || 'Workspace scan failed');
                    }
                    else if (status.status === 'cancelled') {
                        return;
                    }
                    // Wait before next poll
                    await new Promise(resolve => setTimeout(resolve, 2000));
                    attempts++;
                    progress.report({
                        message: `Scanning workspace... (${Math.min(70, 30 + attempts)}%)`,
                        increment: 0.5
                    });
                }
                if (attempts >= maxAttempts) {
                    throw new Error('Workspace scan timeout - please try again');
                }
            });
        }
        catch (error) {
            console.error('Workspace scan failed:', error);
            vscode.window.showErrorMessage(`AgentScan: Workspace scan failed - ${error}`);
        }
    }
    async suppressFinding(finding) {
        try {
            const reason = await vscode.window.showInputBox({
                prompt: 'Enter reason for suppressing this finding',
                placeHolder: 'e.g., False positive - this code is safe',
                validateInput: (value) => {
                    if (!value || value.trim().length === 0) {
                        return 'Reason is required';
                    }
                    if (value.length > 500) {
                        return 'Reason must be less than 500 characters';
                    }
                    return null;
                }
            });
            if (!reason) {
                return; // User cancelled
            }
            // Ask for expiration
            const expirationOptions = [
                { label: 'Never expires', value: null },
                { label: '30 days', value: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000) },
                { label: '90 days', value: new Date(Date.now() + 90 * 24 * 60 * 60 * 1000) },
                { label: '1 year', value: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000) }
            ];
            const selectedExpiration = await vscode.window.showQuickPick(expirationOptions, {
                placeHolder: 'Select expiration for this suppression'
            });
            if (!selectedExpiration) {
                return; // User cancelled
            }
            await this.apiClient.suppressFinding(finding.id, {
                reason: reason.trim(),
                expiresAt: selectedExpiration.value?.toISOString()
            });
            // Remove the finding from diagnostics
            const activeEditor = vscode.window.activeTextEditor;
            if (activeEditor) {
                const currentFindings = this.diagnosticsManager.getFindingsForFile(activeEditor.document.uri);
                const updatedFindings = currentFindings.filter(f => f.id !== finding.id);
                this.diagnosticsManager.updateFindings(activeEditor.document.uri, updatedFindings);
            }
            vscode.window.showInformationMessage('Finding suppressed successfully');
        }
        catch (error) {
            console.error('Failed to suppress finding:', error);
            vscode.window.showErrorMessage(`Failed to suppress finding: ${error}`);
        }
    }
    filterFindings(findings) {
        return findings.filter(finding => this.config.shouldShowFinding(finding.severity));
    }
    groupFindingsByFile(findings) {
        const grouped = new Map();
        for (const finding of findings) {
            const filePath = path.resolve(finding.filePath);
            if (!grouped.has(filePath)) {
                grouped.set(filePath, []);
            }
            grouped.get(filePath).push(finding);
        }
        return grouped;
    }
    showWorkspaceScanSummary(findings) {
        const total = findings.length;
        const high = findings.filter(f => f.severity === 'high').length;
        const medium = findings.filter(f => f.severity === 'medium').length;
        const low = findings.filter(f => f.severity === 'low').length;
        if (total === 0) {
            vscode.window.showInformationMessage('🎉 No security issues found in workspace!');
        }
        else {
            const message = `Found ${total} security issues: ${high} high, ${medium} medium, ${low} low severity`;
            if (high > 0) {
                vscode.window.showWarningMessage(message);
            }
            else {
                vscode.window.showInformationMessage(message);
            }
        }
    }
    getRelativePath(filePath) {
        const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
        if (workspaceFolder) {
            return path.relative(workspaceFolder.uri.fsPath, filePath);
        }
        return path.basename(filePath);
    }
    async markAsFixed(finding) {
        try {
            await this.apiClient.updateFindingStatus(finding.id, 'fixed', 'Marked as fixed from VS Code');
            // Remove the finding from diagnostics
            const activeEditor = vscode.window.activeTextEditor;
            if (activeEditor) {
                const currentFindings = this.diagnosticsManager.getFindingsForFile(activeEditor.document.uri);
                const updatedFindings = currentFindings.filter(f => f.id !== finding.id);
                this.diagnosticsManager.updateFindings(activeEditor.document.uri, updatedFindings);
            }
            vscode.window.showInformationMessage('Finding marked as fixed');
        }
        catch (error) {
            console.error('Failed to mark finding as fixed:', error);
            vscode.window.showErrorMessage(`Failed to mark finding as fixed: ${error}`);
        }
    }
    async ignoreRule(finding) {
        try {
            const confirmation = await vscode.window.showWarningMessage(`This will ignore all future findings for rule "${finding.ruleId}". Are you sure?`, { modal: true }, 'Yes, Ignore Rule', 'Cancel');
            if (confirmation !== 'Yes, Ignore Rule') {
                return;
            }
            // In a real implementation, you would add the rule to an ignore list
            // For now, we'll just suppress all findings with this rule ID
            const activeEditor = vscode.window.activeTextEditor;
            if (activeEditor) {
                const currentFindings = this.diagnosticsManager.getFindingsForFile(activeEditor.document.uri);
                const updatedFindings = currentFindings.filter(f => f.ruleId !== finding.ruleId);
                this.diagnosticsManager.updateFindings(activeEditor.document.uri, updatedFindings);
            }
            vscode.window.showInformationMessage(`Rule "${finding.ruleId}" has been ignored`);
        }
        catch (error) {
            console.error('Failed to ignore rule:', error);
            vscode.window.showErrorMessage(`Failed to ignore rule: ${error}`);
        }
    }
    updateConfiguration(config) {
        this.config = config;
        this.apiClient.updateConfiguration(config);
        this.maxConcurrentScans = config.get('maxConcurrentScans', 3);
    }
    dispose() {
        // Clear all debounce timers
        this.scanDebounceTimers.forEach(timer => clearTimeout(timer));
        this.scanDebounceTimers.clear();
        this.activeScanJobs.clear();
    }
}
exports.AgentScanProvider = AgentScanProvider;
//# sourceMappingURL=AgentScanProvider.js.map