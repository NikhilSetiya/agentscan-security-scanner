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
exports.StatusBarManager = void 0;
const vscode = __importStar(require("vscode"));
class StatusBarManager {
    constructor() {
        // Main status item
        this.statusBarItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 100);
        this.statusBarItem.command = 'agentscan.showSettings';
        this.statusBarItem.tooltip = 'AgentScan Security - Click to open settings';
        // Scan status item
        this.scanStatusItem = vscode.window.createStatusBarItem(vscode.StatusBarAlignment.Right, 99);
        this.updateStatus('idle');
        this.statusBarItem.show();
    }
    updateStatus(status, details) {
        switch (status) {
            case 'idle':
                this.statusBarItem.text = "$(shield) AgentScan";
                this.statusBarItem.color = undefined;
                this.statusBarItem.tooltip = 'AgentScan Security - Ready';
                this.scanStatusItem.hide();
                break;
            case 'scanning':
                this.statusBarItem.text = "$(sync~spin) AgentScan";
                this.statusBarItem.color = undefined;
                this.statusBarItem.tooltip = 'AgentScan Security - Scanning...';
                this.scanStatusItem.text = details || 'Scanning...';
                this.scanStatusItem.show();
                break;
            case 'connected':
                this.statusBarItem.text = "$(check) AgentScan";
                this.statusBarItem.color = new vscode.ThemeColor('statusBarItem.prominentForeground');
                this.statusBarItem.tooltip = 'AgentScan Security - Connected';
                this.scanStatusItem.hide();
                break;
            case 'disconnected':
                this.statusBarItem.text = "$(circle-outline) AgentScan";
                this.statusBarItem.color = new vscode.ThemeColor('statusBarItem.warningForeground');
                this.statusBarItem.tooltip = 'AgentScan Security - Disconnected';
                this.scanStatusItem.hide();
                break;
            case 'error':
                this.statusBarItem.text = "$(warning) AgentScan";
                this.statusBarItem.color = new vscode.ThemeColor('statusBarItem.errorForeground');
                this.statusBarItem.tooltip = `AgentScan Security - Error: ${details || 'Unknown error'}`;
                this.scanStatusItem.hide();
                break;
        }
    }
    updateScanProgress(progress, currentAgent) {
        if (currentAgent) {
            this.scanStatusItem.text = `${currentAgent}: ${progress}%`;
        }
        else {
            this.scanStatusItem.text = `${progress}%`;
        }
        this.scanStatusItem.show();
    }
    showFindingsSummary(total, high, medium, low) {
        if (total === 0) {
            this.scanStatusItem.text = "$(check) No issues found";
            this.scanStatusItem.color = new vscode.ThemeColor('statusBarItem.prominentForeground');
        }
        else {
            const highText = high > 0 ? `${high} high` : '';
            const mediumText = medium > 0 ? `${medium} medium` : '';
            const lowText = low > 0 ? `${low} low` : '';
            const parts = [highText, mediumText, lowText].filter(Boolean);
            this.scanStatusItem.text = `$(warning) ${total} issues (${parts.join(', ')})`;
            if (high > 0) {
                this.scanStatusItem.color = new vscode.ThemeColor('statusBarItem.errorForeground');
            }
            else if (medium > 0) {
                this.scanStatusItem.color = new vscode.ThemeColor('statusBarItem.warningForeground');
            }
            else {
                this.scanStatusItem.color = undefined;
            }
        }
        this.scanStatusItem.command = 'agentscan.showFindings';
        this.scanStatusItem.tooltip = `Click to view security findings`;
        this.scanStatusItem.show();
    }
    dispose() {
        this.statusBarItem.dispose();
        this.scanStatusItem.dispose();
    }
}
exports.StatusBarManager = StatusBarManager;
//# sourceMappingURL=StatusBarManager.js.map