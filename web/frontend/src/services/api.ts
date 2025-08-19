/**
 * API Service Layer for AgentScan Frontend
 * Provides centralized API communication with proper error handling and authentication
 */

import { observeLogger } from './observeLogger'
import { enhancedApiCall } from '../utils/retryMechanism'

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

// Backend response wrapper format
interface BackendResponse<T> {
  success: boolean;
  data?: T;
  error?: ApiError;
  meta?: {
    pagination?: Pagination;
    timestamp: string;
  };
  request_id?: string;
  timestamp: string;
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

// HTTP Client Class
class ApiClient {
  private baseURL: string;
  private timeout: number;
  private authToken: string | null = null;

  constructor(baseURL: string, timeout: number = API_TIMEOUT) {
    this.baseURL = baseURL;
    this.timeout = timeout;
    this.loadAuthToken();
  }

  private loadAuthToken(): void {
    this.authToken = localStorage.getItem('auth_token');
    console.log('[API] Loaded auth token from localStorage:', this.authToken ? this.authToken.substring(0, 20) + '...' : 'null');
  }

  private saveAuthToken(token: string): void {
    console.log('[API] Saving auth token:', token.substring(0, 20) + '...');
    this.authToken = token;
    localStorage.setItem('auth_token', token);
    console.log('[API] Token saved to localStorage, current authToken set');
  }

  private clearAuthToken(): void {
    this.authToken = null;
    localStorage.removeItem('auth_token');
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
    const url = `${this.baseURL}${endpoint}`;
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), this.timeout);
    const startTime = Date.now();

    // Create trace for this API call
    const traceId = observeLogger.createTrace(`API ${options.method || 'GET'} ${endpoint}`);

    try {
      const response = await fetch(url, {
        ...options,
        headers: {
          ...this.getHeaders(),
          ...options.headers,
        },
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

        // Extract error from standardized backend response
        const apiError: ApiError = backendResponse.error || {
          code: `HTTP_${response.status}`,
          message: `Request failed with status ${response.status}`,
        };

        return {
          data: undefined,
          error: apiError,
          status: response.status,
          meta: backendResponse.meta,
          request_id: backendResponse.request_id,
        };
      }

      // Handle successful response
      if (backendResponse.success === false && backendResponse.error) {
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
            error: backendResponse.error.message
          },
          duration
        );

        observeLogger.endTrace(traceId, false, {
          status: response.status,
          error: backendResponse.error.message,
          request_id: backendResponse.request_id
        });

        return {
          data: undefined,
          error: backendResponse.error,
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

      // Return the data from the standardized backend response with meta information
      return {
        data: backendResponse.data as T,
        error: undefined,
        status: response.status,
        meta: backendResponse.meta,
        request_id: backendResponse.request_id,
      };
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
    options: RequestInit = {}
  ): Promise<ApiResponse<T>> {
    return enhancedApiCall(
      () => this.request<T>(endpoint, options),
      {
        maxAttempts: 3,
        baseDelay: 1000,
        retryCondition: (error) => {
          // Don't retry authentication errors or client errors
          if (error?.status >= 400 && error?.status < 500) return false
          // Retry network errors and server errors
          return true
        }
      }
    )
  }

  // Authentication Methods
  async login(credentials: LoginRequest): Promise<ApiResponse<LoginResponse>> {
    const response = await this.enhancedRequest<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify(credentials),
    });

    if (response.data?.token) {
      this.saveAuthToken(response.data.token);
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
    return this.request<User>('/user/me');
  }

  // Repository Methods
  async getRepositories(params: PaginationParams & { search?: string } = {}): Promise<ApiResponse<RepositoryListResponse>> {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.set('page', params.page.toString());
    if (params.limit) searchParams.set('limit', params.limit.toString());
    if (params.search) searchParams.set('search', params.search);

    const query = searchParams.toString();
    const endpoint = query ? `/repositories?${query}` : '/repositories';

    return this.request<RepositoryListResponse>(endpoint);
  }

  async createRepository(repository: CreateRepositoryRequest): Promise<ApiResponse<Repository>> {
    return this.request<Repository>('/repositories', {
      method: 'POST',
      body: JSON.stringify(repository),
    });
  }

  // Scan Methods
  async getScans(params: PaginationParams & { repository_id?: string; status?: string } = {}): Promise<ApiResponse<ScanListResponse>> {
    const searchParams = new URLSearchParams();
    if (params.page) searchParams.set('page', params.page.toString());
    if (params.limit) searchParams.set('limit', params.limit.toString());
    if (params.repository_id) searchParams.set('repository_id', params.repository_id);
    if (params.status) searchParams.set('status', params.status);

    const query = searchParams.toString();
    const endpoint = query ? `/scans?${query}` : '/scans';

    return this.request<ScanListResponse>(endpoint);
  }

  async submitScan(scanRequest: SubmitScanRequest): Promise<ApiResponse<Scan>> {
    return this.request<Scan>('/scans', {
      method: 'POST',
      body: JSON.stringify(scanRequest),
    });
  }

  async getScanResults(scanId: string): Promise<ApiResponse<ScanResults>> {
    return this.request<ScanResults>(`/scans/${scanId}/results`);
  }

  async getScan(scanId: string): Promise<ApiResponse<Scan>> {
    return this.request<Scan>(`/scans/${scanId}`);
  }

  // Dashboard Methods
  async getDashboardStats(): Promise<ApiResponse<DashboardStats>> {
    return this.enhancedRequest<DashboardStats>('/dashboard/stats');
  }

  // Health Check
  async healthCheck(): Promise<ApiResponse<{ status: string; timestamp: string }>> {
    return this.enhancedRequest<{ status: string; timestamp: string }>('/health');
  }

  // Utility Methods
  isAuthenticated(): boolean {
    return !!this.authToken;
  }

  getAuthToken(): string | null {
    return this.authToken;
  }
}

// Create and export singleton instance
export const apiClient = new ApiClient(API_BASE_URL);

// Export default instance
export default apiClient;