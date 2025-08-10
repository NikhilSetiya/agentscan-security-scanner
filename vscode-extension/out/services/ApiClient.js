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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.ApiClient = void 0;
const axios_1 = __importDefault(require("axios"));
const vscode = __importStar(require("vscode"));
class ApiClient {
    constructor(config) {
        this.config = config;
        this.client = axios_1.default.create({
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
        this.client.interceptors.response.use((response) => response, (error) => {
            this.handleApiError(error);
            return Promise.reject(error);
        });
    }
    async scanFile(filePath, content) {
        try {
            const request = {
                scanType: 'file',
                priority: 1,
                files: [filePath],
                content: { [filePath]: content }
            };
            const response = await this.client.post('/api/v1/scans', request);
            return response.data;
        }
        catch (error) {
            throw new Error(`Failed to scan file: ${error}`);
        }
    }
    async scanWorkspace(workspacePath) {
        try {
            const request = {
                repositoryUrl: workspacePath,
                scanType: 'workspace',
                priority: 3 // Lower priority for workspace scans
            };
            const response = await this.client.post('/api/v1/scans', request);
            return response.data;
        }
        catch (error) {
            throw new Error(`Failed to scan workspace: ${error}`);
        }
    }
    async getScanStatus(scanId) {
        try {
            const response = await this.client.get(`/api/v1/scans/${scanId}`);
            return response.data;
        }
        catch (error) {
            throw new Error(`Failed to get scan status: ${error}`);
        }
    }
    async getScanResults(scanId) {
        try {
            const response = await this.client.get(`/api/v1/scans/${scanId}/results`);
            return response.data.findings;
        }
        catch (error) {
            throw new Error(`Failed to get scan results: ${error}`);
        }
    }
    async suppressFinding(findingId, request) {
        try {
            await this.client.post(`/api/v1/findings/${findingId}/suppress`, request);
        }
        catch (error) {
            throw new Error(`Failed to suppress finding: ${error}`);
        }
    }
    async updateFindingStatus(findingId, status, comment) {
        try {
            await this.client.patch(`/api/v1/findings/${findingId}/status`, {
                status,
                comment
            });
        }
        catch (error) {
            throw new Error(`Failed to update finding status: ${error}`);
        }
    }
    async getFindings(filter) {
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
            const response = await this.client.get(`/api/v1/findings?${params}`);
            return response.data.findings;
        }
        catch (error) {
            throw new Error(`Failed to get findings: ${error}`);
        }
    }
    async healthCheck() {
        try {
            await this.client.get('/health');
            return true;
        }
        catch (error) {
            return false;
        }
    }
    handleApiError(error) {
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
        }
        else if (error.request) {
            // Network error
            vscode.window.showErrorMessage('AgentScan: Unable to connect to server. Please check your connection and server URL.');
        }
        else {
            // Other error
            vscode.window.showErrorMessage(`AgentScan: Unexpected error - ${error.message}`);
        }
    }
    updateConfiguration(config) {
        this.config = config;
        this.client.defaults.baseURL = config.getServerUrl();
    }
}
exports.ApiClient = ApiClient;
//# sourceMappingURL=ApiClient.js.map