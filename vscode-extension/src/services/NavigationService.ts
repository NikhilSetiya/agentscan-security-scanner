import * as vscode from 'vscode';
import { Finding } from './ApiClient';
import { DiagnosticsManager } from './DiagnosticsManager';

export class NavigationService {
    private diagnosticsManager: DiagnosticsManager;
    private currentFindingIndex: number = -1;
    private currentFindings: Finding[] = [];

    constructor(diagnosticsManager: DiagnosticsManager) {
        this.diagnosticsManager = diagnosticsManager;
    }

    /**
     * Navigate to the next finding in the current file or workspace
     */
    async goToNextFinding(): Promise<void> {
        const findings = this.getAllFindings();
        if (findings.length === 0) {
            vscode.window.showInformationMessage('No security findings to navigate to');
            return;
        }

        this.currentFindings = findings;
        this.currentFindingIndex = (this.currentFindingIndex + 1) % findings.length;
        
        await this.navigateToFinding(findings[this.currentFindingIndex]);
        this.showNavigationStatus();
    }

    /**
     * Navigate to the previous finding in the current file or workspace
     */
    async goToPreviousFinding(): Promise<void> {
        const findings = this.getAllFindings();
        if (findings.length === 0) {
            vscode.window.showInformationMessage('No security findings to navigate to');
            return;
        }

        this.currentFindings = findings;
        this.currentFindingIndex = this.currentFindingIndex <= 0 
            ? findings.length - 1 
            : this.currentFindingIndex - 1;
        
        await this.navigateToFinding(findings[this.currentFindingIndex]);
        this.showNavigationStatus();
    }

    /**
     * Navigate to a specific finding
     */
    async navigateToFinding(finding: Finding): Promise<void> {
        try {
            const uri = vscode.Uri.file(finding.filePath);
            
            // Check if file exists and is accessible
            try {
                await vscode.workspace.fs.stat(uri);
            } catch (error) {
                vscode.window.showErrorMessage(`File not found: ${finding.filePath}`);
                return;
            }

            // Open the document
            const document = await vscode.workspace.openTextDocument(uri);
            const editor = await vscode.window.showTextDocument(document, {
                selection: this.createSelectionRange(finding),
                viewColumn: vscode.ViewColumn.Active
            });

            // Highlight the finding
            this.highlightFinding(editor, finding);

            // Show finding details in a hover-like popup
            this.showFindingDetails(finding);

        } catch (error) {
            console.error('Failed to navigate to finding:', error);
            vscode.window.showErrorMessage(`Failed to navigate to finding: ${error}`);
        }
    }

    /**
     * Get the finding at the current cursor position
     */
    getFindingAtCursor(): Finding | null {
        const activeEditor = vscode.window.activeTextEditor;
        if (!activeEditor) {
            return null;
        }

        const position = activeEditor.selection.active;
        return this.diagnosticsManager.getFindingAtPosition(activeEditor.document.uri, position) || null;
    }

    /**
     * Get all findings sorted by severity and file location
     */
    private getAllFindings(): Finding[] {
        const allFindings = this.diagnosticsManager.getAllFindings();
        
        // Sort by severity (high -> medium -> low) then by file path and line number
        return allFindings.sort((a, b) => {
            // First sort by severity
            const severityOrder = { high: 3, medium: 2, low: 1 };
            const severityDiff = (severityOrder[b.severity as keyof typeof severityOrder] || 0) - 
                               (severityOrder[a.severity as keyof typeof severityOrder] || 0);
            
            if (severityDiff !== 0) {
                return severityDiff;
            }

            // Then sort by file path
            const pathDiff = a.filePath.localeCompare(b.filePath);
            if (pathDiff !== 0) {
                return pathDiff;
            }

            // Finally sort by line number
            return a.lineNumber - b.lineNumber;
        });
    }

    /**
     * Create a selection range for a finding
     */
    private createSelectionRange(finding: Finding): vscode.Range {
        const line = Math.max(0, finding.lineNumber - 1);
        const column = Math.max(0, (finding.columnNumber || 1) - 1);
        
        // Try to select the entire problematic code if available
        let endColumn = column + 10; // Default selection length
        
        if (finding.codeSnippet) {
            // If we have the code snippet, try to select the actual problematic part
            const lines = finding.codeSnippet.split('\n');
            if (lines.length > 0) {
                endColumn = column + lines[0].length;
            }
        }

        const startPosition = new vscode.Position(line, column);
        const endPosition = new vscode.Position(line, endColumn);
        
        return new vscode.Range(startPosition, endPosition);
    }

    /**
     * Highlight the finding in the editor temporarily
     */
    private highlightFinding(editor: vscode.TextEditor, finding: Finding): void {
        const range = this.createSelectionRange(finding);
        
        // Create a temporary decoration type for highlighting
        const decorationType = vscode.window.createTextEditorDecorationType({
            backgroundColor: new vscode.ThemeColor('editor.findMatchHighlightBackground'),
            border: '2px solid',
            borderColor: new vscode.ThemeColor('editor.findMatchHighlightBorder'),
            borderRadius: '3px'
        });

        // Apply the decoration
        editor.setDecorations(decorationType, [range]);

        // Remove the decoration after 3 seconds
        setTimeout(() => {
            decorationType.dispose();
        }, 3000);

        // Reveal the range in the center of the editor
        editor.revealRange(range, vscode.TextEditorRevealType.InCenter);
    }

    /**
     * Show finding details in a status bar message
     */
    private showFindingDetails(finding: Finding): void {
        const severityEmoji = finding.severity === 'high' ? '🔴' : 
                             finding.severity === 'medium' ? '🟡' : '🔵';
        
        const message = `${severityEmoji} ${finding.title} (${finding.tool})`;
        
        vscode.window.setStatusBarMessage(message, 5000);
    }

    /**
     * Show navigation status in the status bar
     */
    private showNavigationStatus(): void {
        if (this.currentFindings.length === 0) {
            return;
        }

        const current = this.currentFindingIndex + 1;
        const total = this.currentFindings.length;
        const message = `Finding ${current} of ${total}`;
        
        vscode.window.setStatusBarMessage(message, 3000);
    }

    /**
     * Get findings for the current file only
     */
    getCurrentFileFindings(): Finding[] {
        const activeEditor = vscode.window.activeTextEditor;
        if (!activeEditor) {
            return [];
        }

        return this.diagnosticsManager.getFindingsForFile(activeEditor.document.uri);
    }

    /**
     * Navigate to next finding in current file only
     */
    async goToNextFindingInFile(): Promise<void> {
        const findings = this.getCurrentFileFindings();
        if (findings.length === 0) {
            vscode.window.showInformationMessage('No security findings in current file');
            return;
        }

        const activeEditor = vscode.window.activeTextEditor;
        if (!activeEditor) {
            return;
        }

        const currentLine = activeEditor.selection.active.line;
        
        // Find the next finding after the current cursor position
        const nextFinding = findings.find(f => (f.lineNumber - 1) > currentLine) || findings[0];
        
        await this.navigateToFinding(nextFinding);
    }

    /**
     * Navigate to previous finding in current file only
     */
    async goToPreviousFindingInFile(): Promise<void> {
        const findings = this.getCurrentFileFindings();
        if (findings.length === 0) {
            vscode.window.showInformationMessage('No security findings in current file');
            return;
        }

        const activeEditor = vscode.window.activeTextEditor;
        if (!activeEditor) {
            return;
        }

        const currentLine = activeEditor.selection.active.line;
        
        // Find the previous finding before the current cursor position
        const reversedFindings = [...findings].reverse();
        const previousFinding = reversedFindings.find(f => (f.lineNumber - 1) < currentLine) || 
                               reversedFindings[0];
        
        await this.navigateToFinding(previousFinding);
    }

    /**
     * Reset navigation state
     */
    reset(): void {
        this.currentFindingIndex = -1;
        this.currentFindings = [];
    }

    /**
     * Get navigation statistics
     */
    getNavigationStats(): { totalFindings: number; currentIndex: number; hasFindings: boolean } {
        return {
            totalFindings: this.currentFindings.length,
            currentIndex: this.currentFindingIndex,
            hasFindings: this.currentFindings.length > 0
        };
    }
}