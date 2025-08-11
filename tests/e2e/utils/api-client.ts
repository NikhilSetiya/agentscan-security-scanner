import axios, { AxiosInstance, AxiosResponse } from 'axios';
import jwt from 'jsonwebtoken';

export interface User {
  email: string;
  username: string;
  role: string;
  password: string;
}

export interface Repository {
  name: string;
  url: string;
  language: string;
  branch?: string;
}

export interface ScanJob {
  id: string;
  repository_id: string;
  status: string;
  created_at: string;
  completed_at?: string;
  findings_count: number;
}

export interface Finding {
  id: string;
  rule_id: string;
  severity: string;
  title: string;
  description: string;
  file_path: string;
  line_number: number;
  tool: string;
}

export class ApiClient {
  private client: AxiosInstance;
  private authToken?: string;

  constructor(baseURL: string) {
    this.client = axios.create({
      baseURL: `${baseURL}/api/v1`,
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    });

    // Add request interceptor to include auth token
    this.client.interceptors.request.use((config) => {
      if (this.authToken) {
        config.headers.Authorization = `Bearer ${this.authToken}`;
      }
      return config;
    });

    // Add response interceptor for error handling
    this.client.interceptors.response.use(
      (response) => response,
      (error) => {
        console.error('API Error:', {
          url: error.config?.url,
          method: error.config?.method,
          status: error.response?.status,
          data: error.response?.data,
        });
        return Promise.reject(error);
      }
    );
  }

  async healthCheck(): Promise<void> {
    await this.client.get('/health');
  }

  async login(username: string, password: string): Promise<string> {
    const response = await this.client.post('/auth/login', {
      username,
      password,
    });
    
    this.authToken = response.data.token;
    return this.authToken;
  }

  async logout(): Promise<void> {
    if (this.authToken) {
      await this.client.post('/auth/logout');
      this.authToken = undefined;
    }
  }

  async createUser(user: User): Promise<any> {
    const response = await this.client.post('/users', user);
    return response.data;
  }

  async deleteUser(username: string): Promise<void> {
    await this.client.delete(`/users/${username}`);
  }

  async getUser(username: string): Promise<any> {
    const response = await this.client.get(`/users/${username}`);
    return response.data;
  }

  async createRepository(repository: Repository): Promise<any> {
    const response = await this.client.post('/repositories', repository);
    return response.data;
  }

  async deleteRepository(repositoryId: string): Promise<void> {
    await this.client.delete(`/repositories/${repositoryId}`);
  }

  async getRepositories(): Promise<any[]> {
    const response = await this.client.get('/repositories');
    return response.data;
  }

  async getRepository(repositoryId: string): Promise<any> {
    const response = await this.client.get(`/repositories/${repositoryId}`);
    return response.data;
  }

  async submitScan(repositoryId: string, options: any = {}): Promise<ScanJob> {
    const response = await this.client.post('/scans', {
      repository_id: repositoryId,
      ...options,
    });
    return response.data;
  }

  async getScanStatus(scanId: string): Promise<ScanJob> {
    const response = await this.client.get(`/scans/${scanId}`);
    return response.data;
  }

  async getScanResults(scanId: string): Promise<Finding[]> {
    const response = await this.client.get(`/scans/${scanId}/results`);
    return response.data;
  }

  async getScans(repositoryId?: string): Promise<ScanJob[]> {
    const url = repositoryId ? `/scans?repository_id=${repositoryId}` : '/scans';
    const response = await this.client.get(url);
    return response.data;
  }

  async suppressFinding(findingId: string, reason: string): Promise<void> {
    await this.client.post(`/findings/${findingId}/suppress`, { reason });
  }

  async updateFindingStatus(findingId: string, status: string): Promise<void> {
    await this.client.patch(`/findings/${findingId}`, { status });
  }

  async exportScanResults(scanId: string, format: 'json' | 'pdf'): Promise<Blob> {
    const response = await this.client.get(`/scans/${scanId}/export`, {
      params: { format },
      responseType: 'blob',
    });
    return response.data;
  }

  async getOrganizations(): Promise<any[]> {
    const response = await this.client.get('/organizations');
    return response.data;
  }

  async createOrganization(organization: any): Promise<any> {
    const response = await this.client.post('/organizations', organization);
    return response.data;
  }

  async inviteUserToOrganization(orgId: string, email: string, role: string): Promise<void> {
    await this.client.post(`/organizations/${orgId}/invitations`, {
      email,
      role,
    });
  }

  async getAuditLogs(filters: any = {}): Promise<any[]> {
    const response = await this.client.get('/audit-logs', { params: filters });
    return response.data;
  }

  async updateUserProfile(updates: any): Promise<any> {
    const response = await this.client.patch('/profile', updates);
    return response.data;
  }

  async changePassword(currentPassword: string, newPassword: string): Promise<void> {
    await this.client.post('/auth/change-password', {
      current_password: currentPassword,
      new_password: newPassword,
    });
  }

  async setupTwoFactor(): Promise<any> {
    const response = await this.client.post('/auth/2fa/setup');
    return response.data;
  }

  async verifyTwoFactor(token: string): Promise<void> {
    await this.client.post('/auth/2fa/verify', { token });
  }

  async getNotificationSettings(): Promise<any> {
    const response = await this.client.get('/notifications/settings');
    return response.data;
  }

  async updateNotificationSettings(settings: any): Promise<void> {
    await this.client.patch('/notifications/settings', settings);
  }

  // Utility methods
  isAuthenticated(): boolean {
    return !!this.authToken;
  }

  getCurrentUser(): any | null {
    if (!this.authToken) return null;
    
    try {
      const decoded = jwt.decode(this.authToken) as any;
      return decoded;
    } catch {
      return null;
    }
  }

  setAuthToken(token: string): void {
    this.authToken = token;
  }

  clearAuthToken(): void {
    this.authToken = undefined;
  }
}