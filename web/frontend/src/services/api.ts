/**
 * API Service Layer for AgentScan Frontend
 * Provides centralized API communication with proper error handling and authentication
 */

import { observeLogger } from './observeLogger'
import { enhancedApiCall } from '../utils/retryMechanism'
import { supabase } from '../lib/supabase'

// API Configuration
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';
const API_TIMEOUT = 30000; // 30 seconds

// Types - Updated to match backend standardized response format
export interface ApiError {
  code: string;
  message: string;
  details?: Record<string, any>;
}

export interface ApiResponse<T> {
  data?: T;
  error?: ApiError;
  status: number;
  meta?: {
    pagination?: Pagination;
    timestamp?: string;
  };
  request_id?: string;
}

export interface PaginationParams {
  page?: number;
  limit?: number;
}

export interface Pagination {
  page: number;
  page_size: number;
  total: number;
  total_pages: number;
  has_next: boolean;
  has_prev: boolean;
}

// Backend response wrapper format - Updated to match new clean architecture
interface BackendResponse<T> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
  details?: Record<string, any>;
  meta?: {
    pagination?: Pagination;
    timestamp?: string;
  };
  request_id?: string;
}

// Authentication Types
export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: User;
  expires_at: string;
}

export interface User {
  id: string;
  name: string;
  email: string;
  avatar_url: string;
  github_id?: number;
  gitlab_id?: number;
  created_at: string;
  updated_at: string;
  // Legacy frontend compatibility
  username?: string;
  role?: 'admin' | 'developer' | 'viewer';
}

// Repository Types
export interface Repository {
  id: string;
  name: string;
  url: string;
  language: string;
  branch: string;
  created_at: string;
  last_scan_at?: string;
}

export interface CreateRepositoryRequest {
  name: string;
  url: string;
  language: string;
  branch?: string;
}

export interface RepositoryListResponse {
  repositories: Repository[];
}

// Scan Types
export interface Scan {
  id: string;
  repository_id: string;
  repository?: Repository;
  status: 'queued' | 'running' | 'completed' | 'failed' | 'cancelled';
  progress: number;
  findings_count: number;
  started_at: string;
  completed_at?: string;
  duration?: string;
  branch: string;
  commit: string;
  commit_message?: string;
  triggered_by?: string;
  scan_type: 'full' | 'incremental';
}

export interface SubmitScanRequest {
  repository_id: string;
  scan_type?: 'full' | 'incremental';
  agents?: string[];
  branch?: string;
  commit?: string;
}

export interface ScanListResponse {
  scans: Scan[];
}

export interface Finding {
  id: string;
  rule_id: string;
  title: string;
  description: string;
  severity: 'critical' | 'high' | 'medium' | 'low' | 'info';
  file_path: string;
  line_number: number;
  column_number?: number;
  tool: string;
  tools?: string[];
  confidence: number;
  status: 'open' | 'ignored' | 'fixed' | 'false_positive';
  code_snippet?: string;
  fix_suggestion?: string;
  references?: string[];
}

export interface ScanResults {
  scan: Scan;
  findings: Finding[];
  statistics: {
    total: number;
    by_severity: Record<string, number>;
    by_status: Record<string, number>;
    by_tool: Record<string, number>;
  };
}

export interface DashboardStats {
  total_scans: number;
  total_repositories: number;
  findings_by_severity: {
    critical: number;
    high: number;
    medium: number;
    low: number;
    info: number;
  };
  recent_scans: Scan[];
  trend_data: Array<{
    date: string;
    critical: number;
    high: number;
    medium: number;
    low: number;
    info: number;
  }>;
}

// Request/Response Interceptor Types
export interface RequestInterceptor {
  (config: RequestConfig): RequestConfig | Promise<RequestConfig>;
}

export interface ResponseInterceptor {
  (response: ApiResponse<any>): ApiResponse<any> | Promise<ApiResponse<any>>;
}

export interface RequestConfig {
  url: string;
  method: string;
  headers: Record<string, string>;
  body?: any;
  timeout?: number;
}

// HTTP Client Class
class ApiClient {
  private baseURL: string;
  private timeout: number;
  private authToken: string | null = null;
  private requestInterceptors: RequestInterceptor[] = [];
  private responseInterceptors: ResponseInterceptor[] = [];

  constructor(baseURL: string, timeout: number = API_TIMEOUT) {
    this.baseURL = baseURL;
    this.timeout = timeout;
    // Initialize auth token asynchronously
    this.loadAuthToken().catch(console.error);
    
    // Add default interceptors
    this.addDefaultInterceptors();
  }

  private addDefaultInterceptors(): void {
    // Request interceptor for logging
    this.addRequestInterceptor((config) => {
      console.log(`[API] ${config.method} ${config.url}`, {
        headers: config.headers,
        body: config.body
      });
      return config;
    });

    // Response interceptor for logging
    this.addResponseInterceptor((response) => {
      if (response.error) {
        console.error(`[API] Response error:`, response.error);
      } else {
        console.log(`[API] Response success:`, {
          status: response.status,
          data: response.data
        });
      }
      return response;
    });
  }

  // Interceptor management
  addRequestInterceptor(interceptor: RequestInterceptor): void {
    this.requestInterceptors.push(interceptor);
  }

  addResponseInterceptor(interceptor: ResponseInterceptor): void {
    this.responseInterceptors.push(interceptor);
  }

  private async applyRequestInterceptors(config: RequestConfig): Promise<RequestConfig> {
    let processedConfig = config;
    for (const interceptor of this.requestInterceptors) {
      processedConfig = await interceptor(processedConfig);
    }
    return processedConfig;
  }

  private async applyResponseInterceptors<T>(response: ApiResponse<T>): Promise<ApiResponse<T>> {
    let processedResponse = response;
    for (const interceptor of this.responseInterceptors) {
      processedResponse = await interceptor(processedResponse);
    }
    return processedResponse;
  }

  private async loadAuthToken(): Promise<void> {
    // Get Supabase session instead of localStorage token
    const { data: { session } } = await supabase.auth.getSession();
    this.authToken = session?.access_token || null;
    console.log('[API] Loaded auth token from Supabase:', this.authToken ? this.authToken.substring(0, 20) + '...' : 'null');
  }

  private saveAuthToken(token: string): void {
    console.log('[API] Saving auth token:', token.substring(0, 20) + '...');
    this.authToken = token;
    localStorage.setItem('auth_token', token);
    console.log('[API] Token saved to localStorage, current authToken set');
  }

  private clearAuthToken(): void {
    this.authToken = null;
    // Clear localStorage for backward compatibility, but don't rely on it
    localStorage.removeItem('auth_token');
  }

  // Add method to refresh token from Supabase
  async refreshAuthToken(): Promise<void> {
    await this.loadAuthToken();
  }

  private getHeaders(): Record<string, string> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    };

    if (this.authToken) {
      headers['Authorization'] = `Bearer ${this.authToken}`;
      console.log('[API] Adding Authorization header with token:', this.authToken.substring(0, 20) + '...');
    } else {
      console.log('[API] No auth token available for request');
    }

    return headers;
  }

  private async request<T>(
    endpoint: string,
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    // Ensure we have a fresh auth token before making the request
    await this.loadAuthToken();
    
    const url = `${this.baseURL}${endpoint}`;
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);
    const startTime = Date.now();

    // Create trace for this API call
    const traceId = observeLogger.createTrace(`API ${options.method || 'GET'} ${endpoint}`);

    // Prepare request config
    let requestConfig: RequestConfig = {
      url: endpoint,
      method: options.method || 'GET',
      headers: {
        ...this.getHeaders(),
        ...options.headers,
      },
      body: options.body,
      timeout: this.timeout
    };

    // Apply request interceptors
    try {
      requestConfig = await this.applyRequestInterceptors(requestConfig);
    } catch (error) {
      console.error('[API] Request interceptor error:', error);
    }

    try {
      const response = await fetch(url, {
        ...options,
        method: requestConfig.method,
        headers: requestConfig.headers,
        body: requestConfig.body,
        signal: controller.signal,
      });

      clearTimeout(timeoutId);
      const duration = Date.now() - startTime;

      const contentType = response.headers.get('content-type');
      let data: any = null;

      if (contentType && contentType.includes('application/json')) {
        data = await response.json();
      } else {
        data = await response.text();
      }

      // Parse backend response format
      const backendResponse = data as BackendResponse<T>;

      if (!response.ok) {
        // Log API call failure
        observeLogger.logApiCall(
          {
            method: options.method || 'GET',
            url: endpoint,
            headers: this.getHeaders(),
            body: options.body
          },
          {
            status: response.status,
            body: data,
            error: `HTTP ${response.status}`
          },
          duration
        );

        // End trace with failure
        observeLogger.endTrace(traceId, false, {
          status: response.status,
          error: `HTTP ${response.status}`,
          request_id: backendResponse.request_id
        });

        // Handle authentication errors
        if (response.status === 401) {
          console.log('[API] 401 Unauthorized received, clearing token and dispatching logout event');
          this.clearAuthToken();
          window.dispatchEvent(new CustomEvent('auth:logout'));
        } else {
          console.log(`[API] Request failed with status ${response.status}:`, url);
        }

        // Extract error from new backend response format
        const apiError: ApiError = {
          code: `HTTP_${response.status}`,
          message: backendResponse.error || backendResponse.message || `Request failed with status ${response.status}`,
          details: backendResponse.details
        };

        return {
          data: undefined,
          error: apiError,
          status: response.status,
          meta: backendResponse.meta,
          request_id: backendResponse.request_id,
        };
      }

      // Handle successful response with success: false
      if (backendResponse.success === false) {
        // Backend returned success: false with error details
        observeLogger.logApiCall(
          {
            method: options.method || 'GET',
            url: endpoint,
            headers: this.getHeaders(),
            body: options.body
          },
          {
            status: response.status,
            body: data,
            error: backendResponse.error || backendResponse.message
          },
          duration
        );

        observeLogger.endTrace(traceId, false, {
          status: response.status,
          error: backendResponse.error || backendResponse.message,
          request_id: backendResponse.request_id
        });

        const apiError: ApiError = {
          code: 'BACKEND_ERROR',
          message: backendResponse.error || backendResponse.message || 'Backend returned error',
          details: backendResponse.details
        };

        return {
          data: undefined,
          error: apiError,
          status: response.status,
          meta: backendResponse.meta,
          request_id: backendResponse.request_id,
        };
      }

      // Log successful API call
      observeLogger.logApiCall(
        {
          method: options.method || 'GET',
          url: endpoint,
          headers: this.getHeaders(),
          body: options.body
        },
        {
          status: response.status,
          body: data
        },
        duration
      );

      // End trace with success
      observeLogger.endTrace(traceId, true, {
        status: response.status,
        duration_ms: duration,
        request_id: backendResponse.request_id
      });

      // Create successful response
      let apiResponse: ApiResponse<T> = {
        data: backendResponse.data as T,
        error: undefined,
        status: response.status,
        meta: backendResponse.meta,
        request_id: backendResponse.request_id,
      };

      // Apply response interceptors
      try {
        apiResponse = await this.applyResponseInterceptors(apiResponse);
      } catch (error) {
        console.error('[API] Response interceptor error:', error);
      }

      return apiResponse;
    } catch (error) {
      clearTimeout(timeoutId);
      const duration = Date.now() - startTime;

      // Log error to Observe
      if (error instanceof Error) {
        observeLogger.logError(error, {
          endpoint,
          method: options.method || 'GET',
          duration_ms: duration
        });
      }

      // End trace with failure
      observeLogger.endTrace(traceId, false, {
        error: error instanceof Error ? error.message : 'Unknown error',
        duration_ms: duration
      });

      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          return {
            data: undefined,
            error: { code: 'TIMEOUT', message: 'Request timeout' },
            status: 408,
          };
        }

        return {
          data: undefined,
          error: { code: 'NETWORK_ERROR', message: error.message },
          status: 0,
        };
      }

      return {
        data: undefined,
        error: { code: 'UNKNOWN_ERROR', message: 'Unknown error occurred' },
        status: 0,
      };
    }
  }

  // Enhanced request method with retry mechanism
  private async enhancedRequest<T>(
    endpoint: string,
    options: RequestInit = {},
    retryConfig?: {
      maxAttempts?: number;
      baseDelay?: number;
      maxDelay?: number;
      retryCondition?: (error: ApiResponse<any>) => boolean;
    }
  ): Promise<ApiResponse<T>> {
    const defaultRetryConfig = {
      maxAttempts: 3,
      baseDelay: 1000,
      maxDelay: 10000,
      retryCondition: (error: ApiResponse<any>) => {
        // Don't retry authentication errors or client errors (4xx)
        if (error?.status >= 400 && error?.status < 500) return false;
        // Don't retry if it's a validation error
        if (error?.error?.code === 'VALIDATION_ERROR') return false;
        // Retry network errors, timeouts, and server errors (5xx)
        return error?.status === 0 || error?.status >= 500 || error?.error?.code === 'TIMEOUT';
      }
    };

    const config = { ...defaultRetryConfig, ...retryConfig };

    return enhancedApiCall(
      () => this.request<T>(endpoint, options),
      config
    );
  }

  // Authentication Methods
  async login(credentials: LoginRequest): Promise<ApiResponse<LoginResponse>> {
    const response = await this.enhancedRequest<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    }, {
      maxAttempts: 2, // Don't retry login too many times
      retryCondition: (error) => error?.status >= 500 // Only retry server errors
    });

    if (response.data?.token) {
      this.saveAuthToken(response.data.token);
      this.setConnectionState('connected');
    }

    return response;
  }

  async logout(): Promise<ApiResponse<{ message: string }>> {
    const response = await this.request<{ message: string }>('/auth/logout', {
      method: 'POST',
    });

    this.clearAuthToken();
    return response;
  }

  async getCurrentUser(): Promise<ApiResponse<User>> {
    return this.enhancedRequest<User>('/user/me');
  }

  // Repository Methods
  async getRepositories(params: PaginationParams & { search?: string; organization_id?: string } = {}): Promise<ApiResponse<RepositoryListResponse>> {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.set('page', params.page.toString());
    if (params.limit) searchParams.set('page_size', params.limit.toString()); // Updated to match backend
    if (params.search) searchParams.set('search', params.search);
    if (params.organization_id) searchParams.set('organization_id', params.organization_id);

    const query = searchParams.toString();
    const endpoint = query ? `/repositories?${query}` : '/repositories';

    return this.enhancedRequest<RepositoryListResponse>(endpoint);
  }

  async getRepository(id: string): Promise<ApiResponse<Repository>> {
    return this.enhancedRequest<Repository>(`/repositories/${id}`);
  }

  async createRepository(repository: CreateRepositoryRequest): Promise<ApiResponse<Repository>> {
    return this.enhancedRequest<Repository>('/repositories', {
      method: 'POST',
      body: JSON.stringify(repository),
    }, {
      maxAttempts: 2, // Don't retry creation too many times
      retryCondition: (error) => error?.status >= 500
    });
  }

  async updateRepository(id: string, updates: Partial<CreateRepositoryRequest>): Promise<ApiResponse<Repository>> {
    return this.enhancedRequest<Repository>(`/repositories/${id}`, {
      method: 'PUT',
      body: JSON.stringify(updates),
    });
  }

  async deleteRepository(id: string): Promise<ApiResponse<{ message: string }>> {
    return this.enhancedRequest<{ message: string }>(`/repositories/${id}`, {
      method: 'DELETE',
    });
  }

  // Scan Job Methods (Updated to match new backend endpoints)
  async getScanJobs(params: PaginationParams & { repository_id?: string; status?: string; user_id?: string } = {}): Promise<ApiResponse<ScanListResponse>> {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.set('page', params.page.toString());
    if (params.limit) searchParams.set('page_size', params.limit.toString());
    if (params.status) searchParams.set('status', params.status);

    const query = searchParams.toString();
    let endpoint = '/scan-jobs';
    
    // Use specific endpoints for repository or user scans
    if (params.repository_id) {
      endpoint = `/repositories/${params.repository_id}/scan-jobs`;
    } else if (params.user_id) {
      endpoint = `/users/${params.user_id}/scan-jobs`;
    }
    
    if (query) {
      endpoint += `?${query}`;
    }

    return this.enhancedRequest<ScanListResponse>(endpoint);
  }

  async createScanJob(scanRequest: SubmitScanRequest): Promise<ApiResponse<Scan>> {
    return this.enhancedRequest<Scan>('/scan-jobs', {
      method: 'POST',
      body: JSON.stringify(scanRequest),
    }, {
      maxAttempts: 2,
      retryCondition: (error) => error?.status >= 500
    });
  }

  async getScanJob(scanId: string): Promise<ApiResponse<Scan>> {
    return this.enhancedRequest<Scan>(`/scan-jobs/${scanId}`);
  }

  async getScanJobWithDetails(scanId: string): Promise<ApiResponse<ScanResults>> {
    return this.enhancedRequest<ScanResults>(`/scan-jobs/${scanId}/details`);
  }

  async startScanJob(scanId: string): Promise<ApiResponse<{ message: string }>> {
    return this.enhancedRequest<{ message: string }>(`/scan-jobs/${scanId}/start`, {
      method: 'POST',
    });
  }

  async cancelScanJob(scanId: string): Promise<ApiResponse<{ message: string }>> {
    return this.enhancedRequest<{ message: string }>(`/scan-jobs/${scanId}/cancel`, {
      method: 'POST',
    });
  }

  async retryScanJob(scanId: string): Promise<ApiResponse<Scan>> {
    return this.enhancedRequest<Scan>(`/scan-jobs/${scanId}/retry`, {
      method: 'POST',
    });
  }

  async getQueuedJobs(limit: number = 50): Promise<ApiResponse<ScanListResponse>> {
    return this.enhancedRequest<ScanListResponse>(`/scan-jobs/queued?limit=${limit}`);
  }

  async getRunningJobs(): Promise<ApiResponse<ScanListResponse>> {
    return this.enhancedRequest<ScanListResponse>('/scan-jobs/running');
  }

  // Legacy methods for backward compatibility
  async getScans(params: PaginationParams & { repository_id?: string; status?: string } = {}): Promise<ApiResponse<ScanListResponse>> {
    return this.getScanJobs(params);
  }

  async submitScan(scanRequest: SubmitScanRequest): Promise<ApiResponse<Scan>> {
    return this.createScanJob(scanRequest);
  }

  async getScanResults(scanId: string): Promise<ApiResponse<ScanResults>> {
    return this.getScanJobWithDetails(scanId);
  }

  async getScan(scanId: string): Promise<ApiResponse<Scan>> {
    return this.getScanJob(scanId);
  }

  // Dashboard Methods
  async getDashboardStats(): Promise<ApiResponse<DashboardStats>> {
    return this.enhancedRequest<DashboardStats>('/dashboard/stats');
  }

  // Health Check
  async healthCheck(): Promise<ApiResponse<{ status: string; timestamp: string }>> {
    return this.enhancedRequest<{ status: string; timestamp: string }>('/health');
  }

  // Connection state management
  private connectionState: 'connected' | 'disconnected' | 'reconnecting' = 'connected';
  private connectionListeners: Array<(state: string) => void> = [];

  onConnectionStateChange(listener: (state: string) => void): () => void {
    this.connectionListeners.push(listener);
    return () => {
      const index = this.connectionListeners.indexOf(listener);
      if (index > -1) {
        this.connectionListeners.splice(index, 1);
      }
    };
  }

  private setConnectionState(state: 'connected' | 'disconnected' | 'reconnecting'): void {
    if (this.connectionState !== state) {
      this.connectionState = state;
      this.connectionListeners.forEach(listener => listener(state));
    }
  }

  getConnectionState(): string {
    return this.connectionState;
  }

  // Network connectivity check
  async checkConnectivity(): Promise<boolean> {
    try {
      const response = await this.request<{ status: string }>('/health');
      const isConnected = !response.error;
      this.setConnectionState(isConnected ? 'connected' : 'disconnected');
      return isConnected;
    } catch {
      this.setConnectionState('disconnected');
      return false;
    }
  }

  // Utility Methods
  isAuthenticated(): boolean {
    return !!this.authToken;
  }

  getAuthToken(): string | null {
    return this.authToken;
  }

  setAuthToken(token: string | null): void {
    this.authToken = token;
    if (token) {
      this.saveAuthToken(token);
    } else {
      this.clearAuthToken();
    }
  }

  // Request timeout configuration
  setTimeout(timeout: number): void {
    this.timeout = timeout;
  }

  getTimeout(): number {
    return this.timeout;
  }
}

// Create and export singleton instance
export const apiClient = new ApiClient(API_BASE_URL);

// Add global error handling interceptor
apiClient.addResponseInterceptor((response) => {
  // Handle global errors
  if (response.error) {
    switch (response.error.code) {
      case 'NETWORK_ERROR':
        // Show network error notification
        console.error('[API] Network error detected:', response.error.message);
        break;
      case 'TIMEOUT':
        // Show timeout notification
        console.error('[API] Request timeout:', response.error.message);
        break;
      case 'HTTP_401':
        // Handle authentication error
        console.error('[API] Authentication error, redirecting to login');
        window.dispatchEvent(new CustomEvent('auth:logout'));
        break;
      case 'HTTP_403':
        // Handle authorization error
        console.error('[API] Authorization error:', response.error.message);
        break;
      case 'HTTP_429':
        // Handle rate limiting
        console.warn('[API] Rate limit exceeded:', response.error.message);
        break;
      case 'HTTP_500':
      case 'HTTP_502':
      case 'HTTP_503':
      case 'HTTP_504':
        // Handle server errors
        console.error('[API] Server error:', response.error.message);
        break;
    }
  }
  return response;
});

// Add request monitoring interceptor
apiClient.addRequestInterceptor((config) => {
  // Add request timestamp for monitoring
  config.headers['X-Request-Timestamp'] = Date.now().toString();
  
  // Add client version for debugging
  config.headers['X-Client-Version'] = '1.0.0';
  
  return config;
});

// Periodic connectivity check
let connectivityCheckInterval: NodeJS.Timeout | null = null;

export function startConnectivityMonitoring(intervalMs: number = 30000): void {
  if (connectivityCheckInterval) {
    clearInterval(connectivityCheckInterval);
  }
  
  connectivityCheckInterval = setInterval(async () => {
    await apiClient.checkConnectivity();
  }, intervalMs);
}

export function stopConnectivityMonitoring(): void {
  if (connectivityCheckInterval) {
    clearInterval(connectivityCheckInterval);
    connectivityCheckInterval = null;
  }
}

// Export utility functions
export function isApiError(error: any): error is ApiError {
  return error && typeof error.code === 'string' && typeof error.message === 'string';
}

export function getErrorMessage(error: ApiError | Error | string): string {
  if (typeof error === 'string') return error;
  if (error instanceof Error) return error.message;
  if (isApiError(error)) return error.message;
  return 'An unknown error occurred';
}

// Export default instance
export default apiClient;