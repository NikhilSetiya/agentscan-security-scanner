import * as vscode from 'vscode';
import * as path from 'path';
import { ApiClient, Finding, ScanResult } from '../services/ApiClient';
import { DiagnosticsManager } from '../services/DiagnosticsManager';
import { ConfigurationManager } from '../services/ConfigurationManager';

export class AgentScanProvider {
    private apiClient: ApiClient;
    private diagnosticsManager: DiagnosticsManager;
    private config: ConfigurationManager;
    private activeScanJobs: Map<string, string> = new Map(); // file path -> scan job ID
    private scanDebounceTimers: Map<string, NodeJS.Timeout> = new Map();

    constructor(
        apiClient: ApiClient,
        diagnosticsManager: DiagnosticsManager,
        config: ConfigurationManager
    ) {
        this.apiClient = apiClient;
        this.diagnosticsManager = diagnosticsManager;
        this.config = config;
    }

    async scanFile(document: vscode.TextDocument, isLiveMode: boolean = false): Promise<void> {
        // Check if language is supported
        if (!this.config.isLanguageSupported(document.languageId)) {
            return;
        }

        // Validate configuration
        const validation = this.config.validateConfiguration();
        if (!validation.isValid) {
            vscode.window.showErrorMessage(
                `AgentScan configuration error: ${validation.errors.join(', ')}`
            );
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

    private async performFileScan(
        document: vscode.TextDocument, 
        content: string, 
        isLiveMode: boolean
    ): Promise<void> {
        const filePath = document.fileName;
        const relativePath = this.getRelativePath(filePath);

        try {
            // Cancel existing scan for this file
            const existingScanId = this.activeScanJobs.get(filePath);
            if (existingScanId) {
                // In a real implementation, you would cancel the existing scan
                console.log(`Cancelling existing scan ${existingScanId} for ${relativePath}`);
            }

            // Show progress
            if (!isLiveMode) {
                vscode.window.withProgress({
                    location: vscode.ProgressLocation.Notification,
                    title: `Scanning ${path.basename(filePath)}...`,
                    cancellable: true
                }, async (progress, token) => {
                    return this.executeScan(document, content, progress, token);
                });
            } else {
                // Silent scan for live mode
                await this.executeScan(document, content);
            }

        } catch (error) {
            console.error('File scan failed:', error);
            if (!isLiveMode) {
                vscode.window.showErrorMessage(`AgentScan: Failed to scan file - ${error}`);
            }
        }
    }

    private async executeScan(
        document: vscode.TextDocument,
        content: string,
        progress?: vscode.Progress<{ message?: string; increment?: number }>,
        token?: vscode.CancellationToken
    ): Promise<void> {
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
                
                progress?.report({ message: 'Scan completed', increment: 100 });
                break;
                
            } else if (status.status === 'failed') {
                throw new Error(status.errorMessage || 'Scan failed');
                
            } else if (status.status === 'cancelled') {
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

    async scanWorkspace(): Promise<void> {
        const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
        if (!workspaceFolder) {
            vscode.window.showWarningMessage('No workspace folder open');
            return;
        }

        // Validate configuration
        const validation = this.config.validateConfiguration();
        if (!validation.isValid) {
            vscode.window.showErrorMessage(
                `AgentScan configuration error: ${validation.errors.join(', ')}`
            );
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
                        
                    } else if (status.status === 'failed') {
                        throw new Error(status.errorMessage || 'Workspace scan failed');
                        
                    } else if (status.status === 'cancelled') {
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

        } catch (error) {
            console.error('Workspace scan failed:', error);
            vscode.window.showErrorMessage(`AgentScan: Workspace scan failed - ${error}`);
        }
    }

    async suppressFinding(finding: Finding): Promise<void> {
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

        } catch (error) {
            console.error('Failed to suppress finding:', error);
            vscode.window.showErrorMessage(`Failed to suppress finding: ${error}`);
        }
    }

    private filterFindings(findings: Finding[]): Finding[] {
        return findings.filter(finding => this.config.shouldShowFinding(finding.severity));
    }

    private groupFindingsByFile(findings: Finding[]): Map<string, Finding[]> {
        const grouped = new Map<string, Finding[]>();
        
        for (const finding of findings) {
            const filePath = path.resolve(finding.filePath);
            if (!grouped.has(filePath)) {
                grouped.set(filePath, []);
            }
            grouped.get(filePath)!.push(finding);
        }
        
        return grouped;
    }

    private showWorkspaceScanSummary(findings: Finding[]) {
        const total = findings.length;
        const high = findings.filter(f => f.severity === 'high').length;
        const medium = findings.filter(f => f.severity === 'medium').length;
        const low = findings.filter(f => f.severity === 'low').length;

        if (total === 0) {
            vscode.window.showInformationMessage('🎉 No security issues found in workspace!');
        } else {
            const message = `Found ${total} security issues: ${high} high, ${medium} medium, ${low} low severity`;
            if (high > 0) {
                vscode.window.showWarningMessage(message);
            } else {
                vscode.window.showInformationMessage(message);
            }
        }
    }

    private getRelativePath(filePath: string): string {
        const workspaceFolder = vscode.workspace.workspaceFolders?.[0];
        if (workspaceFolder) {
            return path.relative(workspaceFolder.uri.fsPath, filePath);
        }
        return path.basename(filePath);
    }

    updateConfiguration(config: ConfigurationManager) {
        this.config = config;
        this.apiClient.updateConfiguration(config);
    }

    dispose() {
        // Clear all debounce timers
        this.scanDebounceTimers.forEach(timer => clearTimeout(timer));
        this.scanDebounceTimers.clear();
        this.activeScanJobs.clear();
    }
}