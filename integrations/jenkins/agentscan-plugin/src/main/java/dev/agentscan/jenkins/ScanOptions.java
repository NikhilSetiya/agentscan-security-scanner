package dev.agentscan.jenkins;

/**
 * Configuration options for AgentScan security scanning.
 */
public class ScanOptions {
    
    private String failOnSeverity = "high";
    private String excludePaths = "";
    private String includePaths = "";
    private String outputFormat = "json,sarif";
    private boolean uploadSarif = true;
    private boolean generateReport = true;
    private int timeoutMinutes = 30;
    
    public String getFailOnSeverity() {
        return failOnSeverity;
    }
    
    public void setFailOnSeverity(String failOnSeverity) {
        this.failOnSeverity = failOnSeverity;
    }
    
    public String getExcludePaths() {
        return excludePaths;
    }
    
    public void setExcludePaths(String excludePaths) {
        this.excludePaths = excludePaths;
    }
    
    public String getIncludePaths() {
        return includePaths;
    }
    
    public void setIncludePaths(String includePaths) {
        this.includePaths = includePaths;
    }
    
    public String getOutputFormat() {
        return outputFormat;
    }
    
    public void setOutputFormat(String outputFormat) {
        this.outputFormat = outputFormat;
    }
    
    public boolean isUploadSarif() {
        return uploadSarif;
    }
    
    public void setUploadSarif(boolean uploadSarif) {
        this.uploadSarif = uploadSarif;
    }
    
    public boolean isGenerateReport() {
        return generateReport;
    }
    
    public void setGenerateReport(boolean generateReport) {
        this.generateReport = generateReport;
    }
    
    public int getTimeoutMinutes() {
        return timeoutMinutes;
    }
    
    public void setTimeoutMinutes(int timeoutMinutes) {
        this.timeoutMinutes = timeoutMinutes;
    }
}