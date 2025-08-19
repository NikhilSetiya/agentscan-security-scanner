/**
 * Supabase Authentication Service
 * Handles all authentication operations using Supabase Auth
 */

import { supabase } from '../lib/supabase'
import type { 
  AuthError, 
  User, 
  Session,
  AuthChangeEvent,
  SignUpWithPasswordCredentials,
  SignInWithPasswordCredentials
} from '@supabase/supabase-js'

export interface AuthUser {
  id: string
  email: string
  name: string
  avatarUrl?: string
  createdAt: string
  updatedAt: string
}

export interface AuthSession {
  user: AuthUser
  accessToken: string
  refreshToken: string
  expiresAt: number
}

export interface SignUpData {
  email: string
  password: string
  name?: string
}

export interface SignInData {
  email: string
  password: string
}

export interface ResetPasswordData {
  email: string
}

class SupabaseAuthService {
  /**
   * Sign up a new user
   */
  async signUp(data: SignUpData): Promise<{ user: AuthUser | null; error: AuthError | null }> {
    try {
      const credentials: SignUpWithPasswordCredentials = {
        email: data.email,
        password: data.password,
        options: {
          data: {
            name: data.name || data.email.split('@')[0]
          }
        }
      }

      const { data: authData, error } = await supabase.auth.signUp(credentials)

      if (error) {
        console.error('[AUTH] Sign up error:', error)
        return { user: null, error }
      }

      if (!authData.user) {
        return { user: null, error: { message: 'No user returned from sign up', name: 'SignUpError', status: 400 } as AuthError }
      }

      // Create user profile in our database
      const user = await this.createUserProfile(authData.user)
      
      return { user, error: null }
    } catch (error) {
      console.error('[AUTH] Sign up exception:', error)
      return { 
        user: null, 
        error: { 
          message: error instanceof Error ? error.message : 'Sign up failed', 
          name: 'SignUpError',
          status: 500
        } as AuthError 
      }
    }
  }

  /**
   * Sign in an existing user
   */
  async signIn(data: SignInData): Promise<{ user: AuthUser | null; session: AuthSession | null; error: AuthError | null }> {
    try {
      const credentials: SignInWithPasswordCredentials = {
        email: data.email,
        password: data.password
      }

      const { data: authData, error } = await supabase.auth.signInWithPassword(credentials)

      if (error) {
        console.error('[AUTH] Sign in error:', error)
        return { user: null, session: null, error }
      }

      if (!authData.user || !authData.session) {
        return { 
          user: null, 
          session: null, 
          error: { message: 'No user or session returned from sign in', name: 'SignInError', status: 400 } as AuthError 
        }
      }

      // Get or create user profile
      const user = await this.getUserProfile(authData.user.id) || await this.createUserProfile(authData.user)
      const session = this.mapSession(authData.session, user)

      return { user, session, error: null }
    } catch (error) {
      console.error('[AUTH] Sign in exception:', error)
      return { 
        user: null, 
        session: null,
        error: { 
          message: error instanceof Error ? error.message : 'Sign in failed', 
          name: 'SignInError',
          status: 500
        } as AuthError 
      }
    }
  }

  /**
   * Sign out the current user
   */
  async signOut(): Promise<{ error: AuthError | null }> {
    try {
      const { error } = await supabase.auth.signOut()
      
      if (error) {
        console.error('[AUTH] Sign out error:', error)
        return { error }
      }

      return { error: null }
    } catch (error) {
      console.error('[AUTH] Sign out exception:', error)
      return { 
        error: { 
          message: error instanceof Error ? error.message : 'Sign out failed', 
          name: 'SignOutError',
          status: 500
        } as AuthError 
      }
    }
  }

  /**
   * Reset user password
   */
  async resetPassword(data: ResetPasswordData): Promise<{ error: AuthError | null }> {
    try {
      const { error } = await supabase.auth.resetPasswordForEmail(data.email, {
        redirectTo: `${window.location.origin}/reset-password`
      })

      if (error) {
        console.error('[AUTH] Reset password error:', error)
        return { error }
      }

      return { error: null }
    } catch (error) {
      console.error('[AUTH] Reset password exception:', error)
      return { 
        error: { 
          message: error instanceof Error ? error.message : 'Password reset failed', 
          name: 'ResetPasswordError',
          status: 500
        } as AuthError 
      }
    }
  }

  /**
   * Get current session
   */
  async getSession(): Promise<{ session: AuthSession | null; error: AuthError | null }> {
    try {
      const { data: { session }, error } = await supabase.auth.getSession()

      if (error) {
        console.error('[AUTH] Get session error:', error)
        return { session: null, error }
      }

      if (!session) {
        return { session: null, error: null }
      }

      const user = await this.getUserProfile(session.user.id)
      if (!user) {
        return { 
          session: null, 
          error: { message: 'User profile not found', name: 'ProfileError', status: 404 } as AuthError 
        }
      }

      const authSession = this.mapSession(session, user)
      return { session: authSession, error: null }
    } catch (error) {
      console.error('[AUTH] Get session exception:', error)
      return { 
        session: null,
        error: { 
          message: error instanceof Error ? error.message : 'Get session failed', 
          name: 'SessionError',
          status: 500
        } as AuthError 
      }
    }
  }

  /**
   * Listen to authentication state changes
   */
  onAuthStateChange(callback: (event: AuthChangeEvent, session: Session | null) => void) {
    return supabase.auth.onAuthStateChange(callback)
  }

  /**
   * Get user profile from database
   */
  private async getUserProfile(supabaseId: string): Promise<AuthUser | null> {
    try {
      const { data, error } = await supabase
        .from('users')
        .select('*')
        .eq('supabase_id', supabaseId)
        .single()

      if (error || !data) {
        console.warn('[AUTH] User profile not found:', error?.message)
        return null
      }

      return {
        id: data.id,
        email: data.email,
        name: data.name || data.email.split('@')[0],
        avatarUrl: data.avatar_url || undefined,
        createdAt: data.created_at,
        updatedAt: data.updated_at
      }
    } catch (error) {
      console.error('[AUTH] Get user profile exception:', error)
      return null
    }
  }

  /**
   * Create user profile in database
   */
  private async createUserProfile(supabaseUser: User): Promise<AuthUser> {
    try {
      const userData = {
        supabase_id: supabaseUser.id,
        email: supabaseUser.email!,
        name: supabaseUser.user_metadata?.name || supabaseUser.email!.split('@')[0],
        avatar_url: supabaseUser.user_metadata?.avatar_url || null
      }

      const { data, error } = await supabase
        .from('users')
        .insert(userData)
        .select()
        .single()

      if (error) {
        console.error('[AUTH] Create user profile error:', error)
        // If user already exists, try to get it
        const existingUser = await this.getUserProfile(supabaseUser.id)
        if (existingUser) {
          return existingUser
        }
        throw error
      }

      return {
        id: data.id,
        email: data.email,
        name: data.name || data.email.split('@')[0],
        avatarUrl: data.avatar_url || undefined,
        createdAt: data.created_at,
        updatedAt: data.updated_at
      }
    } catch (error) {
      console.error('[AUTH] Create user profile exception:', error)
      // Fallback to basic user data
      return {
        id: supabaseUser.id,
        email: supabaseUser.email!,
        name: supabaseUser.user_metadata?.name || supabaseUser.email!.split('@')[0],
        avatarUrl: supabaseUser.user_metadata?.avatar_url,
        createdAt: supabaseUser.created_at!,
        updatedAt: supabaseUser.updated_at!
      }
    }
  }

  /**
   * Map Supabase session to our session format
   */
  private mapSession(session: Session, user: AuthUser): AuthSession {
    return {
      user,
      accessToken: session.access_token,
      refreshToken: session.refresh_token,
      expiresAt: session.expires_at || 0
    }
  }
}

// Export singleton instance
export const supabaseAuth = new SupabaseAuthService()
export default supabaseAuth