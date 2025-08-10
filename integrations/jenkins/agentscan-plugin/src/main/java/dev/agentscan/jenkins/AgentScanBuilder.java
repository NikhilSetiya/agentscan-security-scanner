package dev.agentscan.jenkins;

import com.cloudbees.plugins.credentials.CredentialsProvider;
import com.cloudbees.plugins.credentials.common.StandardUsernamePasswordCredentials;
import com.cloudbees.plugins.credentials.domains.DomainRequirement;
import hudson.Extension;
import hudson.FilePath;
import hudson.Launcher;
import hudson.model.AbstractProject;
import hudson.model.Run;
import hudson.model.TaskListener;
import hudson.security.ACL;
import hudson.tasks.BuildStepDescriptor;
import hudson.tasks.Builder;
import hudson.util.FormValidation;
import hudson.util.ListBoxModel;
import jenkins.tasks.SimpleBuildStep;
import org.jenkinsci.Symbol;
import org.kohsuke.stapler.DataBoundConstructor;
import org.kohsuke.stapler.DataBoundSetter;
import org.kohsuke.stapler.QueryParameter;

import javax.annotation.Nonnull;
import java.io.IOException;
import java.util.Collections;
import java.util.List;

/**
 * Jenkins build step for AgentScan security scanning.
 */
public class AgentScanBuilder extends Builder implements SimpleBuildStep {

    private String apiUrl = "https://api.agentscan.dev";
    private String credentialsId;
    private String failOnSeverity = "high";
    private String excludePaths = "";
    private String includePaths = "";
    private String outputFormat = "json,sarif";
    private boolean uploadSarif = true;
    private boolean generateReport = true;
    private int timeoutMinutes = 30;

    @DataBoundConstructor
    public AgentScanBuilder() {
    }

    public String getApiUrl() {
        return apiUrl;
    }

    @DataBoundSetter
    public void setApiUrl(String apiUrl) {
        this.apiUrl = apiUrl;
    }

    public String getCredentialsId() {
        return credentialsId;
    }

    @DataBoundSetter
    public void setCredentialsId(String credentialsId) {
        this.credentialsId = credentialsId;
    }

    public String getFailOnSeverity() {
        return failOnSeverity;
    }

    @DataBoundSetter
    public void setFailOnSeverity(String failOnSeverity) {
        this.failOnSeverity = failOnSeverity;
    }

    public String getExcludePaths() {
        return excludePaths;
    }

    @DataBoundSetter
    public void setExcludePaths(String excludePaths) {
        this.excludePaths = excludePaths;
    }

    public String getIncludePaths() {
        return includePaths;
    }

    @DataBoundSetter
    public void setIncludePaths(String includePaths) {
        this.includePaths = includePaths;
    }

    public String getOutputFormat() {
        return outputFormat;
    }

    @DataBoundSetter
    public void setOutputFormat(String outputFormat) {
        this.outputFormat = outputFormat;
    }

    public boolean isUploadSarif() {
        return uploadSarif;
    }

    @DataBoundSetter
    public void setUploadSarif(boolean uploadSarif) {
        this.uploadSarif = uploadSarif;
    }

    public boolean isGenerateReport() {
        return generateReport;
    }

    @DataBoundSetter
    public void setGenerateReport(boolean generateReport) {
        this.generateReport = generateReport;
    }

    public int getTimeoutMinutes() {
        return timeoutMinutes;
    }

    @DataBoundSetter
    public void setTimeoutMinutes(int timeoutMinutes) {
        this.timeoutMinutes = timeoutMinutes;
    }

    @Override
    public void perform(@Nonnull Run<?, ?> run, @Nonnull FilePath workspace, 
                       @Nonnull Launcher launcher, @Nonnull TaskListener listener) 
                       throws InterruptedException, IOException {
        
        listener.getLogger().println("🔒 Starting AgentScan security analysis...");
        
        // Get API token from credentials
        String apiToken = null;
        if (credentialsId != null && !credentialsId.isEmpty()) {
            List<StandardUsernamePasswordCredentials> credentials = CredentialsProvider.lookupCredentials(
                StandardUsernamePasswordCredentials.class,
                run.getParent(),
                ACL.SYSTEM,
                Collections.<DomainRequirement>emptyList()
            );
            
            for (StandardUsernamePasswordCredentials cred : credentials) {
                if (credentialsId.equals(cred.getId())) {
                    apiToken = cred.getPassword().getPlainText();
                    break;
                }
            }
        }
        
        if (apiToken == null || apiToken.isEmpty()) {
            listener.getLogger().println("⚠️  No API token found. Proceeding without authentication.");
        }
        
        // Create AgentScan service
        AgentScanService service = new AgentScanService(apiUrl, apiToken, listener);
        
        // Configure scan options
        ScanOptions options = new ScanOptions();
        options.setFailOnSeverity(failOnSeverity);
        options.setExcludePaths(excludePaths);
        options.setIncludePaths(includePaths);
        options.setOutputFormat(outputFormat);
        options.setUploadSarif(uploadSarif);
        options.setGenerateReport(generateReport);
        options.setTimeoutMinutes(timeoutMinutes);
        
        // Execute scan
        ScanResult result = service.executeScan(workspace, options);
        
        // Process results
        if (result.isSuccess()) {
            listener.getLogger().println("✅ Security scan completed successfully");
            
            // Generate reports if requested
            if (generateReport) {
                generateJenkinsReport(workspace, result, listener);
            }
            
            // Archive artifacts
            archiveResults(workspace, run, listener);
            
            // Check if build should fail based on findings
            if (shouldFailBuild(result, failOnSeverity)) {
                listener.getLogger().println("❌ Build failed due to high severity security findings");
                run.setResult(hudson.model.Result.FAILURE);
            } else {
                listener.getLogger().println("✅ No critical security issues found");
            }
            
        } else {
            listener.getLogger().println("❌ Security scan failed: " + result.getErrorMessage());
            run.setResult(hudson.model.Result.FAILURE);
        }
    }
    
    private void generateJenkinsReport(FilePath workspace, ScanResult result, TaskListener listener) 
            throws IOException, InterruptedException {
        
        listener.getLogger().println("📊 Generating security report...");
        
        String htmlReport = HtmlReportGenerator.generateReport(result);
        
        FilePath reportFile = workspace.child("agentscan-security-report.html");
        reportFile.write(htmlReport, "UTF-8");
        
        listener.getLogger().println("📄 Security report generated: agentscan-security-report.html");
    }
    
    private void archiveResults(FilePath workspace, Run<?, ?> run, TaskListener listener) 
            throws IOException, InterruptedException {
        
        // Archive JSON and SARIF results
        FilePath jsonFile = workspace.child("agentscan-results.json");
        FilePath sarifFile = workspace.child("agentscan-results.sarif");
        FilePath reportFile = workspace.child("agentscan-security-report.html");
        
        if (jsonFile.exists()) {
            listener.getLogger().println("📁 Archiving scan results...");
            // Note: In a real implementation, you would use Jenkins' artifact archiving
            // run.addAction(new ArtifactArchiver("agentscan-results.*"));
        }
    }
    
    private boolean shouldFailBuild(ScanResult result, String failOnSeverity) {
        if (result.getSummary() == null) {
            return false;
        }
        
        ScanSummary summary = result.getSummary();
        
        switch (failOnSeverity.toLowerCase()) {
            case "high":
                return summary.getHighSeverityCount() > 0;
            case "medium":
                return summary.getHighSeverityCount() > 0 || summary.getMediumSeverityCount() > 0;
            case "low":
                return summary.getTotalFindings() > 0;
            default:
                return false;
        }
    }

    @Symbol("agentScan")
    @Extension
    public static final class DescriptorImpl extends BuildStepDescriptor<Builder> {

        public FormValidation doCheckApiUrl(@QueryParameter String value) {
            if (value == null || value.isEmpty()) {
                return FormValidation.error("API URL is required");
            }
            if (!value.startsWith("http://") && !value.startsWith("https://")) {
                return FormValidation.error("API URL must start with http:// or https://");
            }
            return FormValidation.ok();
        }

        public FormValidation doCheckTimeoutMinutes(@QueryParameter String value) {
            try {
                int timeout = Integer.parseInt(value);
                if (timeout <= 0) {
                    return FormValidation.error("Timeout must be a positive number");
                }
                if (timeout > 120) {
                    return FormValidation.warning("Timeout is very high. Consider reducing it.");
                }
                return FormValidation.ok();
            } catch (NumberFormatException e) {
                return FormValidation.error("Timeout must be a valid number");
            }
        }

        public ListBoxModel doFillFailOnSeverityItems() {
            ListBoxModel items = new ListBoxModel();
            items.add("High severity only", "high");
            items.add("Medium and high severity", "medium");
            items.add("All findings", "low");
            items.add("Never fail", "never");
            return items;
        }

        public ListBoxModel doFillCredentialsIdItems() {
            ListBoxModel items = new ListBoxModel();
            items.add("- Select credentials -", "");
            
            List<StandardUsernamePasswordCredentials> credentials = CredentialsProvider.lookupCredentials(
                StandardUsernamePasswordCredentials.class,
                jenkins.model.Jenkins.get(),
                ACL.SYSTEM,
                Collections.<DomainRequirement>emptyList()
            );
            
            for (StandardUsernamePasswordCredentials cred : credentials) {
                items.add(cred.getDescription(), cred.getId());
            }
            
            return items;
        }

        @Override
        public boolean isApplicable(Class<? extends AbstractProject> aClass) {
            return true;
        }

        @Override
        @Nonnull
        public String getDisplayName() {
            return "AgentScan Security Scanner";
        }
    }
}