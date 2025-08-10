package dev.agentscan.jenkins;

import com.fasterxml.jackson.databind.ObjectMapper;
import hudson.FilePath;
import hudson.model.TaskListener;
import org.apache.http.HttpEntity;
import org.apache.http.HttpResponse;
import org.apache.http.client.methods.HttpPost;
import org.apache.http.entity.StringEntity;
import org.apache.http.impl.client.CloseableHttpClient;
import org.apache.http.impl.client.HttpClients;
import org.apache.http.util.EntityUtils;

import java.io.IOException;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.TimeUnit;

/**
 * Service class for interacting with the AgentScan API.
 */
public class AgentScanService {
    
    private final String apiUrl;
    private final String apiToken;
    private final TaskListener listener;
    private final ObjectMapper objectMapper;
    
    public AgentScanService(String apiUrl, String apiToken, TaskListener listener) {
        this.apiUrl = apiUrl;
        this.apiToken = apiToken;
        this.listener = listener;
        this.objectMapper = new ObjectMapper();
    }
    
    public ScanResult executeScan(FilePath workspace, ScanOptions options) {
        try {
            listener.getLogger().println("🔍 Submitting scan request to AgentScan API...");
            
            // Create scan request
            Map<String, Object> scanRequest = createScanRequest(workspace, options);
            
            // Submit scan to API
            String jobId = submitScan(scanRequest);
            if (jobId == null) {
                return ScanResult.failure("Failed to submit scan to AgentScan API");
            }
            
            listener.getLogger().println("📋 Scan submitted with job ID: " + jobId);
            
            // Poll for results
            ScanResult result = pollForResults(jobId, options.getTimeoutMinutes());
            
            // Save results to workspace
            if (result.isSuccess()) {
                saveResultsToWorkspace(workspace, result, options);
            }
            
            return result;
            
        } catch (Exception e) {
            listener.getLogger().println("❌ Error during scan execution: " + e.getMessage());
            return ScanResult.failure("Scan execution failed: " + e.getMessage());
        }
    }
    
    private Map<String, Object> createScanRequest(FilePath workspace, ScanOptions options) throws IOException, InterruptedException {
        Map<String, Object> request = new HashMap<>();
        
        // Basic scan configuration
        request.put("repo_url", detectRepositoryUrl(workspace));
        request.put("branch", detectBranch(workspace));
        request.put("commit_sha", detectCommitSha(workspace));
        request.put("scan_type", "full");
        request.put("priority", 5); // Medium priority
        
        // Scan options
        Map<String, Object> scanOptions = new HashMap<>();
        if (options.getExcludePaths() != null && !options.getExcludePaths().isEmpty()) {
            scanOptions.put("exclude_paths", options.getExcludePaths());
        }
        if (options.getIncludePaths() != null && !options.getIncludePaths().isEmpty()) {
            scanOptions.put("include_paths", options.getIncludePaths());
        }
        scanOptions.put("output_format", options.getOutputFormat());
        scanOptions.put("jenkins_build", true);
        
        request.put("options", scanOptions);
        
        return request;
    }
    
    private String detectRepositoryUrl(FilePath workspace) throws IOException, InterruptedException {
        // Try to detect Git repository URL
        FilePath gitConfig = workspace.child(".git/config");
        if (gitConfig.exists()) {
            String config = gitConfig.readToString();
            // Simple regex to extract origin URL
            String[] lines = config.split("\n");
            for (String line : lines) {
                if (line.trim().startsWith("url = ")) {
                    return line.trim().substring(6);
                }
            }
        }
        
        // Fallback to Jenkins environment variables
        String gitUrl = System.getenv("GIT_URL");
        if (gitUrl != null && !gitUrl.isEmpty()) {
            return gitUrl;
        }
        
        return "unknown-repository";
    }
    
    private String detectBranch(FilePath workspace) throws IOException, InterruptedException {
        // Try to detect current branch
        FilePath gitHead = workspace.child(".git/HEAD");
        if (gitHead.exists()) {
            String head = gitHead.readToString().trim();
            if (head.startsWith("ref: refs/heads/")) {
                return head.substring(16);
            }
        }
        
        // Fallback to Jenkins environment variables
        String branch = System.getenv("GIT_BRANCH");
        if (branch != null && !branch.isEmpty()) {
            // Remove origin/ prefix if present
            if (branch.startsWith("origin/")) {
                return branch.substring(7);
            }
            return branch;
        }
        
        return "main";
    }
    
    private String detectCommitSha(FilePath workspace) throws IOException, InterruptedException {
        // Try to detect current commit SHA
        FilePath gitHead = workspace.child(".git/HEAD");
        if (gitHead.exists()) {
            String head = gitHead.readToString().trim();
            if (!head.startsWith("ref:")) {
                return head; // Direct SHA
            } else {
                // Read from refs file
                String refPath = head.substring(5); // Remove "ref: "
                FilePath refFile = workspace.child(".git/" + refPath);
                if (refFile.exists()) {
                    return refFile.readToString().trim();
                }
            }
        }
        
        // Fallback to Jenkins environment variables
        String commit = System.getenv("GIT_COMMIT");
        if (commit != null && !commit.isEmpty()) {
            return commit;
        }
        
        return "unknown-commit";
    }
    
    private String submitScan(Map<String, Object> scanRequest) throws IOException {
        try (CloseableHttpClient httpClient = HttpClients.createDefault()) {
            HttpPost post = new HttpPost(apiUrl + "/api/v1/scans");
            
            // Set headers
            post.setHeader("Content-Type", "application/json");
            if (apiToken != null && !apiToken.isEmpty()) {
                post.setHeader("Authorization", "Bearer " + apiToken);
            }
            
            // Set request body
            String jsonBody = objectMapper.writeValueAsString(scanRequest);
            post.setEntity(new StringEntity(jsonBody));
            
            // Execute request
            HttpResponse response = httpClient.execute(post);
            HttpEntity entity = response.getEntity();
            String responseBody = EntityUtils.toString(entity);
            
            if (response.getStatusLine().getStatusCode() == 201) {
                // Parse response to get job ID
                Map<String, Object> responseMap = objectMapper.readValue(responseBody, Map.class);
                return (String) responseMap.get("job_id");
            } else {
                listener.getLogger().println("❌ API request failed: " + response.getStatusLine().getStatusCode());
                listener.getLogger().println("Response: " + responseBody);
                return null;
            }
        }
    }
    
    private ScanResult pollForResults(String jobId, int timeoutMinutes) throws IOException, InterruptedException {
        long startTime = System.currentTimeMillis();
        long timeoutMs = TimeUnit.MINUTES.toMillis(timeoutMinutes);
        
        try (CloseableHttpClient httpClient = HttpClients.createDefault()) {
            while (System.currentTimeMillis() - startTime < timeoutMs) {
                // Check scan status
                HttpPost statusRequest = new HttpPost(apiUrl + "/api/v1/scans/" + jobId + "/status");
                if (apiToken != null && !apiToken.isEmpty()) {
                    statusRequest.setHeader("Authorization", "Bearer " + apiToken);
                }
                
                HttpResponse response = httpClient.execute(statusRequest);
                String responseBody = EntityUtils.toString(response.getEntity());
                
                if (response.getStatusLine().getStatusCode() == 200) {
                    Map<String, Object> statusMap = objectMapper.readValue(responseBody, Map.class);
                    String status = (String) statusMap.get("status");
                    
                    listener.getLogger().println("📊 Scan status: " + status);
                    
                    if ("completed".equals(status)) {
                        // Get full results
                        return getFullResults(httpClient, jobId);
                    } else if ("failed".equals(status)) {
                        String errorMessage = (String) statusMap.get("error_message");
                        return ScanResult.failure("Scan failed: " + errorMessage);
                    }
                    
                    // Still running, wait and retry
                    Thread.sleep(10000); // Wait 10 seconds
                } else {
                    listener.getLogger().println("❌ Failed to get scan status: " + response.getStatusLine().getStatusCode());
                    Thread.sleep(10000);
                }
            }
            
            return ScanResult.failure("Scan timed out after " + timeoutMinutes + " minutes");
        }
    }
    
    private ScanResult getFullResults(CloseableHttpClient httpClient, String jobId) throws IOException {
        HttpPost resultsRequest = new HttpPost(apiUrl + "/api/v1/scans/" + jobId + "/results");
        if (apiToken != null && !apiToken.isEmpty()) {
            resultsRequest.setHeader("Authorization", "Bearer " + apiToken);
        }
        
        HttpResponse response = httpClient.execute(resultsRequest);
        String responseBody = EntityUtils.toString(response.getEntity());
        
        if (response.getStatusLine().getStatusCode() == 200) {
            // Parse results
            Map<String, Object> resultsMap = objectMapper.readValue(responseBody, Map.class);
            return ScanResult.success(resultsMap);
        } else {
            return ScanResult.failure("Failed to get scan results: " + response.getStatusLine().getStatusCode());
        }
    }
    
    private void saveResultsToWorkspace(FilePath workspace, ScanResult result, ScanOptions options) 
            throws IOException, InterruptedException {
        
        // Save JSON results
        String jsonResults = objectMapper.writeValueAsString(result.getResultsData());
        FilePath jsonFile = workspace.child("agentscan-results.json");
        jsonFile.write(jsonResults, "UTF-8");
        
        // Save SARIF results if requested
        if (options.getOutputFormat().contains("sarif")) {
            String sarifResults = convertToSarif(result.getResultsData());
            FilePath sarifFile = workspace.child("agentscan-results.sarif");
            sarifFile.write(sarifResults, "UTF-8");
        }
        
        listener.getLogger().println("💾 Scan results saved to workspace");
    }
    
    private String convertToSarif(Map<String, Object> resultsData) {
        // Convert AgentScan results to SARIF format
        // This is a simplified implementation
        Map<String, Object> sarif = new HashMap<>();
        sarif.put("version", "2.1.0");
        sarif.put("$schema", "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json");
        
        // Add runs array with tool information and results
        // Implementation would convert findings to SARIF format
        
        try {
            return objectMapper.writeValueAsString(sarif);
        } catch (Exception e) {
            return "{}"; // Return empty SARIF on error
        }
    }
}