import * as vscode from 'vscode';
import WebSocket from 'ws';
import { ConfigurationManager } from './ConfigurationManager';
import { DiagnosticsManager } from './DiagnosticsManager';
import { Finding } from './ApiClient';

interface WebSocketMessage {
    type: 'scan_started' | 'scan_progress' | 'scan_completed' | 'scan_failed' | 'finding_updated';
    data: any;
}

interface ScanProgressData {
    scanId: string;
    progress: number;
    currentAgent?: string;
    message?: string;
}

interface ScanCompletedData {
    scanId: string;
    findings: Finding[];
    filePath?: string;
}

export class WebSocketClient {
    private ws: WebSocket | null = null;
    private config: ConfigurationManager;
    private diagnosticsManager: DiagnosticsManager;
    private reconnectAttempts = 0;
    private maxReconnectAttempts = 5;
    private reconnectDelay = 1000; // Start with 1 second
    private isConnecting = false;
    private statusBarItem: vscode.StatusBarItem;

    constructor(config: ConfigurationManager, diagnosticsManager: DiagnosticsManager) {
        this.config = config;
        this.diagnosticsManager = diagnosticsManager;
        this.statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
        this.statusBarItem.text = "$(sync~spin) AgentScan: Connecting...";
        this.statusBarItem.show();
    }

    async connect(): Promise<void> {
        if (this.isConnecting || (this.ws && this.ws.readyState === WebSocket.OPEN)) {
            return;
        }

        this.isConnecting = true;
        const wsUrl = this.config.getWebSocketUrl();
        const apiKey = this.config.getApiKey();

        try {
            this.ws = new WebSocket(wsUrl, {
                headers: {
                    'Authorization': `Bearer ${apiKey}`,
                    'User-Agent': 'AgentScan-VSCode-Extension/0.1.0'
                }
            });

            this.ws!.on('open', () => {
                console.log('WebSocket connected to AgentScan server');
                this.isConnecting = false;
                this.reconnectAttempts = 0;
                this.reconnectDelay = 1000;
                this.updateStatusBar('connected');
                
                // Send initial message to identify the client
                this.send({
                    type: 'client_info',
                    data: {
                        clientType: 'vscode-extension',
                        version: '0.1.0',
                        workspaceFolder: vscode.workspace.workspaceFolders?.[0]?.uri.fsPath
                    }
                });
            });

            this.ws!.on('message', (data: WebSocket.Data) => {
                try {
                    const message: WebSocketMessage = JSON.parse(data.toString());
                    this.handleMessage(message);
                } catch (error) {
                    console.error('Failed to parse WebSocket message:', error);
                }
            });

            this.ws!.on('close', (code: number, reason: string) => {
                console.log(`WebSocket connection closed: ${code} - ${reason}`);
                this.isConnecting = false;
                this.updateStatusBar('disconnected');
                this.scheduleReconnect();
            });

            this.ws!.on('error', (error: Error) => {
                console.error('WebSocket error:', error);
                this.isConnecting = false;
                this.updateStatusBar('error');
                
                if (this.reconnectAttempts === 0) {
                    vscode.window.showWarningMessage(
                        'AgentScan: Failed to connect to server for real-time updates. Retrying...'
                    );
                }
            });

        } catch (error) {
            console.error('Failed to create WebSocket connection:', error);
            this.isConnecting = false;
            this.updateStatusBar('error');
            this.scheduleReconnect();
        }
    }

    private handleMessage(message: WebSocketMessage) {
        switch (message.type) {
            case 'scan_started':
                this.handleScanStarted(message.data);
                break;
            case 'scan_progress':
                this.handleScanProgress(message.data);
                break;
            case 'scan_completed':
                this.handleScanCompleted(message.data);
                break;
            case 'scan_failed':
                this.handleScanFailed(message.data);
                break;
            case 'finding_updated':
                this.handleFindingUpdated(message.data);
                break;
            default:
                console.log('Unknown WebSocket message type:', message.type);
        }
    }

    private handleScanStarted(data: any) {
        console.log('Scan started:', data);
        this.updateStatusBar('scanning');
        
        if (data.scanType === 'file') {
            vscode.window.showInformationMessage(`AgentScan: Started scanning ${data.fileName || 'file'}`);
        } else {
            vscode.window.showInformationMessage('AgentScan: Started workspace scan');
        }
    }

    private handleScanProgress(data: ScanProgressData) {
        console.log('Scan progress:', data);
        
        const progressText = data.currentAgent 
            ? `Scanning with ${data.currentAgent} (${data.progress}%)`
            : `Scanning... ${data.progress}%`;
            
        this.statusBarItem.text = `$(sync~spin) AgentScan: ${progressText}`;
        
        if (data.message) {
            console.log(`Scan progress: ${data.message}`);
        }
    }

    private handleScanCompleted(data: ScanCompletedData) {
        console.log('Scan completed:', data);
        this.updateStatusBar('connected');
        
        if (data.filePath && data.findings) {
            // Update findings for specific file
            const uri = vscode.Uri.file(data.filePath);
            this.diagnosticsManager.updateFindings(uri, data.findings);
            
            const findingCount = data.findings.length;
            const highSeverityCount = data.findings.filter(f => f.severity === 'high').length;
            
            if (findingCount === 0) {
                vscode.window.showInformationMessage('AgentScan: No security issues found! ✅');
            } else {
                const message = highSeverityCount > 0 
                    ? `AgentScan: Found ${findingCount} security issues (${highSeverityCount} high severity)`
                    : `AgentScan: Found ${findingCount} security issues`;
                    
                vscode.window.showWarningMessage(message);
            }
        } else {
            vscode.window.showInformationMessage('AgentScan: Scan completed');
        }
    }

    private handleScanFailed(data: any) {
        console.error('Scan failed:', data);
        this.updateStatusBar('connected');
        
        const errorMessage = data.error || 'Unknown error occurred';
        vscode.window.showErrorMessage(`AgentScan: Scan failed - ${errorMessage}`);
    }

    private handleFindingUpdated(data: any) {
        console.log('Finding updated:', data);
        
        // Refresh findings for the affected file
        if (data.filePath) {
            const uri = vscode.Uri.file(data.filePath);
            // In a real implementation, you would fetch updated findings from the API
            // For now, we'll just show a notification
            vscode.window.showInformationMessage(`AgentScan: Finding updated in ${data.filePath}`);
        }
    }

    private send(message: any) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(message));
        }
    }

    private scheduleReconnect() {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.log('Max reconnection attempts reached');
            this.updateStatusBar('failed');
            vscode.window.showErrorMessage(
                'AgentScan: Failed to connect to server after multiple attempts. Please check your configuration.'
            );
            return;
        }

        this.reconnectAttempts++;
        console.log(`Scheduling reconnection attempt ${this.reconnectAttempts} in ${this.reconnectDelay}ms`);
        
        setTimeout(() => {
            if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
                this.connect();
            }
        }, this.reconnectDelay);

        // Exponential backoff with jitter
        this.reconnectDelay = Math.min(this.reconnectDelay * 2 + Math.random() * 1000, 30000);
    }

    private updateStatusBar(status: 'connecting' | 'connected' | 'disconnected' | 'scanning' | 'error' | 'failed') {
        switch (status) {
            case 'connecting':
                this.statusBarItem.text = "$(sync~spin) AgentScan: Connecting...";
                this.statusBarItem.color = undefined;
                break;
            case 'connected':
                this.statusBarItem.text = "$(check) AgentScan: Connected";
                this.statusBarItem.color = undefined;
                break;
            case 'disconnected':
                this.statusBarItem.text = "$(circle-outline) AgentScan: Disconnected";
                this.statusBarItem.color = new vscode.ThemeColor('statusBarItem.warningForeground');
                break;
            case 'scanning':
                this.statusBarItem.text = "$(sync~spin) AgentScan: Scanning...";
                this.statusBarItem.color = undefined;
                break;
            case 'error':
                this.statusBarItem.text = "$(warning) AgentScan: Connection Error";
                this.statusBarItem.color = new vscode.ThemeColor('statusBarItem.errorForeground');
                break;
            case 'failed':
                this.statusBarItem.text = "$(x) AgentScan: Connection Failed";
                this.statusBarItem.color = new vscode.ThemeColor('statusBarItem.errorForeground');
                break;
        }
    }

    disconnect() {
        if (this.ws) {
            this.ws.close();
            this.ws = null;
        }
        this.isConnecting = false;
        this.reconnectAttempts = this.maxReconnectAttempts; // Prevent reconnection
        this.updateStatusBar('disconnected');
    }

    dispose() {
        this.disconnect();
        this.statusBarItem.dispose();
    }

    isConnected(): boolean {
        return this.ws !== null && this.ws.readyState === WebSocket.OPEN;
    }
}