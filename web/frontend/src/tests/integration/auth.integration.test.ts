import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { AuthProvider } from '../../contexts/AuthContext'
import { LoginForm } from '../../components/auth/LoginForm'
import { SignupForm } from '../../components/auth/SignupForm'
import { UserProfile } from '../../components/auth/UserProfile'
import { supabaseAuth } from '../../services/supabaseAuth'
import { observeLogger } from '../../services/observeLogger'

// Mock Supabase auth service
vi.mock('../../services/supabaseAuth', () => ({
  supabaseAuth: {
    signIn: vi.fn(),
    signUp: vi.fn(),
    signOut: vi.fn(),
    getCurrentUser: vi.fn(),
    getCurrentSession: vi.fn(),
    isAuthenticated: vi.fn(),
    isEmailConfirmed: vi.fn(),
    hasRole: vi.fn(),
    onAuthStateChange: vi.fn(() => () => {}),
    updateUser: vi.fn(),
    resetPassword: vi.fn(),
    signInWithProvider: vi.fn(),
  }
}))

// Mock observe logger
vi.mock('../../services/observeLogger', () => ({
  observeLogger: {
    logEvent: vi.fn(),
    logError: vi.fn(),
    logUserAction: vi.fn(),
  }
}))

const renderWithProviders = (component: React.ReactElement) => {
  return render(
    <BrowserRouter>
      <AuthProvider>
        {component}
      </AuthProvider>
    </BrowserRouter>
  )
}

describe('Authentication Integration Tests', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    // Reset auth state
    vi.mocked(supabaseAuth.getCurrentUser).mockReturnValue(null)
    vi.mocked(supabaseAuth.getCurrentSession).mockReturnValue(null)
    vi.mocked(supabaseAuth.isAuthenticated).mockReturnValue(false)
    vi.mocked(supabaseAuth.isEmailConfirmed).mockReturnValue(false)
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('Login Flow', () => {
    it('should handle successful login', async () => {
      const mockUser = {
        id: 'test-user-id',
        email: 'test@example.com',
        name: 'Test User',
        role: 'user',
        provider: 'supabase',
        emailConfirmed: true,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      }

      const mockSession = {
        accessToken: 'mock-access-token',
        refreshToken: 'mock-refresh-token',
        expiresAt: Date.now() + 3600000,
        expiresIn: 3600,
        tokenType: 'bearer',
        user: mockUser,
      }

      vi.mocked(supabaseAuth.signIn).mockResolvedValue({
        user: mockUser,
        session: mockSession,
        error: null,
      })

      renderWithProviders(<LoginForm />)

      // Fill in login form
      const emailInput = screen.getByLabelText(/email address/i)
      const passwordInput = screen.getByLabelText(/password/i)
      const submitButton = screen.getByRole('button', { name: /sign in/i })

      fireEvent.change(emailInput, { target: { value: 'test@example.com' } })
      fireEvent.change(passwordInput, { target: { value: 'testpassword123' } })
      fireEvent.click(submitButton)

      // Wait for login to complete
      await waitFor(() => {
        expect(supabaseAuth.signIn).toHaveBeenCalledWith({
          email: 'test@example.com',
          password: 'testpassword123',
        })
      })

      // Verify observe logging
      expect(observeLogger.logUserAction).toHaveBeenCalledWith(
        'login_attempt',
        expect.objectContaining({
          email: 'test@example.com',
        })
      )
    })

    it('should handle login failure', async () => {
      vi.mocked(supabaseAuth.signIn).mockResolvedValue({
        user: null,
        session: null,
        error: { message: 'Invalid credentials' },
      })

      renderWithProviders(<LoginForm />)

      const emailInput = screen.getByLabelText(/email address/i)
      const passwordInput = screen.getByLabelText(/password/i)
      const submitButton = screen.getByRole('button', { name: /sign in/i })

      fireEvent.change(emailInput, { target: { value: 'test@example.com' } })
      fireEvent.change(passwordInput, { target: { value: 'wrongpassword' } })
      fireEvent.click(submitButton)

      // Wait for error to appear
      await waitFor(() => {
        expect(screen.getByText(/invalid credentials/i)).toBeInTheDocument()
      })

      // Verify error logging
      expect(observeLogger.logError).toHaveBeenCalledWith(
        expect.any(Error),
        expect.objectContaining({
          context: 'login_failure',
          email: 'test@example.com',
        })
      )
    })

    it('should validate form inputs', async () => {
      renderWithProviders(<LoginForm />)

      const submitButton = screen.getByRole('button', { name: /sign in/i })
      fireEvent.click(submitButton)

      // Wait for validation errors
      await waitFor(() => {
        expect(screen.getByText(/please enter a valid email address/i)).toBeInTheDocument()
        expect(screen.getByText(/password must be at least 6 characters/i)).toBeInTheDocument()
      })

      // Should not call signIn with invalid data
      expect(supabaseAuth.signIn).not.toHaveBeenCalled()
    })
  })

  describe('Registration Flow', () => {
    it('should handle successful registration', async () => {
      const mockUser = {
        id: 'new-user-id',
        email: 'newuser@example.com',
        name: 'New User',
        role: 'user',
        provider: 'supabase',
        emailConfirmed: false,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      }

      vi.mocked(supabaseAuth.signUp).mockResolvedValue({
        user: mockUser,
        session: null, // Email confirmation required
        error: null,
      })

      renderWithProviders(<SignupForm />)

      // Fill in registration form
      const nameInput = screen.getByLabelText(/full name/i)
      const emailInput = screen.getByLabelText(/email address/i)
      const passwordInput = screen.getByLabelText(/^password$/i)
      const confirmPasswordInput = screen.getByLabelText(/confirm password/i)
      const termsCheckbox = screen.getByLabelText(/i agree to the/i)
      const submitButton = screen.getByRole('button', { name: /create account/i })

      fireEvent.change(nameInput, { target: { value: 'New User' } })
      fireEvent.change(emailInput, { target: { value: 'newuser@example.com' } })
      fireEvent.change(passwordInput, { target: { value: 'NewPassword123' } })
      fireEvent.change(confirmPasswordInput, { target: { value: 'NewPassword123' } })
      fireEvent.click(termsCheckbox)
      fireEvent.click(submitButton)

      // Wait for registration to complete
      await waitFor(() => {
        expect(supabaseAuth.signUp).toHaveBeenCalledWith({
          email: 'newuser@example.com',
          password: 'NewPassword123',
          fullName: 'New User',
        })
      })

      // Should show email verification message
      await waitFor(() => {
        expect(screen.getByText(/check your email/i)).toBeInTheDocument()
      })

      // Verify observe logging
      expect(observeLogger.logUserAction).toHaveBeenCalledWith(
        'registration_attempt',
        expect.objectContaining({
          email: 'newuser@example.com',
        })
      )
    })

    it('should validate password strength', async () => {
      renderWithProviders(<SignupForm />)

      const passwordInput = screen.getByLabelText(/^password$/i)
      
      // Test weak password
      fireEvent.change(passwordInput, { target: { value: 'weak' } })
      
      await waitFor(() => {
        expect(screen.getByText(/password must contain at least one uppercase letter/i)).toBeInTheDocument()
      })

      // Test strong password
      fireEvent.change(passwordInput, { target: { value: 'StrongPassword123' } })
      
      await waitFor(() => {
        expect(screen.getByText(/strong/i)).toBeInTheDocument()
      })
    })

    it('should validate password confirmation', async () => {
      renderWithProviders(<SignupForm />)

      const passwordInput = screen.getByLabelText(/^password$/i)
      const confirmPasswordInput = screen.getByLabelText(/confirm password/i)
      const submitButton = screen.getByRole('button', { name: /create account/i })

      fireEvent.change(passwordInput, { target: { value: 'Password123' } })
      fireEvent.change(confirmPasswordInput, { target: { value: 'DifferentPassword123' } })
      fireEvent.click(submitButton)

      await waitFor(() => {
        expect(screen.getByText(/passwords don't match/i)).toBeInTheDocument()
      })
    })
  })

  describe('OAuth Integration', () => {
    it('should handle OAuth sign-in', async () => {
      vi.mocked(supabaseAuth.signInWithProvider).mockResolvedValue({
        data: { url: 'https://oauth-redirect-url.com' },
        error: null,
      })

      renderWithProviders(<LoginForm />)

      const googleButton = screen.getByRole('button', { name: /google/i })
      fireEvent.click(googleButton)

      await waitFor(() => {
        expect(supabaseAuth.signInWithProvider).toHaveBeenCalledWith('google')
      })

      // Verify observe logging
      expect(observeLogger.logUserAction).toHaveBeenCalledWith(
        'oauth_signin_attempt',
        expect.objectContaining({
          provider: 'google',
        })
      )
    })

    it('should handle OAuth errors', async () => {
      vi.mocked(supabaseAuth.signInWithProvider).mockResolvedValue({
        data: null,
        error: { message: 'OAuth provider error' },
      })

      renderWithProviders(<LoginForm />)

      const githubButton = screen.getByRole('button', { name: /github/i })
      fireEvent.click(githubButton)

      await waitFor(() => {
        expect(screen.getByText(/oauth provider error/i)).toBeInTheDocument()
      })
    })
  })

  describe('User Profile Management', () => {
    beforeEach(() => {
      const mockUser = {
        id: 'test-user-id',
        email: 'test@example.com',
        name: 'Test User',
        role: 'user',
        provider: 'supabase',
        emailConfirmed: true,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      }

      vi.mocked(supabaseAuth.getCurrentUser).mockReturnValue(mockUser)
      vi.mocked(supabaseAuth.isAuthenticated).mockReturnValue(true)
      vi.mocked(supabaseAuth.isEmailConfirmed).mockReturnValue(true)
    })

    it('should display user profile information', async () => {
      renderWithProviders(<UserProfile />)

      await waitFor(() => {
        expect(screen.getByText('Test User')).toBeInTheDocument()
        expect(screen.getByText('test@example.com')).toBeInTheDocument()
        expect(screen.getByText('user')).toBeInTheDocument()
      })
    })

    it('should handle profile updates', async () => {
      vi.mocked(supabaseAuth.updateUser).mockResolvedValue({
        user: {
          id: 'test-user-id',
          email: 'updated@example.com',
          name: 'Updated User',
          role: 'user',
          provider: 'supabase',
          emailConfirmed: true,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        },
        session: null,
        error: null,
      })

      renderWithProviders(<UserProfile />)

      // Navigate to profile tab and update name
      const nameInput = screen.getByDisplayValue('Test User')
      const updateButton = screen.getByRole('button', { name: /update profile/i })

      fireEvent.change(nameInput, { target: { value: 'Updated User' } })
      fireEvent.click(updateButton)

      await waitFor(() => {
        expect(supabaseAuth.updateUser).toHaveBeenCalledWith({
          email: 'test@example.com',
          data: {
            full_name: 'Updated User',
          },
        })
      })

      // Verify observe logging
      expect(observeLogger.logUserAction).toHaveBeenCalledWith(
        'profile_update',
        expect.objectContaining({
          field: 'name',
        })
      )
    })
  })

  describe('Session Management', () => {
    it('should handle session expiration', async () => {
      // Mock expired session
      vi.mocked(supabaseAuth.getCurrentSession).mockReturnValue({
        accessToken: 'expired-token',
        refreshToken: 'refresh-token',
        expiresAt: Date.now() - 1000, // Expired
        expiresIn: -1,
        tokenType: 'bearer',
        user: {
          id: 'test-user-id',
          email: 'test@example.com',
          name: 'Test User',
          role: 'user',
          provider: 'supabase',
          emailConfirmed: true,
          createdAt: new Date().toISOString(),
          updatedAt: new Date().toISOString(),
        },
      })

      vi.mocked(supabaseAuth.refreshSession).mockResolvedValue({
        user: null,
        session: null,
        error: { message: 'Session expired' },
      })

      renderWithProviders(<UserProfile />)

      // Should trigger session refresh and handle failure
      await waitFor(() => {
        expect(supabaseAuth.refreshSession).toHaveBeenCalled()
      })

      // Verify error logging
      expect(observeLogger.logError).toHaveBeenCalledWith(
        expect.any(Error),
        expect.objectContaining({
          context: 'session_refresh_failed',
        })
      )
    })

    it('should handle successful session refresh', async () => {
      const mockUser = {
        id: 'test-user-id',
        email: 'test@example.com',
        name: 'Test User',
        role: 'user',
        provider: 'supabase',
        emailConfirmed: true,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      }

      const mockSession = {
        accessToken: 'new-access-token',
        refreshToken: 'new-refresh-token',
        expiresAt: Date.now() + 3600000,
        expiresIn: 3600,
        tokenType: 'bearer',
        user: mockUser,
      }

      vi.mocked(supabaseAuth.refreshSession).mockResolvedValue({
        user: mockUser,
        session: mockSession,
        error: null,
      })

      // Simulate session refresh
      await supabaseAuth.refreshSession()

      expect(observeLogger.logEvent).toHaveBeenCalledWith(
        'session_refreshed',
        expect.objectContaining({
          user_id: 'test-user-id',
        })
      )
    })
  })

  describe('Error Handling', () => {
    it('should handle network errors gracefully', async () => {
      vi.mocked(supabaseAuth.signIn).mockRejectedValue(new Error('Network error'))

      renderWithProviders(<LoginForm />)

      const emailInput = screen.getByLabelText(/email address/i)
      const passwordInput = screen.getByLabelText(/password/i)
      const submitButton = screen.getByRole('button', { name: /sign in/i })

      fireEvent.change(emailInput, { target: { value: 'test@example.com' } })
      fireEvent.change(passwordInput, { target: { value: 'testpassword123' } })
      fireEvent.click(submitButton)

      await waitFor(() => {
        expect(screen.getByText(/an unexpected error occurred/i)).toBeInTheDocument()
      })

      // Verify error logging
      expect(observeLogger.logError).toHaveBeenCalledWith(
        expect.any(Error),
        expect.objectContaining({
          context: 'login_network_error',
        })
      )
    })

    it('should handle API rate limiting', async () => {
      vi.mocked(supabaseAuth.signIn).mockResolvedValue({
        user: null,
        session: null,
        error: { message: 'Too many requests' },
      })

      renderWithProviders(<LoginForm />)

      const emailInput = screen.getByLabelText(/email address/i)
      const passwordInput = screen.getByLabelText(/password/i)
      const submitButton = screen.getByRole('button', { name: /sign in/i })

      fireEvent.change(emailInput, { target: { value: 'test@example.com' } })
      fireEvent.change(passwordInput, { target: { value: 'testpassword123' } })
      fireEvent.click(submitButton)

      await waitFor(() => {
        expect(screen.getByText(/too many requests/i)).toBeInTheDocument()
      })

      // Verify rate limit logging
      expect(observeLogger.logEvent).toHaveBeenCalledWith(
        'rate_limit_exceeded',
        expect.objectContaining({
          endpoint: 'login',
          user_email: 'test@example.com',
        })
      )
    })
  })

  describe('Accessibility', () => {
    it('should have proper ARIA labels and roles', () => {
      renderWithProviders(<LoginForm />)

      // Check for proper labels
      expect(screen.getByLabelText(/email address/i)).toBeInTheDocument()
      expect(screen.getByLabelText(/password/i)).toBeInTheDocument()

      // Check for proper button roles
      expect(screen.getByRole('button', { name: /sign in/i })).toBeInTheDocument()

      // Check for form structure
      const form = screen.getByRole('form') || screen.getByTestId('login-form')
      expect(form).toBeInTheDocument()
    })

    it('should handle keyboard navigation', async () => {
      renderWithProviders(<LoginForm />)

      const emailInput = screen.getByLabelText(/email address/i)
      const passwordInput = screen.getByLabelText(/password/i)
      const submitButton = screen.getByRole('button', { name: /sign in/i })

      // Tab navigation
      emailInput.focus()
      fireEvent.keyDown(emailInput, { key: 'Tab' })
      expect(passwordInput).toHaveFocus()

      fireEvent.keyDown(passwordInput, { key: 'Tab' })
      expect(submitButton).toHaveFocus()

      // Enter key submission
      fireEvent.change(emailInput, { target: { value: 'test@example.com' } })
      fireEvent.change(passwordInput, { target: { value: 'testpassword123' } })
      fireEvent.keyDown(submitButton, { key: 'Enter' })

      await waitFor(() => {
        expect(supabaseAuth.signIn).toHaveBeenCalled()
      })
    })
  })

  describe('Performance', () => {
    it('should render components within performance budget', async () => {
      const startTime = performance.now()
      
      renderWithProviders(<LoginForm />)
      
      const renderTime = performance.now() - startTime
      
      // Component should render within 100ms
      expect(renderTime).toBeLessThan(100)

      // Log performance metrics
      expect(observeLogger.logEvent).toHaveBeenCalledWith(
        'component_render_performance',
        expect.objectContaining({
          component: 'LoginForm',
          render_time_ms: expect.any(Number),
        })
      )
    })

    it('should handle rapid user interactions', async () => {
      vi.mocked(supabaseAuth.signIn).mockImplementation(() => 
        new Promise(resolve => setTimeout(() => resolve({
          user: null,
          session: null,
          error: { message: 'Invalid credentials' },
        }), 100))
      )

      renderWithProviders(<LoginForm />)

      const submitButton = screen.getByRole('button', { name: /sign in/i })

      // Rapid clicks should not cause multiple submissions
      fireEvent.click(submitButton)
      fireEvent.click(submitButton)
      fireEvent.click(submitButton)

      await waitFor(() => {
        expect(supabaseAuth.signIn).toHaveBeenCalledTimes(1)
      })
    })
  })

  describe('Real-time Features', () => {
    it('should handle auth state changes', async () => {
      const mockUser = {
        id: 'test-user-id',
        email: 'test@example.com',
        name: 'Test User',
        role: 'user',
        provider: 'supabase',
        emailConfirmed: true,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      }

      let authStateCallback: (user: any) => void = () => {}
      
      vi.mocked(supabaseAuth.onAuthStateChange).mockImplementation((callback) => {
        authStateCallback = callback
        return () => {} // Unsubscribe function
      })

      renderWithProviders(<UserProfile />)

      // Simulate auth state change
      authStateCallback(mockUser)

      await waitFor(() => {
        expect(screen.getByText('Test User')).toBeInTheDocument()
      })

      // Verify state change logging
      expect(observeLogger.logEvent).toHaveBeenCalledWith(
        'auth_state_changed',
        expect.objectContaining({
          user_id: 'test-user-id',
          authenticated: true,
        })
      )
    })
  })
})