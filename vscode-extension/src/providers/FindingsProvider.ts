import * as vscode from 'vscode';
import * as path from 'path';
import { ApiClient, Finding } from '../services/ApiClient';

export class FindingTreeItem extends vscode.TreeItem {
    constructor(
        public readonly finding: Finding,
        public readonly collapsibleState: vscode.TreeItemCollapsibleState,
        public readonly contextValue: string
    ) {
        super(finding.title, collapsibleState);
        
        this.tooltip = this.createTooltip();
        this.description = this.createDescription();
        this.iconPath = this.getIcon();
        this.command = {
            command: 'vscode.open',
            title: 'Open File',
            arguments: [
                vscode.Uri.file(finding.filePath),
                {
                    selection: new vscode.Range(
                        new vscode.Position(Math.max(0, finding.lineNumber - 1), finding.columnNumber || 0),
                        new vscode.Position(Math.max(0, finding.lineNumber - 1), (finding.columnNumber || 0) + 10)
                    )
                }
            ]
        };
    }

    private createTooltip(): string {
        return `${this.finding.title}\n\n` +
               `Severity: ${this.finding.severity.toUpperCase()}\n` +
               `Tool: ${this.finding.tool}\n` +
               `Rule: ${this.finding.ruleId}\n` +
               `File: ${this.finding.filePath}:${this.finding.lineNumber}\n` +
               `Confidence: ${(this.finding.confidence * 100).toFixed(1)}%\n\n` +
               `${this.finding.description}`;
    }

    private createDescription(): string {
        const fileName = path.basename(this.finding.filePath);
        return `${fileName}:${this.finding.lineNumber} • ${this.finding.tool}`;
    }

    private getIcon(): vscode.ThemeIcon {
        switch (this.finding.severity) {
            case 'high':
                return new vscode.ThemeIcon('error', new vscode.ThemeColor('errorForeground'));
            case 'medium':
                return new vscode.ThemeIcon('warning', new vscode.ThemeColor('warningForeground'));
            case 'low':
                return new vscode.ThemeIcon('info', new vscode.ThemeColor('foreground'));
            default:
                return new vscode.ThemeIcon('circle-outline');
        }
    }
}

export class FileTreeItem extends vscode.TreeItem {
    constructor(
        public readonly filePath: string,
        public readonly findings: Finding[],
        public readonly collapsibleState: vscode.TreeItemCollapsibleState
    ) {
        super(path.basename(filePath), collapsibleState);
        
        this.tooltip = this.createTooltip();
        this.description = this.createDescription();
        this.iconPath = new vscode.ThemeIcon('file');
        this.contextValue = 'file';
        this.resourceUri = vscode.Uri.file(filePath);
    }

    private createTooltip(): string {
        const high = this.findings.filter(f => f.severity === 'high').length;
        const medium = this.findings.filter(f => f.severity === 'medium').length;
        const low = this.findings.filter(f => f.severity === 'low').length;
        
        return `${this.filePath}\n\n` +
               `${this.findings.length} findings:\n` +
               `• ${high} high severity\n` +
               `• ${medium} medium severity\n` +
               `• ${low} low severity`;
    }

    private createDescription(): string {
        const high = this.findings.filter(f => f.severity === 'high').length;
        const medium = this.findings.filter(f => f.severity === 'medium').length;
        const low = this.findings.filter(f => f.severity === 'low').length;
        
        const parts = [];
        if (high > 0) parts.push(`${high}🔴`);
        if (medium > 0) parts.push(`${medium}🟡`);
        if (low > 0) parts.push(`${low}🔵`);
        
        return parts.join(' ');
    }
}

export class SeverityTreeItem extends vscode.TreeItem {
    constructor(
        public readonly severity: string,
        public readonly findings: Finding[],
        public readonly collapsibleState: vscode.TreeItemCollapsibleState
    ) {
        super(`${severity.toUpperCase()} (${findings.length})`, collapsibleState);
        
        this.tooltip = `${findings.length} ${severity} severity findings`;
        this.iconPath = this.getIcon();
        this.contextValue = 'severity';
    }

    private getIcon(): vscode.ThemeIcon {
        switch (this.severity) {
            case 'high':
                return new vscode.ThemeIcon('error', new vscode.ThemeColor('errorForeground'));
            case 'medium':
                return new vscode.ThemeIcon('warning', new vscode.ThemeColor('warningForeground'));
            case 'low':
                return new vscode.ThemeIcon('info', new vscode.ThemeColor('foreground'));
            default:
                return new vscode.ThemeIcon('circle-outline');
        }
    }
}

export class FindingsProvider implements vscode.TreeDataProvider<vscode.TreeItem> {
    private _onDidChangeTreeData: vscode.EventEmitter<vscode.TreeItem | undefined | null | void> = new vscode.EventEmitter<vscode.TreeItem | undefined | null | void>();
    readonly onDidChangeTreeData: vscode.Event<vscode.TreeItem | undefined | null | void> = this._onDidChangeTreeData.event;

    private findings: Finding[] = [];
    private groupBy: 'file' | 'severity' = 'file';

    constructor(private apiClient: ApiClient) {}

    refresh(): void {
        this._onDidChangeTreeData.fire();
    }

    async loadFindings(): Promise<void> {
        try {
            this.findings = await this.apiClient.getFindings();
            this.refresh();
        } catch (error) {
            console.error('Failed to load findings:', error);
            vscode.window.showErrorMessage(`Failed to load findings: ${error}`);
        }
    }

    setGroupBy(groupBy: 'file' | 'severity'): void {
        this.groupBy = groupBy;
        this.refresh();
    }

    getTreeItem(element: vscode.TreeItem): vscode.TreeItem {
        return element;
    }

    getChildren(element?: vscode.TreeItem): Thenable<vscode.TreeItem[]> {
        if (!element) {
            // Root level
            if (this.findings.length === 0) {
                return Promise.resolve([]);
            }

            if (this.groupBy === 'file') {
                return Promise.resolve(this.getFileGroups());
            } else {
                return Promise.resolve(this.getSeverityGroups());
            }
        }

        if (element instanceof FileTreeItem) {
            // Return findings for this file
            return Promise.resolve(
                element.findings.map(finding => 
                    new FindingTreeItem(finding, vscode.TreeItemCollapsibleState.None, 'finding')
                )
            );
        }

        if (element instanceof SeverityTreeItem) {
            // Return findings for this severity level
            return Promise.resolve(
                element.findings.map(finding => 
                    new FindingTreeItem(finding, vscode.TreeItemCollapsibleState.None, 'finding')
                )
            );
        }

        return Promise.resolve([]);
    }

    private getFileGroups(): vscode.TreeItem[] {
        const fileGroups = new Map<string, Finding[]>();
        
        // Group findings by file
        for (const finding of this.findings) {
            if (!fileGroups.has(finding.filePath)) {
                fileGroups.set(finding.filePath, []);
            }
            fileGroups.get(finding.filePath)!.push(finding);
        }

        // Create tree items for each file
        const items: FileTreeItem[] = [];
        for (const [filePath, findings] of fileGroups.entries()) {
            items.push(new FileTreeItem(
                filePath,
                findings,
                vscode.TreeItemCollapsibleState.Collapsed
            ));
        }

        // Sort by file name
        items.sort((a, b) => path.basename(a.filePath).localeCompare(path.basename(b.filePath)));
        
        return items;
    }

    private getSeverityGroups(): vscode.TreeItem[] {
        const severityGroups = new Map<string, Finding[]>();
        
        // Group findings by severity
        for (const finding of this.findings) {
            if (!severityGroups.has(finding.severity)) {
                severityGroups.set(finding.severity, []);
            }
            severityGroups.get(finding.severity)!.push(finding);
        }

        // Create tree items for each severity level
        const items: SeverityTreeItem[] = [];
        const severityOrder = ['high', 'medium', 'low'];
        
        for (const severity of severityOrder) {
            const findings = severityGroups.get(severity);
            if (findings && findings.length > 0) {
                items.push(new SeverityTreeItem(
                    severity,
                    findings,
                    vscode.TreeItemCollapsibleState.Collapsed
                ));
            }
        }
        
        return items;
    }

    // Commands for tree view
    async suppressFinding(finding: Finding): Promise<void> {
        try {
            const reason = await vscode.window.showInputBox({
                prompt: 'Enter reason for suppressing this finding',
                placeHolder: 'e.g., False positive - this code is safe'
            });

            if (!reason) {
                return;
            }

            await this.apiClient.suppressFinding(finding.id, { reason });
            
            // Remove from local findings list
            this.findings = this.findings.filter(f => f.id !== finding.id);
            this.refresh();
            
            vscode.window.showInformationMessage('Finding suppressed successfully');
        } catch (error) {
            vscode.window.showErrorMessage(`Failed to suppress finding: ${error}`);
        }
    }

    async markAsFixed(finding: Finding): Promise<void> {
        try {
            await this.apiClient.updateFindingStatus(finding.id, 'fixed', 'Marked as fixed from VS Code');
            
            // Update local findings list
            const index = this.findings.findIndex(f => f.id === finding.id);
            if (index !== -1) {
                this.findings[index].status = 'fixed';
                this.refresh();
            }
            
            vscode.window.showInformationMessage('Finding marked as fixed');
        } catch (error) {
            vscode.window.showErrorMessage(`Failed to mark finding as fixed: ${error}`);
        }
    }

    async openFinding(finding: Finding): Promise<void> {
        try {
            const uri = vscode.Uri.file(finding.filePath);
            const document = await vscode.workspace.openTextDocument(uri);
            const editor = await vscode.window.showTextDocument(document);
            
            // Navigate to the finding location
            const position = new vscode.Position(
                Math.max(0, finding.lineNumber - 1),
                finding.columnNumber || 0
            );
            
            editor.selection = new vscode.Selection(position, position);
            editor.revealRange(new vscode.Range(position, position), vscode.TextEditorRevealType.InCenter);
            
        } catch (error) {
            vscode.window.showErrorMessage(`Failed to open finding: ${error}`);
        }
    }

    getFindings(): Finding[] {
        return this.findings;
    }

    getFindingCount(): number {
        return this.findings.length;
    }

    getFindingCountBySeverity(): { high: number; medium: number; low: number } {
        return {
            high: this.findings.filter(f => f.severity === 'high').length,
            medium: this.findings.filter(f => f.severity === 'medium').length,
            low: this.findings.filter(f => f.severity === 'low').length
        };
    }
}