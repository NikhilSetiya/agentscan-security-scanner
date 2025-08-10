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
exports.deactivate = exports.activate = void 0;
const vscode = __importStar(require("vscode"));
const AgentScanProvider_1 = require("./providers/AgentScanProvider");
const FindingsProvider_1 = require("./providers/FindingsProvider");
const WebSocketClient_1 = require("./services/WebSocketClient");
const ApiClient_1 = require("./services/ApiClient");
const DiagnosticsManager_1 = require("./services/DiagnosticsManager");
const ConfigurationManager_1 = require("./services/ConfigurationManager");
const StatusBarManager_1 = require("./services/StatusBarManager");
let agentScanProvider;
let findingsProvider;
let webSocketClient;
let diagnosticsManager;
let statusBarManager;
function activate(context) {
    console.log('AgentScan Security extension is now active');
    // Initialize services
    const config = new ConfigurationManager_1.ConfigurationManager();
    const apiClient = new ApiClient_1.ApiClient(config);
    diagnosticsManager = new DiagnosticsManager_1.DiagnosticsManager();
    statusBarManager = new StatusBarManager_1.StatusBarManager();
    // Initialize providers
    agentScanProvider = new AgentScanProvider_1.AgentScanProvider(apiClient, diagnosticsManager, config);
    findingsProvider = new FindingsProvider_1.FindingsProvider(apiClient);
    // Initialize WebSocket client if enabled
    if (config.isWebSocketEnabled()) {
        webSocketClient = new WebSocketClient_1.WebSocketClient(config, diagnosticsManager);
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
exports.activate = activate;
function deactivate() {
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
exports.deactivate = deactivate;
function registerCommands(context) {
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
    context.subscriptions.push(scanFileCommand, scanWorkspaceCommand, clearFindingsCommand, showSettingsCommand, suppressFindingCommand);
}
function setupEventListeners(context) {
    // File save listener for real-time scanning
    const onDidSaveDocument = vscode.workspace.onDidSaveTextDocument(async (document) => {
        const config = new ConfigurationManager_1.ConfigurationManager();
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
            const config = new ConfigurationManager_1.ConfigurationManager();
            agentScanProvider.updateConfiguration(config);
            // Reconnect WebSocket if needed
            if (config.isWebSocketEnabled() && !webSocketClient) {
                webSocketClient = new WebSocketClient_1.WebSocketClient(config, diagnosticsManager);
                webSocketClient.connect();
            }
            else if (!config.isWebSocketEnabled() && webSocketClient) {
                webSocketClient.disconnect();
                webSocketClient = undefined;
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
        const config = new ConfigurationManager_1.ConfigurationManager();
        if (config.isRealTimeScanningEnabled() && config.isLanguageSupported(event.document.languageId)) {
            // Very short debounce for live feedback
            setTimeout(async () => {
                await agentScanProvider.scanFile(event.document, true); // Live mode
            }, 500);
        }
    });
    context.subscriptions.push(onDidSaveDocument, onDidChangeConfiguration, onDidChangeActiveTextEditor, onDidChangeTextDocument);
}
function updateContext() {
    const hasFindings = diagnosticsManager.hasFindings();
    vscode.commands.executeCommand('setContext', 'agentscan.hasFindings', hasFindings);
    const activeEditor = vscode.window.activeTextEditor;
    const hasFinding = activeEditor ? diagnosticsManager.hasFindingAtPosition(activeEditor.document.uri, activeEditor.selection.active) : false;
    vscode.commands.executeCommand('setContext', 'agentscan.hasFinding', hasFinding);
}
//# sourceMappingURL=extension.js.map