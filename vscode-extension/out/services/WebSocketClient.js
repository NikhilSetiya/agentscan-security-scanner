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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.WebSocketClient = void 0;
const vscode = __importStar(require("vscode"));
const ws_1 = __importDefault(require("ws"));
class WebSocketClient {
    constructor(config, diagnosticsManager) {
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 10;
        this.reconnectDelay = 1000; // Start with 1 second
        this.maxReconnectDelay = 30000; // Max 30 seconds
        this.isConnecting = false;
        this.isDisposed = false;
        this.heartbeatInterval = null;
        this.connectionTimeout = null;
        this.offlineMode = false;
        this.messageQueue = [];
        this.lastPingTime = 0;
        this.connectionQuality = 'offline';
        this.config = config;
        this.diagnosticsManager = diagnosticsManager;
        this.statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
        this.statusBarItem.text = "$(sync~spin) AgentScan: Connecting...";
        this.statusBarItem.show();
    }
    async connect() {
        if (this.isDisposed || this.isConnecting || (this.ws && this.ws.readyState === ws_1.default.OPEN)) {
            return;
        }
        this.isConnecting = true;
        this.clearTimeouts();
        const wsUrl = this.config.getWebSocketUrl();
        const apiKey = this.config.getApiKey();
        // Set connection timeout
        this.connectionTimeout = setTimeout(() => {
            if (this.isConnecting) {
                console.log('WebSocket connection timeout');
                this.handleConnectionFailure('Connection timeout');
            }
        }, 10000); // 10 second timeout
        try {
            this.ws = new ws_1.default(wsUrl, {
                headers: {
                    'Authorization': `Bearer ${apiKey}`,
                    'User-Agent': 'AgentScan-VSCode-Extension/0.1.0'
                },
                handshakeTimeout: 10000
            });
            this.ws.on('open', () => {
                console.log('WebSocket connected to AgentScan server');
                this.handleConnectionSuccess();
            });
            this.ws.on('message', (data) => {
                this.handleIncomingMessage(data);
            });
            this.ws.on('close', (code, reason) => {
                console.log(`WebSocket connection closed: ${code} - ${reason}`);
                this.handleConnectionClose(code, reason);
            });
            this.ws.on('error', (error) => {
                console.error('WebSocket error:', error);
                this.handleConnectionError(error);
            });
            this.ws.on('pong', () => {
                this.handlePong();
            });
        }
        catch (error) {
            console.error('Failed to create WebSocket connection:', error);
            this.handleConnectionFailure(error instanceof Error ? error.message : 'Unknown error');
        }
    }
    handleConnectionSuccess() {
        this.clearTimeouts();
        this.isConnecting = false;
        this.reconnectAttempts = 0;
        this.reconnectDelay = 1000;
        this.offlineMode = false;
        this.connectionQuality = 'excellent';
        this.updateStatusBar('connected');
        // Start heartbeat
        this.startHeartbeat();
        // Send initial message to identify the client
        this.send({
            type: 'client_info',
            data: {
                clientType: 'vscode-extension',
                version: '0.1.0',
                workspaceFolder: vscode.workspace.workspaceFolders?.[0]?.uri.fsPath,
                capabilities: ['real-time-scanning', 'file-watching', 'progress-updates']
            }
        });
        // Send queued messages
        this.flushMessageQueue();
        // Show connection restored message if we were previously offline
        if (this.reconnectAttempts > 0) {
            vscode.window.showInformationMessage('AgentScan: Connection restored');
        }
    }
    handleConnectionClose(code, reason) {
        this.clearTimeouts();
        this.isConnecting = false;
        // Don't reconnect if this was an intentional disconnect
        if (this.isDisposed || code === 1000) {
            this.updateStatusBar('disconnected');
            return;
        }
        this.connectionQuality = 'offline';
        this.updateStatusBar('disconnected');
        this.scheduleReconnect();
    }
    handleConnectionError(error) {
        this.clearTimeouts();
        this.isConnecting = false;
        this.connectionQuality = 'poor';
        this.updateStatusBar('error');
        if (this.reconnectAttempts === 0) {
            vscode.window.showWarningMessage('AgentScan: Failed to connect to server for real-time updates. Working in offline mode.');
        }
        this.handleConnectionFailure(error.message);
    }
    handleConnectionFailure(reason) {
        this.clearTimeouts();
        this.isConnecting = false;
        this.offlineMode = true;
        this.connectionQuality = 'offline';
        this.updateStatusBar('error');
        this.scheduleReconnect();
    }
    handleIncomingMessage(data) {
        try {
            const message = JSON.parse(data.toString());
            // Handle ping/pong for connection quality monitoring
            if (message.type === 'pong') {
                this.handlePong();
                return;
            }
            this.handleMessage(message);
        }
        catch (error) {
            console.error('Failed to parse WebSocket message:', error);
        }
    }
    startHeartbeat() {
        this.stopHeartbeat();
        this.heartbeatInterval = setInterval(() => {
            if (this.ws && this.ws.readyState === ws_1.default.OPEN) {
                this.lastPingTime = Date.now();
                this.ws.ping();
                // Send periodic ping message
                this.send({
                    type: 'ping',
                    data: { timestamp: this.lastPingTime }
                });
            }
        }, 30000); // Ping every 30 seconds
    }
    stopHeartbeat() {
        if (this.heartbeatInterval) {
            clearInterval(this.heartbeatInterval);
            this.heartbeatInterval = null;
        }
    }
    handlePong() {
        const latency = Date.now() - this.lastPingTime;
        // Update connection quality based on latency
        if (latency < 100) {
            this.connectionQuality = 'excellent';
        }
        else if (latency < 500) {
            this.connectionQuality = 'good';
        }
        else {
            this.connectionQuality = 'poor';
        }
    }
    clearTimeouts() {
        if (this.connectionTimeout) {
            clearTimeout(this.connectionTimeout);
            this.connectionTimeout = null;
        }
    }
    handleMessage(message) {
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
    handleScanStarted(data) {
        console.log('Scan started:', data);
        this.updateStatusBar('scanning');
        if (data.scanType === 'file') {
            vscode.window.showInformationMessage(`AgentScan: Started scanning ${data.fileName || 'file'}`);
        }
        else {
            vscode.window.showInformationMessage('AgentScan: Started workspace scan');
        }
    }
    handleScanProgress(data) {
        console.log('Scan progress:', data);
        const progressText = data.currentAgent
            ? `Scanning with ${data.currentAgent} (${data.progress}%)`
            : `Scanning... ${data.progress}%`;
        this.statusBarItem.text = `$(sync~spin) AgentScan: ${progressText}`;
        if (data.message) {
            console.log(`Scan progress: ${data.message}`);
        }
    }
    handleScanCompleted(data) {
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
            }
            else {
                const message = highSeverityCount > 0
                    ? `AgentScan: Found ${findingCount} security issues (${highSeverityCount} high severity)`
                    : `AgentScan: Found ${findingCount} security issues`;
                vscode.window.showWarningMessage(message);
            }
        }
        else {
            vscode.window.showInformationMessage('AgentScan: Scan completed');
        }
    }
    handleScanFailed(data) {
        console.error('Scan failed:', data);
        this.updateStatusBar('connected');
        const errorMessage = data.error || 'Unknown error occurred';
        vscode.window.showErrorMessage(`AgentScan: Scan failed - ${errorMessage}`);
    }
    handleFindingUpdated(data) {
        console.log('Finding updated:', data);
        // Refresh findings for the affected file
        if (data.filePath) {
            const uri = vscode.Uri.file(data.filePath);
            // In a real implementation, you would fetch updated findings from the API
            // For now, we'll just show a notification
            vscode.window.showInformationMessage(`AgentScan: Finding updated in ${data.filePath}`);
        }
    }
    send(message) {
        if (this.ws && this.ws.readyState === ws_1.default.OPEN) {
            try {
                this.ws.send(JSON.stringify(message));
                return true;
            }
            catch (error) {
                console.error('Failed to send WebSocket message:', error);
                this.queueMessage(message);
                return false;
            }
        }
        else {
            // Queue message for later if we're offline
            this.queueMessage(message);
            return false;
        }
    }
    queueMessage(message) {
        // Only queue certain types of messages
        if (message.type === 'scan_request' || message.type === 'client_info') {
            this.messageQueue.push({
                ...message,
                queuedAt: Date.now()
            });
            // Limit queue size
            if (this.messageQueue.length > 50) {
                this.messageQueue.shift();
            }
        }
    }
    flushMessageQueue() {
        const now = Date.now();
        const validMessages = this.messageQueue.filter(msg => now - msg.queuedAt < 300000 // Only send messages queued within last 5 minutes
        );
        for (const message of validMessages) {
            const { queuedAt, ...messageToSend } = message;
            this.send(messageToSend);
        }
        this.messageQueue = [];
    }
    scheduleReconnect() {
        if (this.isDisposed) {
            return;
        }
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.log('Max reconnection attempts reached, entering offline mode');
            this.offlineMode = true;
            this.updateStatusBar('failed');
            // Show a less intrusive message for offline mode
            vscode.window.setStatusBarMessage('AgentScan: Working in offline mode. Real-time features unavailable.', 10000);
            // Schedule a retry after a longer delay
            setTimeout(() => {
                if (!this.isDisposed) {
                    this.reconnectAttempts = 0;
                    this.reconnectDelay = 1000;
                    this.connect();
                }
            }, 300000); // Retry after 5 minutes
            return;
        }
        this.reconnectAttempts++;
        console.log(`Scheduling reconnection attempt ${this.reconnectAttempts} in ${this.reconnectDelay}ms`);
        setTimeout(() => {
            if (!this.isDisposed && (!this.ws || this.ws.readyState !== ws_1.default.OPEN)) {
                this.connect();
            }
        }, this.reconnectDelay);
        // Exponential backoff with jitter and max delay
        const jitter = Math.random() * 1000;
        this.reconnectDelay = Math.min(this.reconnectDelay * 1.5 + jitter, this.maxReconnectDelay);
    }
    updateStatusBar(status) {
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
        this.isDisposed = true;
        this.clearTimeouts();
        this.stopHeartbeat();
        if (this.ws) {
            this.ws.close(1000, 'Client disconnect');
            this.ws = null;
        }
        this.isConnecting = false;
        this.reconnectAttempts = this.maxReconnectAttempts; // Prevent reconnection
        this.messageQueue = [];
        this.updateStatusBar('disconnected');
    }
    dispose() {
        this.disconnect();
        this.statusBarItem.dispose();
    }
    isConnected() {
        return this.ws !== null && this.ws.readyState === ws_1.default.OPEN;
    }
    isOffline() {
        return this.offlineMode;
    }
    getConnectionQuality() {
        return this.connectionQuality;
    }
    getConnectionStats() {
        return {
            isConnected: this.isConnected(),
            isOffline: this.offlineMode,
            reconnectAttempts: this.reconnectAttempts,
            queuedMessages: this.messageQueue.length,
            connectionQuality: this.connectionQuality
        };
    }
    // Force reconnection (useful for manual retry)
    forceReconnect() {
        this.reconnectAttempts = 0;
        this.reconnectDelay = 1000;
        this.offlineMode = false;
        if (this.ws) {
            this.ws.close();
        }
        setTimeout(() => {
            this.connect();
        }, 1000);
    }
}
exports.WebSocketClient = WebSocketClient;
//# sourceMappingURL=WebSocketClient.js.map