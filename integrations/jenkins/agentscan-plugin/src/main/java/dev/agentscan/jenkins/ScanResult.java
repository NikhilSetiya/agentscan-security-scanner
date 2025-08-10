package dev.agentscan.jenkins;

import java.util.List;
import java.util.Map;

/**
 * Represents the result of an AgentScan security scan.
 */
public class ScanResult {
    
    private final boolean success;
    private final String errorMessage;
    private final Map<String, Object> resultsData;
    private final ScanSummary summary;
    
    private ScanResult(boolean success, String errorMessage, Map<String, Object> resultsData) {
        this.success = success;
        this.errorMessage = errorMessage;
        this.resultsData = resultsData;
        this.summary = success ? parseSummary(resultsData) : null;
    }
    
    public static ScanResult success(Map<String, Object> resultsData) {
        return new ScanResult(true, null, resultsData);
    }
    
    public static ScanResult failure(String errorMessage) {
        return new ScanResult(false, errorMessage, null);
    }
    
    public boolean isSuccess() {
        return success;
    }
    
    public String getErrorMessage() {
        return errorMessage;
    }
    
    public Map<String, Object> getResultsData() {
        return resultsData;
    }
    
    public ScanSummary getSummary() {
        return summary;
    }
    
    @SuppressWarnings("unchecked")
    private ScanSummary parseSummary(Map<String, Object> data) {
        if (data == null) {
            return new ScanSummary(0, 0, 0, 0);
        }
        
        Map<String, Object> summary = (Map<String, Object>) data.get("summary");
        if (summary == null) {
            // Try to count findings directly
            List<Map<String, Object>> findings = (List<Map<String, Object>>) data.get("findings");
            if (findings != null) {
                return countFindingsBySeverity(findings);
            }
            return new ScanSummary(0, 0, 0, 0);
        }
        
        int totalFindings = getIntValue(summary, "total_findings");
        
        Map<String, Object> bySeverity = (Map<String, Object>) summary.get("by_severity");
        if (bySeverity != null) {
            int high = getIntValue(bySeverity, "high");
            int medium = getIntValue(bySeverity, "medium");
            int low = getIntValue(bySeverity, "low");
            return new ScanSummary(totalFindings, high, medium, low);
        }
        
        return new ScanSummary(totalFindings, 0, 0, 0);
    }
    
    @SuppressWarnings("unchecked")
    private ScanSummary countFindingsBySeverity(List<Map<String, Object>> findings) {
        int high = 0, medium = 0, low = 0;
        
        for (Map<String, Object> finding : findings) {
            String severity = (String) finding.get("severity");
            if ("high".equals(severity)) {
                high++;
            } else if ("medium".equals(severity)) {
                medium++;
            } else if ("low".equals(severity)) {
                low++;
            }
        }
        
        return new ScanSummary(findings.size(), high, medium, low);
    }
    
    private int getIntValue(Map<String, Object> map, String key) {
        Object value = map.get(key);
        if (value instanceof Integer) {
            return (Integer) value;
        } else if (value instanceof String) {
            try {
                return Integer.parseInt((String) value);
            } catch (NumberFormatException e) {
                return 0;
            }
        }
        return 0;
    }
}