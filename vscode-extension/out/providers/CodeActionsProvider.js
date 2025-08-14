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
exports.FindingHoverProvider = exports.CodeActionsProvider = void 0;
const vscode = __importStar(require("vscode"));
class CodeActionsProvider {
    constructor(diagnosticsManager, telemetryService) {
        this.diagnosticsManager = diagnosticsManager;
        this.telemetryService = telemetryService;
    }
    provideCodeActions(document, range, context, token) {
        const actions = [];
        // Get findings at the current position
        const findings = this.getFindingsInRange(document.uri, range);
        for (const finding of findings) {
            // Add suppress finding action
            actions.push(this.createSuppressFindingAction(finding, document, range));
            // Add mark as fixed action
            actions.push(this.createMarkAsFixedAction(finding, document, range));
            // Add ignore rule action
            actions.push(this.createIgnoreRuleAction(finding, document, range));
            // Add learn more action
            actions.push(this.createLearnMoreAction(finding, document, range));
            // Add fix suggestion action if available
            if (finding.fixSuggestion) {
                actions.push(this.createApplyFixAction(finding, document, range));
            }
        }
        return actions;
    }
    getFindingsInRange(uri, range) {
        const allFindings = this.diagnosticsManager.getFindingsForFile(uri);
        return allFindings.filter(finding => {
            const findingLine = Math.max(0, finding.lineNumber - 1);
            return range.start.line <= findingLine && findingLine <= range.end.line;
        });
    }
    createSuppressFindingAction(finding, document, range) {
        const action = new vscode.CodeAction(`Suppress "${finding.title}"`, vscode.CodeActionKind.QuickFix);
        action.command = {
            command: 'agentscan.suppressFinding',
            title: 'Suppress Finding',
            arguments: [finding]
        };
        action.diagnostics = this.getDiagnosticsForFinding(finding, document.uri);
        action.isPreferred = false;
        return action;
    }
    createMarkAsFixedAction(finding, document, range) {
        const action = new vscode.CodeAction(`Mark "${finding.title}" as fixed`, vscode.CodeActionKind.QuickFix);
        action.command = {
            command: 'agentscan.markAsFixed',
            title: 'Mark as Fixed',
            arguments: [finding]
        };
        action.diagnostics = this.getDiagnosticsForFinding(finding, document.uri);
        action.isPreferred = false;
        return action;
    }
    createIgnoreRuleAction(finding, document, range) {
        const action = new vscode.CodeAction(`Ignore rule "${finding.ruleId}"`, vscode.CodeActionKind.QuickFix);
        action.command = {
            command: 'agentscan.ignoreRule',
            title: 'Ignore Rule',
            arguments: [finding]
        };
        action.diagnostics = this.getDiagnosticsForFinding(finding, document.uri);
        action.isPreferred = false;
        return action;
    }
    createLearnMoreAction(finding, document, range) {
        const action = new vscode.CodeAction(`Learn more about "${finding.title}"`, vscode.CodeActionKind.QuickFix);
        action.command = {
            command: 'agentscan.learnMore',
            title: 'Learn More',
            arguments: [finding]
        };
        action.diagnostics = this.getDiagnosticsForFinding(finding, document.uri);
        action.isPreferred = false;
        return action;
    }
    createApplyFixAction(finding, document, range) {
        const action = new vscode.CodeAction(`Apply fix: ${finding.fixSuggestion}`, vscode.CodeActionKind.QuickFix);
        // Create workspace edit to apply the fix
        const edit = new vscode.WorkspaceEdit();
        // This is a simplified implementation - in reality, you'd need more sophisticated
        // parsing to apply the fix correctly
        const fixRange = new vscode.Range(new vscode.Position(Math.max(0, finding.lineNumber - 1), 0), new vscode.Position(Math.max(0, finding.lineNumber - 1), Number.MAX_SAFE_INTEGER));
        // For demonstration, we'll add a comment with the fix suggestion
        const fixText = `// AgentScan Fix: ${finding.fixSuggestion}\n${document.lineAt(finding.lineNumber - 1).text}`;
        edit.replace(document.uri, fixRange, fixText);
        action.edit = edit;
        action.diagnostics = this.getDiagnosticsForFinding(finding, document.uri);
        action.isPreferred = true; // Make fix suggestions preferred
        return action;
    }
    getDiagnosticsForFinding(finding, uri) {
        // Get all diagnostics for the document
        const diagnostics = vscode.languages.getDiagnostics(uri);
        // Filter to find diagnostics that match this finding
        return diagnostics.filter(diagnostic => {
            const agentScanFinding = diagnostic.agentScanFinding;
            return agentScanFinding && agentScanFinding.id === finding.id;
        });
    }
}
exports.CodeActionsProvider = CodeActionsProvider;
/**
 * Hover provider to show rich tooltips for findings
 */
class FindingHoverProvider {
    constructor(diagnosticsManager, telemetryService) {
        this.diagnosticsManager = diagnosticsManager;
        this.telemetryService = telemetryService;
    }
    provideHover(document, position, token) {
        const finding = this.diagnosticsManager.getFindingAtPosition(document.uri, position);
        if (!finding) {
            return null;
        }
        // Track hover event
        this.telemetryService.trackUserAction('hover.finding', {
            severity: finding.severity,
            tool: finding.tool,
            ruleId: finding.ruleId
        });
        const markdown = this.createRichHoverContent(finding);
        // Create hover range
        const line = Math.max(0, finding.lineNumber - 1);
        const startColumn = Math.max(0, (finding.columnNumber || 1) - 1);
        const endColumn = startColumn + (finding.codeSnippet?.length || 10);
        const range = new vscode.Range(new vscode.Position(line, startColumn), new vscode.Position(line, endColumn));
        return new vscode.Hover(markdown, range);
    }
    createRichHoverContent(finding) {
        const markdown = new vscode.MarkdownString();
        markdown.isTrusted = true;
        markdown.supportHtml = true;
        // Header with severity badge
        const severityEmoji = finding.severity === 'high' ? '🔴' :
            finding.severity === 'medium' ? '🟡' : '🔵';
        const severityColor = finding.severity === 'high' ? '#dc2626' :
            finding.severity === 'medium' ? '#d97706' : '#2563eb';
        markdown.appendMarkdown(`### ${severityEmoji} ${finding.title}\n\n`);
        // Severity badge
        markdown.appendMarkdown(`<span style="background-color: ${severityColor}; color: white; padding: 2px 6px; border-radius: 3px; font-size: 11px; font-weight: bold;">${finding.severity.toUpperCase()}</span>\n\n`);
        // Description
        markdown.appendMarkdown(`**Description:** ${finding.description}\n\n`);
        // Tool and rule information
        markdown.appendMarkdown(`**Tool:** ${finding.tool} | **Rule:** \`${finding.ruleId}\`\n\n`);
        // Confidence and consensus scores
        markdown.appendMarkdown(`**Confidence:** ${(finding.confidence * 100).toFixed(1)}%`);
        if (finding.consensusScore) {
            markdown.appendMarkdown(` | **Consensus:** ${(finding.consensusScore * 100).toFixed(1)}%`);
        }
        markdown.appendMarkdown('\n\n');
        // Code snippet with syntax highlighting
        if (finding.codeSnippet) {
            const language = this.detectLanguage(finding.filePath);
            markdown.appendMarkdown(`**Code:**\n\`\`\`${language}\n${finding.codeSnippet}\n\`\`\`\n\n`);
        }
        // Fix suggestion
        if (finding.fixSuggestion) {
            markdown.appendMarkdown(`💡 **Suggested Fix:** ${finding.fixSuggestion}\n\n`);
        }
        // References
        if (finding.references && finding.references.length > 0) {
            markdown.appendMarkdown(`**References:**\n`);
            finding.references.slice(0, 3).forEach(ref => {
                const displayUrl = ref.length > 50 ? ref.substring(0, 47) + '...' : ref;
                markdown.appendMarkdown(`- [${displayUrl}](${ref})\n`);
            });
            if (finding.references.length > 3) {
                markdown.appendMarkdown(`- *...and ${finding.references.length - 3} more*\n`);
            }
            markdown.appendMarkdown('\n');
        }
        // Action buttons
        markdown.appendMarkdown(`---\n`);
        markdown.appendMarkdown(`[$(eye-closed) Suppress](command:agentscan.suppressFinding?${encodeURIComponent(JSON.stringify(finding))}) | ` +
            `[$(check) Mark Fixed](command:agentscan.markAsFixed?${encodeURIComponent(JSON.stringify(finding))}) | ` +
            `[$(exclude) Ignore Rule](command:agentscan.ignoreRule?${encodeURIComponent(JSON.stringify(finding))}) | ` +
            `[$(info) Learn More](command:agentscan.learnMore?${encodeURIComponent(JSON.stringify(finding))})`);
        return markdown;
    }
    detectLanguage(filePath) {
        const extension = filePath.split('.').pop()?.toLowerCase();
        switch (extension) {
            case 'js':
            case 'jsx':
                return 'javascript';
            case 'ts':
            case 'tsx':
                return 'typescript';
            case 'py':
                return 'python';
            case 'go':
                return 'go';
            case 'java':
                return 'java';
            case 'cs':
                return 'csharp';
            case 'cpp':
            case 'cc':
            case 'cxx':
                return 'cpp';
            case 'c':
                return 'c';
            case 'php':
                return 'php';
            case 'rb':
                return 'ruby';
            case 'rs':
                return 'rust';
            default:
                return 'text';
        }
    }
}
exports.FindingHoverProvider = FindingHoverProvider;
//# sourceMappingURL=CodeActionsProvider.js.map