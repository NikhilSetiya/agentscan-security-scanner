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
const CodeActionsProvider_1 = require("./providers/CodeActionsProvider");
const WebSocketClient_1 = require("./services/WebSocketClient");
const ApiClient_1 = require("./services/ApiClient");
const DiagnosticsManager_1 = require("./services/DiagnosticsManager");
const ConfigurationManager_1 = require("./services/ConfigurationManager");
const StatusBarManager_1 = require("./services/StatusBarManager");
const CacheManager_1 = require("./services/CacheManager");
const TelemetryService_1 = require("./services/TelemetryService");
const NavigationService_1 = require("./services/NavigationService");
let agentScanProvider;
let findingsProvider;
let webSocketClient;
let diagnosticsManager;
let statusBarManager;
let cacheManager;
let telemetryService;
let navigationService;
let codeActionsProvider;
let hoverProvider;
function activate(context) {
    console.log('AgentScan Security extension is now active');
    try {
        // Initialize services
        const config = new ConfigurationManager_1.ConfigurationManager();
        const apiClient = new ApiClient_1.ApiClient(config);
        // Initialize core services
        diagnosticsManager = new DiagnosticsManager_1.DiagnosticsManager();
        statusBarManager = new StatusBarManager_1.StatusBarManager();
        cacheManager = new CacheManager_1.CacheManager(context, config.get('cacheMaxAge', 300));
        telemetryService = new TelemetryService_1.TelemetryService(context, config.get('enableTelemetry', true));
        navigationService = new NavigationService_1.NavigationService(diagnosticsManager);
        // Initialize providers
        agentScanProvider = new AgentScanProvider_1.AgentScanProvider(apiClient, diagnosticsManager, config, cacheManager, telemetryService);
        findingsProvider = new FindingsProvider_1.FindingsProvider(apiClient);
        codeActionsProvider = new CodeActionsProvider_1.CodeActionsProvider(diagnosticsManager, telemetryService);
        hoverProvider = new CodeActionsProvider_1.FindingHoverProvider(diagnosticsManager, telemetryService);
        // Initialize WebSocket client if enabled
        if (config.isWebSocketEnabled()) {
            webSocketClient = new WebSocketClient_1.WebSocketClient(config, diagnosticsManager);
            webSocketClient.connect();
        }
        // Register providers
        registerProviders(context);
        // Register commands
        registerCommands(context);
        // Set up file watchers and event listeners
        setupEventListeners(context);
        // Update context for when clauses
        updateContext();
        // Track activation
        telemetryService.trackActivation();
        // Show welcome message for first-time users
        showWelcomeMessage(context);
        console.log('AgentScan Security extension activated successfully');
    }
    catch (error) {
        console.error('Failed to activate AgentScan extension:', error);
        vscode.window.showErrorMessage(`AgentScan: Failed to activate extension - ${error}`);
        if (telemetryService) {
            telemetryService.trackError(error instanceof Error ? error : new Error(String(error)), 'activation');
        }
    }
}
exports.activate = activate;
function deactivate() {
    console.log('AgentScan Security extension is being deactivated');
    try {
        // Dispose services in reverse order
        if (webSocketClient) {
            webSocketClient.dispose();
        }
        if (agentScanProvider) {
            agentScanProvider.dispose();
        }
        if (diagnosticsManager) {
            diagnosticsManager.dispose();
        }
        if (statusBarManager) {
            statusBarManager.dispose();
        }
        if (cacheManager) {
            cacheManager.dispose();
        }
        if (telemetryService) {
            telemetryService.dispose();
        }
        console.log('AgentScan Security extension deactivated successfully');
    }
    catch (error) {
        console.error('Error during extension deactivation:', error);
    }
}
exports.deactivate = deactivate;
function registerProviders(context) {
    // Register tree data providers
    vscode.window.registerTreeDataProvider('agentscanFindings', findingsProvider);
    // Register code actions provider
    const codeActionsDisposable = vscode.languages.registerCodeActionsProvider({ scheme: 'file' }, codeActionsProvider, {
        providedCodeActionKinds: [vscode.CodeActionKind.QuickFix]
    });
    // Register hover provider
    const hoverDisposable = vscode.languages.registerHoverProvider({ scheme: 'file' }, hoverProvider);
    context.subscriptions.push(codeActionsDisposable, hoverDisposable);
}
function registerCommands(context) {
    // Scan current file command
    const scanFileCommand = vscode.commands.registerCommand('agentscan.scanFile', async () => {
        const activeEditor = vscode.window.activeTextEditor;
        if (!activeEditor) {
            vscode.window.showWarningMessage('No active file to scan');
            return;
        }
        telemetryService.trackUserAction('command.scanFile');
        await agentScanProvider.scanFile(activeEditor.document);
    });
    // Scan workspace command
    const scanWorkspaceCommand = vscode.commands.registerCommand('agentscan.scanWorkspace', async () => {
        if (!vscode.workspace.workspaceFolders) {
            vscode.window.showWarningMessage('No workspace folder open');
            return;
        }
        telemetryService.trackUserAction('command.scanWorkspace');
        await agentScanProvider.scanWorkspace();
    });
    // Clear findings command
    const clearFindingsCommand = vscode.commands.registerCommand('agentscan.clearFindings', () => {
        diagnosticsManager.clearAll();
        findingsProvider.refresh();
        navigationService.reset();
        updateContext();
        telemetryService.trackUserAction('command.clearFindings');
        vscode.window.showInformationMessage('All security findings cleared');
    });
    // Show settings command
    const showSettingsCommand = vscode.commands.registerCommand('agentscan.showSettings', () => {
        telemetryService.trackUserAction('command.showSettings');
        vscode.commands.executeCommand('workbench.action.openSettings', 'agentscan');
    });
    // Suppress finding command
    const suppressFindingCommand = vscode.commands.registerCommand('agentscan.suppressFinding', async (finding) => {
        telemetryService.trackUserAction('command.suppressFinding', { severity: finding?.severity });
        await agentScanProvider.suppressFinding(finding);
    });
    // Mark as fixed command
    const markAsFixedCommand = vscode.commands.registerCommand('agentscan.markAsFixed', async (finding) => {
        telemetryService.trackUserAction('command.markAsFixed', { severity: finding?.severity });
        await agentScanProvider.markAsFixed(finding);
    });
    // Ignore rule command
    const ignoreRuleCommand = vscode.commands.registerCommand('agentscan.ignoreRule', async (finding) => {
        telemetryService.trackUserAction('command.ignoreRule', { ruleId: finding?.ruleId });
        await agentScanProvider.ignoreRule(finding);
    });
    // Learn more command
    const learnMoreCommand = vscode.commands.registerCommand('agentscan.learnMore', async (finding) => {
        telemetryService.trackUserAction('command.learnMore', { ruleId: finding?.ruleId });
        await showLearnMore(finding);
    });
    // Navigation commands
    const nextFindingCommand = vscode.commands.registerCommand('agentscan.nextFinding', async () => {
        telemetryService.trackUserAction('command.nextFinding');
        await navigationService.goToNextFinding();
    });
    const previousFindingCommand = vscode.commands.registerCommand('agentscan.previousFinding', async () => {
        telemetryService.trackUserAction('command.previousFinding');
        await navigationService.goToPreviousFinding();
    });
    // Toggle real-time scanning
    const toggleRealTimeScanningCommand = vscode.commands.registerCommand('agentscan.toggleRealTimeScanning', async () => {
        const config = new ConfigurationManager_1.ConfigurationManager();
        const currentValue = config.isRealTimeScanningEnabled();
        await config.updateConfiguration('enableRealTimeScanning', !currentValue);
        telemetryService.trackUserAction('command.toggleRealTimeScanning', { enabled: String(!currentValue) });
        const status = !currentValue ? 'enabled' : 'disabled';
        vscode.window.showInformationMessage(`AgentScan: Real-time scanning ${status}`);
    });
    // Show security health
    const showSecurityHealthCommand = vscode.commands.registerCommand('agentscan.showSecurityHealth', async () => {
        telemetryService.trackUserAction('command.showSecurityHealth');
        await showSecurityHealthPanel();
    });
    // Register all commands
    context.subscriptions.push(scanFileCommand, scanWorkspaceCommand, clearFindingsCommand, showSettingsCommand, suppressFindingCommand, markAsFixedCommand, ignoreRuleCommand, learnMoreCommand, nextFindingCommand, previousFindingCommand, toggleRealTimeScanningCommand, showSecurityHealthCommand);
}
function setupEventListeners(context) {
    // File save listener for real-time scanning
    const onDidSaveDocument = vscode.workspace.onDidSaveTextDocument(async (document) => {
        const config = new ConfigurationManager_1.ConfigurationManager();
        if (config.isRealTimeScanningEnabled() && config.isLanguageSupported(document.languageId)) {
            // Debounced scanning with performance tracking
            const startTime = Date.now();
            setTimeout(async () => {
                try {
                    await agentScanProvider.scanFile(document);
                    telemetryService.trackPerformance({
                        operation: 'file_save_scan',
                        duration: Date.now() - startTime,
                        success: true
                    });
                }
                catch (error) {
                    telemetryService.trackPerformance({
                        operation: 'file_save_scan',
                        duration: Date.now() - startTime,
                        success: false,
                        error: error instanceof Error ? error.message : String(error)
                    });
                }
            }, config.getScanDebounceMs());
        }
    });
    // Configuration change listener
    const onDidChangeConfiguration = vscode.workspace.onDidChangeConfiguration((event) => {
        if (event.affectsConfiguration('agentscan')) {
            const config = new ConfigurationManager_1.ConfigurationManager();
            // Track configuration changes
            if (event.affectsConfiguration('agentscan.enableRealTimeScanning')) {
                telemetryService.trackConfigurationChange('enableRealTimeScanning', !config.isRealTimeScanningEnabled(), config.isRealTimeScanningEnabled());
            }
            // Update services
            agentScanProvider.updateConfiguration(config);
            cacheManager.updateMaxAge(config.get('cacheMaxAge', 300));
            telemetryService.setEnabled(config.get('enableTelemetry', true));
            // Handle WebSocket connection changes
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
            // Update decorations for the new editor
            diagnosticsManager.onDidChangeActiveTextEditor(editor);
            updateContext();
        }
    });
    // Text document change listener for live feedback
    const onDidChangeTextDocument = vscode.workspace.onDidChangeTextDocument((event) => {
        const config = new ConfigurationManager_1.ConfigurationManager();
        if (config.isRealTimeScanningEnabled() &&
            config.isLanguageSupported(event.document.languageId) &&
            event.contentChanges.length > 0) {
            // Very short debounce for live feedback
            setTimeout(async () => {
                try {
                    await agentScanProvider.scanFile(event.document, true); // Live mode
                }
                catch (error) {
                    // Silent failure for live scanning
                    console.error('Live scan failed:', error);
                }
            }, 500);
        }
    });
    // Selection change listener for context updates
    const onDidChangeTextEditorSelection = vscode.window.onDidChangeTextEditorSelection((event) => {
        updateContext();
    });
    // Workspace folder change listener
    const onDidChangeWorkspaceFolders = vscode.workspace.onDidChangeWorkspaceFolders((event) => {
        // Clear cache when workspace changes
        cacheManager.invalidateAll();
        diagnosticsManager.clearAll();
        findingsProvider.refresh();
        updateContext();
    });
    context.subscriptions.push(onDidSaveDocument, onDidChangeConfiguration, onDidChangeActiveTextEditor, onDidChangeTextDocument, onDidChangeTextEditorSelection, onDidChangeWorkspaceFolders);
}
function updateContext() {
    const hasFindings = diagnosticsManager.hasFindings();
    vscode.commands.executeCommand('setContext', 'agentscan.hasFindings', hasFindings);
    const activeEditor = vscode.window.activeTextEditor;
    const hasFinding = activeEditor ? diagnosticsManager.hasFindingAtPosition(activeEditor.document.uri, activeEditor.selection.active) : false;
    vscode.commands.executeCommand('setContext', 'agentscan.hasFinding', hasFinding);
    // Update status bar with findings summary
    if (hasFindings) {
        const allFindings = diagnosticsManager.getAllFindings();
        const high = allFindings.filter(f => f.severity === 'high').length;
        const medium = allFindings.filter(f => f.severity === 'medium').length;
        const low = allFindings.filter(f => f.severity === 'low').length;
        statusBarManager.showFindingsSummary(allFindings.length, high, medium, low);
    }
    else {
        statusBarManager.updateStatus('idle');
    }
}
async function showLearnMore(finding) {
    if (!finding) {
        const currentFinding = navigationService.getFindingAtCursor();
        if (!currentFinding) {
            vscode.window.showWarningMessage('No finding selected');
            return;
        }
        finding = currentFinding;
    }
    const panel = vscode.window.createWebviewPanel('agentscanLearnMore', `Learn More: ${finding.title}`, vscode.ViewColumn.Beside, {
        enableScripts: true,
        retainContextWhenHidden: true
    });
    panel.webview.html = createLearnMoreContent(finding);
}
function createLearnMoreContent(finding) {
    const severityColor = finding.severity === 'high' ? '#dc2626' :
        finding.severity === 'medium' ? '#d97706' : '#2563eb';
    return `
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>Learn More: ${finding.title}</title>
        <style>
            body {
                font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                line-height: 1.6;
                color: var(--vscode-foreground);
                background-color: var(--vscode-editor-background);
                padding: 20px;
                margin: 0;
            }
            .header {
                border-bottom: 1px solid var(--vscode-panel-border);
                padding-bottom: 20px;
                margin-bottom: 20px;
            }
            .severity-badge {
                background-color: ${severityColor};
                color: white;
                padding: 4px 8px;
                border-radius: 4px;
                font-size: 12px;
                font-weight: bold;
                text-transform: uppercase;
            }
            .section {
                margin-bottom: 30px;
            }
            .section h3 {
                color: var(--vscode-textLink-foreground);
                margin-bottom: 10px;
            }
            .code-block {
                background-color: var(--vscode-textCodeBlock-background);
                border: 1px solid var(--vscode-panel-border);
                border-radius: 4px;
                padding: 15px;
                font-family: 'Courier New', monospace;
                overflow-x: auto;
            }
            .reference-link {
                color: var(--vscode-textLink-foreground);
                text-decoration: none;
                margin-right: 15px;
            }
            .reference-link:hover {
                text-decoration: underline;
            }
        </style>
    </head>
    <body>
        <div class="header">
            <h1>${finding.title}</h1>
            <span class="severity-badge">${finding.severity}</span>
        </div>
        
        <div class="section">
            <h3>Description</h3>
            <p>${finding.description}</p>
        </div>
        
        <div class="section">
            <h3>Details</h3>
            <p><strong>Tool:</strong> ${finding.tool}</p>
            <p><strong>Rule ID:</strong> ${finding.ruleId}</p>
            <p><strong>Category:</strong> ${finding.category || 'Security'}</p>
            <p><strong>Confidence:</strong> ${(finding.confidence * 100).toFixed(1)}%</p>
        </div>
        
        ${finding.codeSnippet ? `
        <div class="section">
            <h3>Code Example</h3>
            <div class="code-block">${finding.codeSnippet}</div>
        </div>
        ` : ''}
        
        ${finding.fixSuggestion ? `
        <div class="section">
            <h3>Suggested Fix</h3>
            <p>${finding.fixSuggestion}</p>
        </div>
        ` : ''}
        
        ${finding.references && finding.references.length > 0 ? `
        <div class="section">
            <h3>References</h3>
            ${finding.references.map((ref) => `<a href="${ref}" class="reference-link" target="_blank">${ref}</a>`).join('<br>')}
        </div>
        ` : ''}
    </body>
    </html>
    `;
}
async function showSecurityHealthPanel() {
    const panel = vscode.window.createWebviewPanel('agentscanSecurityHealth', 'Security Health Dashboard', vscode.ViewColumn.One, {
        enableScripts: true,
        retainContextWhenHidden: true
    });
    const allFindings = diagnosticsManager.getAllFindings();
    const stats = {
        total: allFindings.length,
        high: allFindings.filter(f => f.severity === 'high').length,
        medium: allFindings.filter(f => f.severity === 'medium').length,
        low: allFindings.filter(f => f.severity === 'low').length
    };
    const cacheStats = cacheManager.getStats();
    const connectionStats = webSocketClient ? webSocketClient.getConnectionStats() : null;
    panel.webview.html = createSecurityHealthContent(stats, cacheStats, connectionStats);
}
function createSecurityHealthContent(findingStats, cacheStats, connectionStats) {
    const healthScore = calculateHealthScore(findingStats);
    const healthColor = healthScore >= 80 ? '#059669' : healthScore >= 60 ? '#d97706' : '#dc2626';
    return `
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>Security Health Dashboard</title>
        <style>
            body {
                font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
                line-height: 1.6;
                color: var(--vscode-foreground);
                background-color: var(--vscode-editor-background);
                padding: 20px;
                margin: 0;
            }
            .health-score {
                text-align: center;
                margin-bottom: 30px;
            }
            .score-circle {
                width: 120px;
                height: 120px;
                border-radius: 50%;
                background: conic-gradient(${healthColor} ${healthScore * 3.6}deg, var(--vscode-panel-border) 0deg);
                display: flex;
                align-items: center;
                justify-content: center;
                margin: 0 auto 15px;
                position: relative;
            }
            .score-inner {
                width: 90px;
                height: 90px;
                border-radius: 50%;
                background-color: var(--vscode-editor-background);
                display: flex;
                align-items: center;
                justify-content: center;
                font-size: 24px;
                font-weight: bold;
            }
            .stats-grid {
                display: grid;
                grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
                gap: 20px;
                margin-bottom: 30px;
            }
            .stat-card {
                background-color: var(--vscode-sideBar-background);
                border: 1px solid var(--vscode-panel-border);
                border-radius: 8px;
                padding: 20px;
            }
            .stat-title {
                font-size: 14px;
                color: var(--vscode-descriptionForeground);
                margin-bottom: 10px;
            }
            .stat-value {
                font-size: 24px;
                font-weight: bold;
                color: var(--vscode-foreground);
            }
            .severity-high { color: #dc2626; }
            .severity-medium { color: #d97706; }
            .severity-low { color: #2563eb; }
        </style>
    </head>
    <body>
        <h1>Security Health Dashboard</h1>
        
        <div class="health-score">
            <div class="score-circle">
                <div class="score-inner">${healthScore}</div>
            </div>
            <h2>Security Health Score</h2>
        </div>
        
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-title">Total Findings</div>
                <div class="stat-value">${findingStats.total}</div>
            </div>
            <div class="stat-card">
                <div class="stat-title">High Severity</div>
                <div class="stat-value severity-high">${findingStats.high}</div>
            </div>
            <div class="stat-card">
                <div class="stat-title">Medium Severity</div>
                <div class="stat-value severity-medium">${findingStats.medium}</div>
            </div>
            <div class="stat-card">
                <div class="stat-title">Low Severity</div>
                <div class="stat-value severity-low">${findingStats.low}</div>
            </div>
            <div class="stat-card">
                <div class="stat-title">Cache Entries</div>
                <div class="stat-value">${cacheStats.totalEntries}</div>
            </div>
            ${connectionStats ? `
            <div class="stat-card">
                <div class="stat-title">Connection Status</div>
                <div class="stat-value">${connectionStats.isConnected ? '🟢 Connected' : '🔴 Offline'}</div>
            </div>
            ` : ''}
        </div>
    </body>
    </html>
    `;
}
function calculateHealthScore(stats) {
    if (stats.total === 0)
        return 100;
    // Calculate score based on severity distribution
    const highPenalty = stats.high * 10;
    const mediumPenalty = stats.medium * 5;
    const lowPenalty = stats.low * 1;
    const totalPenalty = highPenalty + mediumPenalty + lowPenalty;
    const maxScore = 100;
    return Math.max(0, Math.min(100, maxScore - totalPenalty));
}
function showWelcomeMessage(context) {
    const hasShownWelcome = context.globalState.get('agentscan.hasShownWelcome', false);
    if (!hasShownWelcome) {
        vscode.window.showInformationMessage('Welcome to AgentScan Security! Configure your API key in settings to get started.', 'Open Settings', 'Learn More').then(selection => {
            if (selection === 'Open Settings') {
                vscode.commands.executeCommand('agentscan.showSettings');
            }
            else if (selection === 'Learn More') {
                vscode.env.openExternal(vscode.Uri.parse('https://github.com/NikhilSetiya/agentscan-security-scanner'));
            }
        });
        context.globalState.update('agentscan.hasShownWelcome', true);
    }
}
//# sourceMappingURL=extension.js.map