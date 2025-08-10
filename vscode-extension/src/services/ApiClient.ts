import axios, { AxiosInstance, AxiosResponse } from 'axios';
import * as vscode from 'vscode';
import { ConfigurationManager } from './ConfigurationManager';

export interface ScanRequest {
    repositoryUrl?: string;
    branch?: string;
    commitSha?: string;
    scanType: 'file' | 'workspace';
    priority?: number;
    agentsRequested?: string[];
    files?: string[];
    content?: { [filePath: string]: string };
}

export interface ScanResult {
    id: string;
    status: 'queued' | 'running' | 'completed' | 'failed' | 'cancelled';
    findings: Finding[];
    startedAt?: string;
    completedAt?: string;
    errorMessage?: string;
}

export interface Finding {
    id: string;
    tool: string;
    ruleId: string;
    severity: 'high' | 'medium' | 'low';
    category: string;
    title: string;
    description: string;
    filePath: string;
    lineNumber: number;
    columnNumber?: number;
    codeSnippet?: string;
    confidence: number;
    consensusScore?: number;
    status: 'open' | 'fixed' | 'ignored' | 'false_positive';
    fixSuggestion?: string;
    references?: string[];
}

export interface SuppressionRequest {
    reason: string;
    expiresAt?: string;
}

export class ApiClient {
    private client: AxiosInstance;
    private config: ConfigurationManager;

    constructor(config: ConfigurationManager) {
        this.config = config;
        this.client = axios.create({
            baseURL: config.getServerUrl(),
            timeout: 30000,
            headers: {
                'Content-Type': 'application/json',
                'User-Agent': 'AgentScan-VSCode-Extension/0.1.0'
            }
        });

        // Add request interceptor for authentication
        this.client.interceptors.request.use((config) => {
            const apiKey = this.config.getApiKey();
            if (apiKey) {
                config.headers.Authorization = `Bearer ${apiKey}`;
            }
            return config;
        });

        // Add response interceptor for error handling
        this.client.interceptors.response.use(
            (response) => response,
            (error) => {
                this.handleApiError(error);
                return Promise.reject(error);
            }
        );
    }

    async scanFile(filePath: string, content: string): Promise<ScanResult> {
        try {
            const request: ScanRequest = {
                scanType: 'file',
                priority: 1, // High priority for IDE scans
                files: [filePath],
                content: { [filePath]: content }
            };

            const response: AxiosResponse<ScanResult> = await this.client.post('/api/v1/scans', request);
            return response.data;
        } catch (error) {
            throw new Error(`Failed to scan file: ${error}`);
        }
    }

    async scanWorkspace(workspacePath: string): Promise<ScanResult> {
        try {
            const request: ScanRequest = {
                repositoryUrl: workspacePath,
                scanType: 'workspace',
                priority: 3 // Lower priority for workspace scans
            };

            const response: AxiosResponse<ScanResult> = await this.client.post('/api/v1/scans', request);
            return response.data;
        } catch (error) {
            throw new Error(`Failed to scan workspace: ${error}`);
        }
    }

    async getScanStatus(scanId: string): Promise<ScanResult> {
        try {
            const response: AxiosResponse<ScanResult> = await this.client.get(`/api/v1/scans/${scanId}`);
            return response.data;
        } catch (error) {
            throw new Error(`Failed to get scan status: ${error}`);
        }
    }

    async getScanResults(scanId: string): Promise<Finding[]> {
        try {
            const response: AxiosResponse<{ findings: Finding[] }> = await this.client.get(`/api/v1/scans/${scanId}/results`);
            return response.data.findings;
        } catch (error) {
            throw new Error(`Failed to get scan results: ${error}`);
        }
    }

    async suppressFinding(findingId: string, request: SuppressionRequest): Promise<void> {
        try {
            await this.client.post(`/api/v1/findings/${findingId}/suppress`, request);
        } catch (error) {
            throw new Error(`Failed to suppress finding: ${error}`);
        }
    }

    async updateFindingStatus(findingId: string, status: string, comment?: string): Promise<void> {
        try {
            await this.client.patch(`/api/v1/findings/${findingId}/status`, {
                status,
                comment
            });
        } catch (error) {
            throw new Error(`Failed to update finding status: ${error}`);
        }
    }

    async getFindings(filter?: {
        severity?: string[];
        status?: string[];
        tool?: string[];
        filePath?: string;
        search?: string;
    }): Promise<Finding[]> {
        try {
            const params = new URLSearchParams();
            if (filter) {
                if (filter.severity) {
                    filter.severity.forEach(s => params.append('severity', s));
                }
                if (filter.status) {
                    filter.status.forEach(s => params.append('status', s));
                }
                if (filter.tool) {
                    filter.tool.forEach(t => params.append('tool', t));
                }
                if (filter.filePath) {
                    params.append('file_path', filter.filePath);
                }
                if (filter.search) {
                    params.append('search', filter.search);
                }
            }

            const response: AxiosResponse<{ findings: Finding[] }> = await this.client.get(`/api/v1/findings?${params}`);
            return response.data.findings;
        } catch (error) {
            throw new Error(`Failed to get findings: ${error}`);
        }
    }

    async healthCheck(): Promise<boolean> {
        try {
            await this.client.get('/health');
            return true;
        } catch (error) {
            return false;
        }
    }

    private handleApiError(error: any) {
        if (error.response) {
            // Server responded with error status
            const status = error.response.status;
            const message = error.response.data?.error || error.response.statusText;
            
            switch (status) {
                case 401:
                    vscode.window.showErrorMessage('AgentScan: Authentication failed. Please check your API key.');
                    break;
                case 403:
                    vscode.window.showErrorMessage('AgentScan: Access denied. Please check your permissions.');
                    break;
                case 404:
                    vscode.window.showErrorMessage('AgentScan: Resource not found. Please check the server URL.');
                    break;
                case 429:
                    vscode.window.showWarningMessage('AgentScan: Rate limit exceeded. Please try again later.');
                    break;
                case 500:
                    vscode.window.showErrorMessage(`AgentScan: Server error - ${message}`);
                    break;
                default:
                    vscode.window.showErrorMessage(`AgentScan: API error (${status}) - ${message}`);
            }
        } else if (error.request) {
            // Network error
            vscode.window.showErrorMessage('AgentScan: Unable to connect to server. Please check your connection and server URL.');
        } else {
            // Other error
            vscode.window.showErrorMessage(`AgentScan: Unexpected error - ${error.message}`);
        }
    }

    updateConfiguration(config: ConfigurationManager) {
        this.config = config;
        this.client.defaults.baseURL = config.getServerUrl();
    }
}