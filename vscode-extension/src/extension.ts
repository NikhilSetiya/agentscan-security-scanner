import * as vscode from 'vscode';
import { AgentScanProvider } from './providers/AgentScanProvider';
import { FindingsProvider } from './providers/FindingsProvider';
import { WebSocketClient } from './services/WebSocketClient';
import { ApiClient } from './services/ApiClient';
import { DiagnosticsManager } from './services/DiagnosticsManager';
import { ConfigurationManager } from './services/ConfigurationManager';
import { StatusBarManager } from './services/StatusBarManager';

let agentScanProvider: AgentScanProvider;
let findingsProvider: FindingsProvider;
let webSocketClient: WebSocketClient;
let diagnosticsManager: DiagnosticsManager;
let statusBarManager: StatusBarManager;

export function activate(context: vscode.ExtensionContext) {
    console.log('AgentScan Security extension is now active');

    // Initialize services
    const config = new ConfigurationManager();
    const apiClient = new ApiClient(config);
    diagnosticsManager = new DiagnosticsManager();
    statusBarManager = new StatusBarManager();
    
    // Initialize providers
    agentScanProvider = new AgentScanProvider(apiClient, diagnosticsManager, config);
    findingsProvider = new FindingsProvider(apiClient);
    
    // Initialize WebSocket client if enabled
    if (config.isWebSocketEnabled()) {
        webSocketClient = new WebSocketClient(config, diagnosticsManager);
        webSocketClient.connect();
    }

    // Register tree data providers
    vscode.window.registerTreeDataProvider('agentscanFindings', findingsProvider);

    // Register commands
    registerCommands(context);

    // Set up file watchers and event listeners
    setupEventListeners(context);

    // Update context for when clauses
    updateContext();

    console.log('AgentScan Security extension activated successfully');
}

export function deactivate() {
    console.log('AgentScan Security extension is being deactivated');
    
    if (webSocketClient) {
        webSocketClient.disconnect();
    }
    
    if (diagnosticsManager) {
        diagnosticsManager.dispose();
    }
    
    if (statusBarManager) {
        statusBarManager.dispose();
    }
}

function registerCommands(context: vscode.ExtensionContext) {
    // Scan current file command
    const scanFileCommand = vscode.commands.registerCommand('agentscan.scanFile', async () => {
        const activeEditor = vscode.window.activeTextEditor;
        if (!activeEditor) {
            vscode.window.showWarningMessage('No active file to scan');
            return;
        }

        await agentScanProvider.scanFile(activeEditor.document);
    });

    // Scan workspace command
    const scanWorkspaceCommand = vscode.commands.registerCommand('agentscan.scanWorkspace', async () => {
        if (!vscode.workspace.workspaceFolders) {
            vscode.window.showWarningMessage('No workspace folder open');
            return;
        }

        await agentScanProvider.scanWorkspace();
    });

    // Clear findings command
    const clearFindingsCommand = vscode.commands.registerCommand('agentscan.clearFindings', () => {
        diagnosticsManager.clearAll();
        findingsProvider.refresh();
        updateContext();
        vscode.window.showInformationMessage('All security findings cleared');
    });

    // Show settings command
    const showSettingsCommand = vscode.commands.registerCommand('agentscan.showSettings', () => {
        vscode.commands.executeCommand('workbench.action.openSettings', 'agentscan');
    });

    // Suppress finding command
    const suppressFindingCommand = vscode.commands.registerCommand('agentscan.suppressFinding', async (finding) => {
        await agentScanProvider.suppressFinding(finding);
    });

    // Register all commands
    context.subscriptions.push(
        scanFileCommand,
        scanWorkspaceCommand,
        clearFindingsCommand,
        showSettingsCommand,
        suppressFindingCommand
    );
}

function setupEventListeners(context: vscode.ExtensionContext) {
    // File save listener for real-time scanning
    const onDidSaveDocument = vscode.workspace.onDidSaveTextDocument(async (document) => {
        const config = new ConfigurationManager();
        if (config.isRealTimeScanningEnabled() && config.isLanguageSupported(document.languageId)) {
            // Debounced scanning
            setTimeout(async () => {
                await agentScanProvider.scanFile(document);
            }, config.getScanDebounceMs());
        }
    });

    // Configuration change listener
    const onDidChangeConfiguration = vscode.workspace.onDidChangeConfiguration((event) => {
        if (event.affectsConfiguration('agentscan')) {
            // Reload configuration
            const config = new ConfigurationManager();
            agentScanProvider.updateConfiguration(config);
            
            // Reconnect WebSocket if needed
            if (config.isWebSocketEnabled() && !webSocketClient) {
                webSocketClient = new WebSocketClient(config, diagnosticsManager);
                webSocketClient.connect();
            } else if (!config.isWebSocketEnabled() && webSocketClient) {
                webSocketClient.disconnect();
                webSocketClient = undefined as any;
            }
        }
    });

    // Active editor change listener
    const onDidChangeActiveTextEditor = vscode.window.onDidChangeActiveTextEditor((editor) => {
        if (editor) {
            updateContext();
        }
    });

    // Text document change listener for live feedback
    const onDidChangeTextDocument = vscode.workspace.onDidChangeTextDocument((event) => {
        const config = new ConfigurationManager();
        if (config.isRealTimeScanningEnabled() && config.isLanguageSupported(event.document.languageId)) {
            // Very short debounce for live feedback
            setTimeout(async () => {
                await agentScanProvider.scanFile(event.document, true); // Live mode
            }, 500);
        }
    });

    context.subscriptions.push(
        onDidSaveDocument,
        onDidChangeConfiguration,
        onDidChangeActiveTextEditor,
        onDidChangeTextDocument
    );
}

function updateContext() {
    const hasFindings = diagnosticsManager.hasFindings();
    vscode.commands.executeCommand('setContext', 'agentscan.hasFindings', hasFindings);
    
    const activeEditor = vscode.window.activeTextEditor;
    const hasFinding = activeEditor ? diagnosticsManager.hasFindingAtPosition(
        activeEditor.document.uri,
        activeEditor.selection.active
    ) : false;
    vscode.commands.executeCommand('setContext', 'agentscan.hasFinding', hasFinding);
}