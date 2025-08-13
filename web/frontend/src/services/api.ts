/**
 * API Service Layer for AgentScan Frontend
 * Provides centralized API communication with proper error handling and authentication
 */

// API Configuration
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1';
const API_TIMEOUT = 30000; // 30 seconds

// Types
export interface ApiError {
  error: string;
  code: string;
  details?: Record<string, string>;
}

export interface ApiResponse<T> {
  data?: T;
  error?: ApiError;
  status: number;
}

export interface PaginationParams {
  page?: number;
  limit?: number;
}

export interface Pagination {
  page: number;
  limit: number;
  total: number;
  total_pages: number;
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
  username: string;
  email: string;
  role: 'admin' | 'developer' | 'viewer';
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
  pagination: Pagination;
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
  pagination: Pagination;
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
  }

  private saveAuthToken(token: string): void {
    this.authToken = token;
    localStorage.setItem('auth_token', token);
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

      const contentType = response.headers.get('content-type');
      let data: any = null;

      if (contentType && contentType.includes('application/json')) {
        data = await response.json();
      } else {
        data = await response.text();
      }

      if (!response.ok) {
        // Handle authentication errors
        if (response.status === 401) {
          this.clearAuthToken();
          window.dispatchEvent(new CustomEvent('auth:logout'));
        }

        return {
          data: undefined,
          error: data as ApiError,
          status: response.status,
        };
      }

      return {
        data: data as T,
        error: undefined,
        status: response.status,
      };
    } catch (error) {
      clearTimeout(timeoutId);

      if (error instanceof Error) {
        if (error.name === 'AbortError') {
          return {
            data: undefined,
            error: { error: 'Request timeout', code: 'TIMEOUT' },
            status: 408,
          };
        }

        return {
          data: undefined,
          error: { error: error.message, code: 'NETWORK_ERROR' },
          status: 0,
        };
      }

      return {
        data: undefined,
        error: { error: 'Unknown error occurred', code: 'UNKNOWN_ERROR' },
        status: 0,
      };
    }
  }

  // Authentication Methods
  async login(credentials: LoginRequest): Promise<ApiResponse<LoginResponse>> {
    const response = await this.request<LoginResponse>('/auth/login', {
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
    return this.request<DashboardStats>('/dashboard/stats');
  }

  // Health Check
  async healthCheck(): Promise<ApiResponse<{ status: string; timestamp: string }>> {
    return this.request<{ status: string; timestamp: string }>('/health');
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