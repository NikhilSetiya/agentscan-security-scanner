import { describe, it, expect, vi, beforeEach } from 'vitest';
import { supabaseAuth } from '../supabaseAuth';
import { supabase } from '../../lib/supabase';

// Mock Supabase
vi.mock('../../lib/supabase', () => ({
  supabase: {
    auth: {
      signUp: vi.fn(),
      signInWithPassword: vi.fn(),
      signOut: vi.fn(),
      resetPasswordForEmail: vi.fn(),
      getSession: vi.fn(),
      onAuthStateChange: vi.fn(),
      setSession: vi.fn(),
      updateUser: vi.fn(),
    },
    from: vi.fn(() => ({
      select: vi.fn(() => ({
        eq: vi.fn(() => ({
          single: vi.fn(),
        })),
      })),
      insert: vi.fn(() => ({
        select: vi.fn(() => ({
          single: vi.fn(),
        })),
      })),
    })),
  },
}));

// Mock observe logger
vi.mock('../observeLogger', () => ({
  observeLogger: {
    logUserAction: vi.fn(),
    logError: vi.fn(),
  },
}));

describe('SupabaseAuthService', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('signUp', () => {
    it('should successfully sign up a user', async () => {
      const mockUser = {
        id: 'user-123',
        email: 'test@example.com',
        user_metadata: { name: 'Test User' },
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      const mockDbUser = {
        id: 'db-user-123',
        email: 'test@example.com',
        name: 'Test User',
        avatar_url: null,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      (supabase.auth.signUp as any).mockResolvedValue({
        data: { user: mockUser },
        error: null,
      });

      (supabase.from as any).mockReturnValue({
        insert: vi.fn(() => ({
          select: vi.fn(() => ({
            single: vi.fn().mockResolvedValue({
              data: mockDbUser,
              error: null,
            }),
          })),
        })),
      });

      const result = await supabaseAuth.signUp({
        email: 'test@example.com',
        password: 'password123',
        name: 'Test User',
      });

      expect(result.error).toBeNull();
      expect(result.user).toEqual({
        id: 'db-user-123',
        email: 'test@example.com',
        name: 'Test User',
        avatarUrl: undefined,
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-01-01T00:00:00Z',
      });
    });

    it('should handle sign up errors', async () => {
      const mockError = { message: 'Email already exists', name: 'SignUpError', status: 400 };

      (supabase.auth.signUp as any).mockResolvedValue({
        data: { user: null },
        error: mockError,
      });

      const result = await supabaseAuth.signUp({
        email: 'test@example.com',
        password: 'password123',
      });

      expect(result.error).toEqual(mockError);
      expect(result.user).toBeNull();
    });
  });

  describe('signIn', () => {
    it('should successfully sign in a user', async () => {
      const mockUser = {
        id: 'user-123',
        email: 'test@example.com',
      };

      const mockSession = {
        access_token: 'access-token',
        refresh_token: 'refresh-token',
        expires_at: Date.now() / 1000 + 3600,
        user: mockUser,
      };

      const mockDbUser = {
        id: 'db-user-123',
        email: 'test@example.com',
        name: 'Test User',
        avatar_url: null,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      (supabase.auth.signInWithPassword as any).mockResolvedValue({
        data: { user: mockUser, session: mockSession },
        error: null,
      });

      (supabase.from as any).mockReturnValue({
        select: vi.fn(() => ({
          eq: vi.fn(() => ({
            single: vi.fn().mockResolvedValue({
              data: mockDbUser,
              error: null,
            }),
          })),
        })),
      });

      const result = await supabaseAuth.signIn({
        email: 'test@example.com',
        password: 'password123',
      });

      expect(result.error).toBeNull();
      expect(result.user).toBeDefined();
      expect(result.session).toBeDefined();
      expect(result.session?.accessToken).toBe('access-token');
    });

    it('should handle sign in errors', async () => {
      const mockError = { message: 'Invalid credentials', name: 'SignInError', status: 401 };

      (supabase.auth.signInWithPassword as any).mockResolvedValue({
        data: { user: null, session: null },
        error: mockError,
      });

      const result = await supabaseAuth.signIn({
        email: 'test@example.com',
        password: 'wrongpassword',
      });

      expect(result.error).toEqual(mockError);
      expect(result.user).toBeNull();
      expect(result.session).toBeNull();
    });
  });

  describe('signOut', () => {
    it('should successfully sign out', async () => {
      (supabase.auth.signOut as any).mockResolvedValue({
        error: null,
      });

      const result = await supabaseAuth.signOut();

      expect(result.error).toBeNull();
      expect(supabase.auth.signOut).toHaveBeenCalled();
    });
  });

  describe('resetPassword', () => {
    it('should successfully send reset password email', async () => {
      (supabase.auth.resetPasswordForEmail as any).mockResolvedValue({
        error: null,
      });

      const result = await supabaseAuth.resetPassword({
        email: 'test@example.com',
      });

      expect(result.error).toBeNull();
      expect(supabase.auth.resetPasswordForEmail).toHaveBeenCalledWith(
        'test@example.com',
        { redirectTo: expect.stringContaining('/reset-password') }
      );
    });
  });

  describe('getSession', () => {
    it('should return current session', async () => {
      const mockSession = {
        access_token: 'access-token',
        refresh_token: 'refresh-token',
        expires_at: Date.now() / 1000 + 3600,
        user: { id: 'user-123', email: 'test@example.com' },
      };

      const mockDbUser = {
        id: 'db-user-123',
        email: 'test@example.com',
        name: 'Test User',
        avatar_url: null,
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      };

      (supabase.auth.getSession as any).mockResolvedValue({
        data: { session: mockSession },
        error: null,
      });

      (supabase.from as any).mockReturnValue({
        select: vi.fn(() => ({
          eq: vi.fn(() => ({
            single: vi.fn().mockResolvedValue({
              data: mockDbUser,
              error: null,
            }),
          })),
        })),
      });

      const result = await supabaseAuth.getSession();

      expect(result.error).toBeNull();
      expect(result.session).toBeDefined();
      expect(result.session?.accessToken).toBe('access-token');
    });

    it('should return null when no session exists', async () => {
      (supabase.auth.getSession as any).mockResolvedValue({
        data: { session: null },
        error: null,
      });

      const result = await supabaseAuth.getSession();

      expect(result.error).toBeNull();
      expect(result.session).toBeNull();
    });
  });
});