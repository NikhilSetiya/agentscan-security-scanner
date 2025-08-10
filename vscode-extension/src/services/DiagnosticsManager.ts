import * as vscode from 'vscode';
import { Finding } from './ApiClient';

export class DiagnosticsManager {
    private diagnosticCollection: vscode.DiagnosticCollection;
    private findingsMap: Map<string, Finding[]> = new Map();
    private decorationTypes: Map<string, vscode.TextEditorDecorationType> = new Map();

    constructor() {
        this.diagnosticCollection = vscode.languages.createDiagnosticCollection('agentscan');
        this.initializeDecorationTypes();
    }

    private initializeDecorationTypes() {
        // High severity decoration
        this.decorationTypes.set('high', vscode.window.createTextEditorDecorationType({
            backgroundColor: new vscode.ThemeColor('agentscan.errorHighlight'),
            borderWidth: '1px',
            borderStyle: 'solid',
            borderColor: new vscode.ThemeColor('agentscan.errorHighlight'),
            borderRadius: '2px',
            overviewRulerColor: new vscode.ThemeColor('agentscan.errorHighlight'),
            overviewRulerLane: vscode.OverviewRulerLane.Right,
            after: {
                contentText: ' 🔴',
                color: new vscode.ThemeColor('agentscan.errorHighlight')
            }
        }));

        // Medium severity decoration
        this.decorationTypes.set('medium', vscode.window.createTextEditorDecorationType({
            backgroundColor: new vscode.ThemeColor('agentscan.warningHighlight'),
            borderWidth: '1px',
            borderStyle: 'solid',
            borderColor: new vscode.ThemeColor('agentscan.warningHighlight'),
            borderRadius: '2px',
            overviewRulerColor: new vscode.ThemeColor('agentscan.warningHighlight'),
            overviewRulerLane: vscode.OverviewRulerLane.Right,
            after: {
                contentText: ' 🟡',
                color: new vscode.ThemeColor('agentscan.warningHighlight')
            }
        }));

        // Low severity decoration
        this.decorationTypes.set('low', vscode.window.createTextEditorDecorationType({
            backgroundColor: new vscode.ThemeColor('agentscan.infoHighlight'),
            borderWidth: '1px',
            borderStyle: 'solid',
            borderColor: new vscode.ThemeColor('agentscan.infoHighlight'),
            borderRadius: '2px',
            overviewRulerColor: new vscode.ThemeColor('agentscan.infoHighlight'),
            overviewRulerLane: vscode.OverviewRulerLane.Right,
            after: {
                contentText: ' 🔵',
                color: new vscode.ThemeColor('agentscan.infoHighlight')
            }
        }));
    }

    updateFindings(uri: vscode.Uri, findings: Finding[]) {
        const uriString = uri.toString();
        this.findingsMap.set(uriString, findings);

        // Convert findings to diagnostics
        const diagnostics: vscode.Diagnostic[] = findings.map(finding => {
            const line = Math.max(0, finding.lineNumber - 1); // VS Code uses 0-based line numbers
            const column = finding.columnNumber ? Math.max(0, finding.columnNumber - 1) : 0;
            
            const range = new vscode.Range(
                new vscode.Position(line, column),
                new vscode.Position(line, column + 10) // Highlight a few characters
            );

            const diagnostic = new vscode.Diagnostic(
                range,
                `${finding.title}: ${finding.description}`,
                this.getSeverityLevel(finding.severity)
            );

            diagnostic.source = `AgentScan (${finding.tool})`;
            diagnostic.code = finding.ruleId;
            
            // Add related information
            if (finding.references && finding.references.length > 0) {
                diagnostic.relatedInformation = finding.references.map(ref => 
                    new vscode.DiagnosticRelatedInformation(
                        new vscode.Location(uri, range),
                        `Reference: ${ref}`
                    )
                );
            }

            // Store finding data for later use
            (diagnostic as any).agentScanFinding = finding;

            return diagnostic;
        });

        // Update diagnostics collection
        this.diagnosticCollection.set(uri, diagnostics);

        // Update decorations for active editor
        this.updateDecorations(uri);
    }

    private updateDecorations(uri: vscode.Uri) {
        const activeEditor = vscode.window.activeTextEditor;
        if (!activeEditor || activeEditor.document.uri.toString() !== uri.toString()) {
            return;
        }

        const findings = this.findingsMap.get(uri.toString()) || [];
        
        // Group findings by severity
        const findingsBySeverity: { [severity: string]: vscode.DecorationOptions[] } = {
            high: [],
            medium: [],
            low: []
        };

        findings.forEach(finding => {
            const line = Math.max(0, finding.lineNumber - 1);
            const column = finding.columnNumber ? Math.max(0, finding.columnNumber - 1) : 0;
            
            const range = new vscode.Range(
                new vscode.Position(line, column),
                new vscode.Position(line, column + 10)
            );

            const decoration: vscode.DecorationOptions = {
                range,
                hoverMessage: this.createHoverMessage(finding)
            };

            findingsBySeverity[finding.severity].push(decoration);
        });

        // Apply decorations
        Object.entries(findingsBySeverity).forEach(([severity, decorations]) => {
            const decorationType = this.decorationTypes.get(severity);
            if (decorationType) {
                activeEditor.setDecorations(decorationType, decorations);
            }
        });
    }

    private createHoverMessage(finding: Finding): vscode.MarkdownString {
        const markdown = new vscode.MarkdownString();
        markdown.isTrusted = true;

        // Title with severity badge
        const severityEmoji = finding.severity === 'high' ? '🔴' : finding.severity === 'medium' ? '🟡' : '🔵';
        markdown.appendMarkdown(`### ${severityEmoji} ${finding.title}\n\n`);

        // Description
        markdown.appendMarkdown(`**Description:** ${finding.description}\n\n`);

        // Details
        markdown.appendMarkdown(`**Tool:** ${finding.tool}\n\n`);
        markdown.appendMarkdown(`**Rule:** ${finding.ruleId}\n\n`);
        markdown.appendMarkdown(`**Severity:** ${finding.severity.toUpperCase()}\n\n`);
        markdown.appendMarkdown(`**Confidence:** ${(finding.confidence * 100).toFixed(1)}%\n\n`);

        if (finding.consensusScore) {
            markdown.appendMarkdown(`**Consensus Score:** ${(finding.consensusScore * 100).toFixed(1)}%\n\n`);
        }

        // Code snippet
        if (finding.codeSnippet) {
            markdown.appendMarkdown(`**Code:**\n\`\`\`\n${finding.codeSnippet}\n\`\`\`\n\n`);
        }

        // Fix suggestion
        if (finding.fixSuggestion) {
            markdown.appendMarkdown(`**💡 Suggested Fix:** ${finding.fixSuggestion}\n\n`);
        }

        // References
        if (finding.references && finding.references.length > 0) {
            markdown.appendMarkdown(`**References:**\n`);
            finding.references.forEach(ref => {
                markdown.appendMarkdown(`- [${ref}](${ref})\n`);
            });
            markdown.appendMarkdown('\n');
        }

        // Actions
        markdown.appendMarkdown(`---\n`);
        markdown.appendMarkdown(`[Suppress Finding](command:agentscan.suppressFinding?${encodeURIComponent(JSON.stringify(finding))}) | `);
        markdown.appendMarkdown(`[Mark as Fixed](command:agentscan.markAsFixed?${encodeURIComponent(JSON.stringify(finding))})`);

        return markdown;
    }

    private getSeverityLevel(severity: string): vscode.DiagnosticSeverity {
        switch (severity.toLowerCase()) {
            case 'high':
                return vscode.DiagnosticSeverity.Error;
            case 'medium':
                return vscode.DiagnosticSeverity.Warning;
            case 'low':
                return vscode.DiagnosticSeverity.Information;
            default:
                return vscode.DiagnosticSeverity.Hint;
        }
    }

    clearFindings(uri: vscode.Uri) {
        const uriString = uri.toString();
        this.findingsMap.delete(uriString);
        this.diagnosticCollection.delete(uri);
        
        // Clear decorations
        const activeEditor = vscode.window.activeTextEditor;
        if (activeEditor && activeEditor.document.uri.toString() === uriString) {
            this.decorationTypes.forEach(decorationType => {
                activeEditor.setDecorations(decorationType, []);
            });
        }
    }

    clearAll() {
        this.findingsMap.clear();
        this.diagnosticCollection.clear();
        
        // Clear all decorations
        const activeEditor = vscode.window.activeTextEditor;
        if (activeEditor) {
            this.decorationTypes.forEach(decorationType => {
                activeEditor.setDecorations(decorationType, []);
            });
        }
    }

    hasFindings(): boolean {
        return this.findingsMap.size > 0;
    }

    hasFindingAtPosition(uri: vscode.Uri, position: vscode.Position): boolean {
        const findings = this.findingsMap.get(uri.toString()) || [];
        return findings.some(finding => {
            const line = Math.max(0, finding.lineNumber - 1);
            return line === position.line;
        });
    }

    getFindingAtPosition(uri: vscode.Uri, position: vscode.Position): Finding | undefined {
        const findings = this.findingsMap.get(uri.toString()) || [];
        return findings.find(finding => {
            const line = Math.max(0, finding.lineNumber - 1);
            return line === position.line;
        });
    }

    getAllFindings(): Finding[] {
        const allFindings: Finding[] = [];
        this.findingsMap.forEach(findings => {
            allFindings.push(...findings);
        });
        return allFindings;
    }

    getFindingsForFile(uri: vscode.Uri): Finding[] {
        return this.findingsMap.get(uri.toString()) || [];
    }

    dispose() {
        this.diagnosticCollection.dispose();
        this.decorationTypes.forEach(decorationType => {
            decorationType.dispose();
        });
        this.decorationTypes.clear();
        this.findingsMap.clear();
    }

    // Event handler for when active editor changes
    onDidChangeActiveTextEditor(editor: vscode.TextEditor | undefined) {
        if (editor) {
            this.updateDecorations(editor.document.uri);
        }
    }
}