package dev.agentscan.jenkins;

import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.List;
import java.util.Map;

/**
 * Generates HTML reports for AgentScan security scan results.
 */
public class HtmlReportGenerator {
    
    private static final SimpleDateFormat DATE_FORMAT = new SimpleDateFormat("yyyy-MM-dd HH:mm:ss");
    
    public static String generateReport(ScanResult result) {
        if (!result.isSuccess()) {
            return generateErrorReport(result.getErrorMessage());
        }
        
        StringBuilder html = new StringBuilder();
        
        // HTML header
        html.append(getHtmlHeader());
        
        // Report content
        html.append(generateReportContent(result));
        
        // HTML footer
        html.append(getHtmlFooter());
        
        return html.toString();
    }
    
    private static String getHtmlHeader() {
        return """
            <!DOCTYPE html>
            <html lang="en">
            <head>
                <meta charset="UTF-8">
                <meta name="viewport" content="width=device-width, initial-scale=1.0">
                <title>AgentScan Security Report</title>
                <style>
                    body {
                        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
                        line-height: 1.6;
                        color: #24292f;
                        max-width: 1200px;
                        margin: 0 auto;
                        padding: 20px;
                        background-color: #ffffff;
                    }
                    .header {
                        border-bottom: 2px solid #d0d7de;
                        padding-bottom: 20px;
                        margin-bottom: 30px;
                    }
                    .header h1 {
                        margin: 0;
                        color: #1f2328;
                        font-size: 32px;
                        font-weight: 600;
                    }
                    .header .subtitle {
                        color: #656d76;
                        font-size: 16px;
                        margin-top: 8px;
                    }
                    .summary {
                        display: grid;
                        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
                        gap: 20px;
                        margin-bottom: 40px;
                    }
                    .metric {
                        background: #f6f8fa;
                        border: 1px solid #d0d7de;
                        border-radius: 8px;
                        padding: 20px;
                        text-align: center;
                    }
                    .metric-value {
                        font-size: 36px;
                        font-weight: 700;
                        margin-bottom: 8px;
                        line-height: 1;
                    }
                    .metric-label {
                        color: #656d76;
                        font-size: 14px;
                        font-weight: 500;
                        text-transform: uppercase;
                        letter-spacing: 0.5px;
                    }
                    .severity-high { color: #d1242f; }
                    .severity-medium { color: #fb8500; }
                    .severity-low { color: #1f883d; }
                    .severity-info { color: #0969da; }
                    .findings-section {
                        margin-top: 40px;
                    }
                    .findings-section h2 {
                        color: #1f2328;
                        font-size: 24px;
                        font-weight: 600;
                        margin-bottom: 20px;
                        border-bottom: 1px solid #d0d7de;
                        padding-bottom: 10px;
                    }
                    .finding {
                        background: #ffffff;
                        border: 1px solid #d0d7de;
                        border-radius: 8px;
                        margin-bottom: 16px;
                        overflow: hidden;
                    }
                    .finding-header {
                        background: #f6f8fa;
                        padding: 16px 20px;
                        border-bottom: 1px solid #d0d7de;
                        display: flex;
                        justify-content: space-between;
                        align-items: center;
                    }
                    .finding-title {
                        font-size: 16px;
                        font-weight: 600;
                        color: #1f2328;
                        margin: 0;
                    }
                    .severity-badge {
                        padding: 4px 12px;
                        border-radius: 20px;
                        font-size: 12px;
                        font-weight: 600;
                        text-transform: uppercase;
                        letter-spacing: 0.5px;
                    }
                    .severity-badge.high {
                        background: #ffebe9;
                        color: #d1242f;
                        border: 1px solid #ffcdd2;
                    }
                    .severity-badge.medium {
                        background: #fff8dc;
                        color: #fb8500;
                        border: 1px solid #ffe0b3;
                    }
                    .severity-badge.low {
                        background: #dafbe1;
                        color: #1f883d;
                        border: 1px solid #b3f0c0;
                    }
                    .finding-body {
                        padding: 20px;
                    }
                    .finding-meta {
                        display: grid;
                        grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
                        gap: 16px;
                        margin-bottom: 16px;
                    }
                    .meta-item {
                        display: flex;
                        flex-direction: column;
                    }
                    .meta-label {
                        font-size: 12px;
                        font-weight: 600;
                        color: #656d76;
                        text-transform: uppercase;
                        letter-spacing: 0.5px;
                        margin-bottom: 4px;
                    }
                    .meta-value {
                        font-size: 14px;
                        color: #1f2328;
                        font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, monospace;
                    }
                    .finding-description {
                        color: #656d76;
                        font-size: 14px;
                        line-height: 1.5;
                        margin-bottom: 16px;
                    }
                    .code-snippet {
                        background: #f6f8fa;
                        border: 1px solid #d0d7de;
                        border-radius: 6px;
                        padding: 16px;
                        font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, monospace;
                        font-size: 13px;
                        line-height: 1.4;
                        overflow-x: auto;
                        white-space: pre-wrap;
                    }
                    .no-findings {
                        text-align: center;
                        padding: 60px 20px;
                        background: #f6f8fa;
                        border-radius: 8px;
                        border: 1px solid #d0d7de;
                    }
                    .no-findings h3 {
                        color: #1f883d;
                        font-size: 24px;
                        margin-bottom: 12px;
                    }
                    .no-findings p {
                        color: #656d76;
                        font-size: 16px;
                        margin: 0;
                    }
                    .footer {
                        margin-top: 60px;
                        padding-top: 20px;
                        border-top: 1px solid #d0d7de;
                        text-align: center;
                        color: #656d76;
                        font-size: 14px;
                    }
                </style>
            </head>
            <body>
            """;
    }
    
    private static String getHtmlFooter() {
        return """
                <div class="footer">
                    <p>Generated by <strong>AgentScan</strong> - Multi-agent security scanning platform</p>
                    <p>Report generated on """ + DATE_FORMAT.format(new Date()) + """</p>
                </div>
            </body>
            </html>
            """;
    }
    
    @SuppressWarnings("unchecked")
    private static String generateReportContent(ScanResult result) {
        StringBuilder content = new StringBuilder();
        
        // Header
        content.append("""
            <div class="header">
                <h1>🔒 Security Scan Report</h1>
                <div class="subtitle">Comprehensive security analysis results</div>
            </div>
            """);
        
        // Summary metrics
        ScanSummary summary = result.getSummary();
        if (summary != null) {
            content.append("""
                <div class="summary">
                    <div class="metric">
                        <div class="metric-value">""").append(summary.getTotalFindings()).append("""</div>
                        <div class="metric-label">Total Findings</div>
                    </div>
                    <div class="metric">
                        <div class="metric-value severity-high">""").append(summary.getHighSeverityCount()).append("""</div>
                        <div class="metric-label">High Severity</div>
                    </div>
                    <div class="metric">
                        <div class="metric-value severity-medium">""").append(summary.getMediumSeverityCount()).append("""</div>
                        <div class="metric-label">Medium Severity</div>
                    </div>
                    <div class="metric">
                        <div class="metric-value severity-low">""").append(summary.getLowSeverityCount()).append("""</div>
                        <div class="metric-label">Low Severity</div>
                    </div>
                </div>
                """);
        }
        
        // Findings details
        Map<String, Object> resultsData = result.getResultsData();
        if (resultsData != null) {
            List<Map<String, Object>> findings = (List<Map<String, Object>>) resultsData.get("findings");
            
            if (findings != null && !findings.isEmpty()) {
                content.append("""
                    <div class="findings-section">
                        <h2>Security Findings</h2>
                    """);
                
                for (Map<String, Object> finding : findings) {
                    content.append(generateFindingHtml(finding));
                }
                
                content.append("</div>");
            } else {
                content.append("""
                    <div class="no-findings">
                        <h3>✅ No Security Issues Found</h3>
                        <p>Excellent! Your code appears to be secure based on our analysis.</p>
                    </div>
                    """);
            }
        }
        
        return content.toString();
    }
    
    private static String generateFindingHtml(Map<String, Object> finding) {
        String title = (String) finding.getOrDefault("title", "Security Issue");
        String severity = (String) finding.getOrDefault("severity", "info");
        String description = (String) finding.getOrDefault("description", "No description available");
        String filePath = (String) finding.getOrDefault("file_path", "unknown");
        String lineNumber = String.valueOf(finding.getOrDefault("line_number", "0"));
        String tool = (String) finding.getOrDefault("tool", "unknown");
        String ruleId = (String) finding.getOrDefault("rule_id", "");
        String codeSnippet = (String) finding.get("code_snippet");
        
        StringBuilder html = new StringBuilder();
        
        html.append("""
            <div class="finding">
                <div class="finding-header">
                    <h3 class="finding-title">""").append(escapeHtml(title)).append("""</h3>
                    <span class="severity-badge """).append(severity).append("""">""").append(severity.toUpperCase()).append("""</span>
                </div>
                <div class="finding-body">
                    <div class="finding-meta">
                        <div class="meta-item">
                            <div class="meta-label">File</div>
                            <div class="meta-value">""").append(escapeHtml(filePath)).append(":").append(lineNumber).append("""</div>
                        </div>
                        <div class="meta-item">
                            <div class="meta-label">Tool</div>
                            <div class="meta-value">""").append(escapeHtml(tool)).append("""</div>
                        </div>
            """);
        
        if (ruleId != null && !ruleId.isEmpty()) {
            html.append("""
                        <div class="meta-item">
                            <div class="meta-label">Rule ID</div>
                            <div class="meta-value">""").append(escapeHtml(ruleId)).append("""</div>
                        </div>
                """);
        }
        
        html.append("""
                    </div>
                    <div class="finding-description">""").append(escapeHtml(description)).append("""</div>
            """);
        
        if (codeSnippet != null && !codeSnippet.isEmpty()) {
            html.append("""
                    <div class="code-snippet">""").append(escapeHtml(codeSnippet)).append("""</div>
                """);
        }
        
        html.append("""
                </div>
            </div>
            """);
        
        return html.toString();
    }
    
    private static String generateErrorReport(String errorMessage) {
        return getHtmlHeader() + """
            <div class="header">
                <h1>❌ Security Scan Failed</h1>
                <div class="subtitle">An error occurred during the security scan</div>
            </div>
            <div style="background: #ffebe9; border: 1px solid #ffcdd2; border-radius: 8px; padding: 20px; margin: 20px 0;">
                <h3 style="color: #d1242f; margin-top: 0;">Error Details</h3>
                <p style="color: #656d76; margin-bottom: 0;">""" + escapeHtml(errorMessage) + """</p>
            </div>
            """ + getHtmlFooter();
    }
    
    private static String escapeHtml(String text) {
        if (text == null) {
            return "";
        }
        return text.replace("&", "&amp;")
                  .replace("<", "&lt;")
                  .replace(">", "&gt;")
                  .replace("\"", "&quot;")
                  .replace("'", "&#x27;");
    }
}