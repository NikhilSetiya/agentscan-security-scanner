import * as vscode from 'vscode';

export class StatusBarManager {
    private statusBarItem: vscode.StatusBarItem;
    private scanStatusItem: vscode.StatusBarItem;

    constructor() {
        // Main status item
        this.statusBarItem = vscode.window.createStatusBarItem(
            vscode.StatusBarAlignment.Right, 
            100
        );
        this.statusBarItem.command = 'agentscan.showSettings';
        this.statusBarItem.tooltip = 'AgentScan Security - Click to open settings';
        
        // Scan status item
        this.scanStatusItem = vscode.window.createStatusBarItem(
            vscode.StatusBarAlignment.Right, 
            99
        );
        
        this.updateStatus('idle');
        this.statusBarItem.show();
    }

    updateStatus(status: 'idle' | 'scanning' | 'connected' | 'disconnected' | 'error', details?: string) {
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

    updateScanProgress(progress: number, currentAgent?: string) {
        if (currentAgent) {
            this.scanStatusItem.text = `${currentAgent}: ${progress}%`;
        } else {
            this.scanStatusItem.text = `${progress}%`;
        }
        this.scanStatusItem.show();
    }

    showFindingsSummary(total: number, high: number, medium: number, low: number) {
        if (total === 0) {
            this.scanStatusItem.text = "$(check) No issues found";
            this.scanStatusItem.color = new vscode.ThemeColor('statusBarItem.prominentForeground');
        } else {
            const highText = high > 0 ? `${high} high` : '';
            const mediumText = medium > 0 ? `${medium} medium` : '';
            const lowText = low > 0 ? `${low} low` : '';
            
            const parts = [highText, mediumText, lowText].filter(Boolean);
            this.scanStatusItem.text = `$(warning) ${total} issues (${parts.join(', ')})`;
            
            if (high > 0) {
                this.scanStatusItem.color = new vscode.ThemeColor('statusBarItem.errorForeground');
            } else if (medium > 0) {
                this.scanStatusItem.color = new vscode.ThemeColor('statusBarItem.warningForeground');
            } else {
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