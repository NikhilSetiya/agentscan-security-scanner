package dev.agentscan.jenkins;

/**
 * Summary statistics for a security scan.
 */
public class ScanSummary {
    
    private final int totalFindings;
    private final int highSeverityCount;
    private final int mediumSeverityCount;
    private final int lowSeverityCount;
    
    public ScanSummary(int totalFindings, int highSeverityCount, int mediumSeverityCount, int lowSeverityCount) {
        this.totalFindings = totalFindings;
        this.highSeverityCount = highSeverityCount;
        this.mediumSeverityCount = mediumSeverityCount;
        this.lowSeverityCount = lowSeverityCount;
    }
    
    public int getTotalFindings() {
        return totalFindings;
    }
    
    public int getHighSeverityCount() {
        return highSeverityCount;
    }
    
    public int getMediumSeverityCount() {
        return mediumSeverityCount;
    }
    
    public int getLowSeverityCount() {
        return lowSeverityCount;
    }
    
    public boolean hasHighSeverityFindings() {
        return highSeverityCount > 0;
    }
    
    public boolean hasMediumOrHighSeverityFindings() {
        return highSeverityCount > 0 || mediumSeverityCount > 0;
    }
    
    public boolean hasAnyFindings() {
        return totalFindings > 0;
    }
}