import { apiClient } from './api';

// Backend auth types
export interface BackendAuthUser {
  id: string;
  email: string;
  name: string;
  avatar_url: string;
  created_at: string;
  updated_at: string;
}

export interface BackendAuthTokens {
  access_token: string;
  refresh_token: string;
  expires_at: number;
}

export interface BackendAuthResponse {
  success: boolean;
  data?: {
    user: BackendAuthUser;
    tokens: BackendAuthTokens;
  };
  error?: string;
}

export interface GitHubAuthUrlResponse {
  success: boolean;
  data?: {
    auth_url: string;
    state: string;
  };
  error?: string;
}

class BackendAuthService {
  private baseUrl = process.env.NODE_ENV === 'production' 
    ? 'https://agentscan-security-scanner.fly.dev/api/v1' 
    : 'http://localhost:8080/api/v1';

  async getGitHubAuthUrl(): Promise<GitHubAuthUrlResponse> {
    try {
      console.log('[BACKEND_AUTH] Getting GitHub auth URL...');
      const response = await fetch(`${this.baseUrl}/auth/github/url`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      const data = await response.json();
      
      if (!response.ok) {
        console.error('[BACKEND_AUTH] Auth URL error:', data);
        return {
          success: false,
          error: data.error || `HTTP ${response.status}`
        };
      }

      console.log('[BACKEND_AUTH] Auth URL generated successfully');
      return data;
    } catch (error) {
      console.error('[BACKEND_AUTH] Auth URL request failed:', error);
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Auth URL request failed'
      };
    }
  }

  async handleGitHubCallback(code: string, state: string): Promise<BackendAuthResponse> {
    try {
      console.log('[BACKEND_AUTH] GitHub callback started');
      const response = await fetch(`${this.baseUrl}/auth/github/callback`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ code, state }),
      });

      const data = await response.json();
      
      if (!response.ok) {
        console.error('[BACKEND_AUTH] GitHub callback failed:', data);
        return {
          success: false,
          error: data.error || `HTTP ${response.status}`
        };
      }

      console.log('[BACKEND_AUTH] GitHub callback successful');
      
      // Store tokens in localStorage for persistence
      if (data.data?.tokens) {
        localStorage.setItem('auth_tokens', JSON.stringify(data.data.tokens));
        localStorage.setItem('user', JSON.stringify(data.data.user));
        
        // Set token for API client
        apiClient.setAuthToken(data.data.tokens.access_token);
      }

      return data;
    } catch (error) {
      console.error('[BACKEND_AUTH] GitHub callback request failed:', error);
      return {
        success: false,
        error: error instanceof Error ? error.message : 'GitHub callback failed'
      };
    }
  }

  async refreshToken(refreshToken: string): Promise<BackendAuthResponse> {
    try {
      console.log('[BACKEND_AUTH] Refreshing token...');
      const response = await fetch(`${this.baseUrl}/auth/refresh`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ refresh_token: refreshToken }),
      });

      const data = await response.json();
      
      if (!response.ok) {
        console.error('[BACKEND_AUTH] Token refresh failed:', data);
        return {
          success: false,
          error: data.error || `HTTP ${response.status}`
        };
      }

      console.log('[BACKEND_AUTH] Token refreshed successfully');
      
      // Update stored tokens
      if (data.data?.tokens) {
        localStorage.setItem('auth_tokens', JSON.stringify(data.data.tokens));
        apiClient.setAuthToken(data.data.tokens.access_token);
      }

      return data;
    } catch (error) {
      console.error('[BACKEND_AUTH] Token refresh request failed:', error);
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Token refresh failed'
      };
    }
  }

  async signOut(): Promise<void> {
    try {
      console.log('[BACKEND_AUTH] Signing out...');
      
      // Clear local storage
      localStorage.removeItem('auth_tokens');
      localStorage.removeItem('user');
      
      // Clear API client token
      apiClient.setAuthToken(null);
      
      console.log('[BACKEND_AUTH] Signed out successfully');
    } catch (error) {
      console.error('[BACKEND_AUTH] Sign out error:', error);
    }
  }

  getStoredTokens(): BackendAuthTokens | null {
    try {
      const tokens = localStorage.getItem('auth_tokens');
      return tokens ? JSON.parse(tokens) : null;
    } catch {
      return null;
    }
  }

  getStoredUser(): BackendAuthUser | null {
    try {
      const user = localStorage.getItem('user');
      return user ? JSON.parse(user) : null;
    } catch {
      return null;
    }
  }

  isTokenValid(tokens: BackendAuthTokens): boolean {
    return Date.now() < tokens.expires_at * 1000;
  }
}

export const backendAuth = new BackendAuthService();