import { supabase } from '../lib/supabase'
import { supabaseAuth } from './supabaseAuth'
import type { AuthUser } from './supabaseAuth'

export interface UserProfile {
  id: string
  email: string
  full_name: string | null
  avatar_url: string | null
  role: string
  organization_id: string | null
  preferences: UserPreferences
  created_at: string
  updated_at: string
  last_sign_in_at: string | null
  email_confirmed_at: string | null
  is_active: boolean
}

export interface UserPreferences {
  theme: 'light' | 'dark' | 'system'
  notifications: {
    email: boolean
    push: boolean
    scan_completion: boolean
    security_alerts: boolean
    weekly_reports: boolean
  }
  dashboard: {
    default_view: 'overview' | 'scans' | 'findings'
    items_per_page: number
    auto_refresh: boolean
  }
  security: {
    two_factor_enabled: boolean
    session_timeout: number // minutes
    require_password_change: boolean
  }
}

export interface UserRole {
  id: string
  name: string
  description: string
  permissions: string[]
  is_system_role: boolean
}

export interface Organization {
  id: string
  name: string
  slug: string
  plan: 'free' | 'pro' | 'enterprise'
  settings: OrganizationSettings
  created_at: string
  updated_at: string
}

export interface OrganizationSettings {
  max_users: number
  max_repositories: number
  scan_frequency_limit: number
  retention_days: number
  sso_enabled: boolean
  audit_logs_enabled: boolean
}

class UserManagementService {
  // User Profile Management
  async getUserProfile(userId?: string): Promise<UserProfile | null> {
    try {
      const targetUserId = userId || supabaseAuth.getCurrentUser()?.id
      if (!targetUserId) {
        throw new Error('No user ID provided')
      }

      const { data, error } = await supabase
        .from('user_profiles')
        .select(`
          *,
          organizations (
            id,
            name,
            slug,
            plan
          )
        `)
        .eq('id', targetUserId)
        .single()

      if (error) {
        console.error('Error fetching user profile:', error)
        return null
      }

      return data
    } catch (error) {
      console.error('Error in getUserProfile:', error)
      return null
    }
  }

  async updateUserProfile(updates: Partial<UserProfile>): Promise<{ error?: any }> {
    try {
      const currentUser = supabaseAuth.getCurrentUser()
      if (!currentUser) {
        return { error: { message: 'Not authenticated' } }
      }

      const { error } = await supabase
        .from('user_profiles')
        .update({
          ...updates,
          updated_at: new Date().toISOString()
        })
        .eq('id', currentUser.id)

      if (error) {
        console.error('Error updating user profile:', error)
        return { error }
      }

      return {}
    } catch (error) {
      console.error('Error in updateUserProfile:', error)
      return { error }
    }
  }

  async updateUserPreferences(preferences: Partial<UserPreferences>): Promise<{ error?: any }> {
    try {
      const currentUser = supabaseAuth.getCurrentUser()
      if (!currentUser) {
        return { error: { message: 'Not authenticated' } }
      }

      // Get current profile to merge preferences
      const currentProfile = await this.getUserProfile()
      if (!currentProfile) {
        return { error: { message: 'Profile not found' } }
      }

      const updatedPreferences = {
        ...currentProfile.preferences,
        ...preferences
      }

      return await this.updateUserProfile({ preferences: updatedPreferences })
    } catch (error) {
      console.error('Error in updateUserPreferences:', error)
      return { error }
    }
  }

  async uploadAvatar(file: File): Promise<{ url?: string; error?: any }> {
    try {
      const currentUser = supabaseAuth.getCurrentUser()
      if (!currentUser) {
        return { error: { message: 'Not authenticated' } }
      }

      // Generate unique filename
      const fileExt = file.name.split('.').pop()
      const fileName = `${currentUser.id}-${Date.now()}.${fileExt}`
      const filePath = `avatars/${fileName}`

      // Upload file to Supabase Storage
      const { error: uploadError } = await supabase.storage
        .from('user-avatars')
        .upload(filePath, file)

      if (uploadError) {
        console.error('Error uploading avatar:', uploadError)
        return { error: uploadError }
      }

      // Get public URL
      const { data: { publicUrl } } = supabase.storage
        .from('user-avatars')
        .getPublicUrl(filePath)

      // Update user profile with new avatar URL
      const { error: updateError } = await this.updateUserProfile({
        avatar_url: publicUrl
      })

      if (updateError) {
        return { error: updateError }
      }

      return { url: publicUrl }
    } catch (error) {
      console.error('Error in uploadAvatar:', error)
      return { error }
    }
  }

  // Role and Permission Management
  async getUserRoles(): Promise<UserRole[]> {
    try {
      const { data, error } = await supabase
        .from('user_roles')
        .select('*')
        .order('name')

      if (error) {
        console.error('Error fetching user roles:', error)
        return []
      }

      return data || []
    } catch (error) {
      console.error('Error in getUserRoles:', error)
      return []
    }
  }

  async assignUserRole(userId: string, roleId: string): Promise<{ error?: any }> {
    try {
      const currentUser = supabaseAuth.getCurrentUser()
      if (!currentUser || !supabaseAuth.hasRole('admin')) {
        return { error: { message: 'Insufficient permissions' } }
      }

      const { error } = await supabase
        .from('user_role_assignments')
        .insert({
          user_id: userId,
          role_id: roleId,
          assigned_by: currentUser.id,
          assigned_at: new Date().toISOString()
        })

      if (error) {
        console.error('Error assigning user role:', error)
        return { error }
      }

      return {}
    } catch (error) {
      console.error('Error in assignUserRole:', error)
      return { error }
    }
  }

  async removeUserRole(userId: string, roleId: string): Promise<{ error?: any }> {
    try {
      const currentUser = supabaseAuth.getCurrentUser()
      if (!currentUser || !supabaseAuth.hasRole('admin')) {
        return { error: { message: 'Insufficient permissions' } }
      }

      const { error } = await supabase
        .from('user_role_assignments')
        .delete()
        .eq('user_id', userId)
        .eq('role_id', roleId)

      if (error) {
        console.error('Error removing user role:', error)
        return { error }
      }

      return {}
    } catch (error) {
      console.error('Error in removeUserRole:', error)
      return { error }
    }
  }

  // Organization Management
  async getUserOrganization(): Promise<Organization | null> {
    try {
      const currentUser = supabaseAuth.getCurrentUser()
      if (!currentUser) {
        return null
      }

      const profile = await this.getUserProfile()
      if (!profile?.organization_id) {
        return null
      }

      const { data, error } = await supabase
        .from('organizations')
        .select('*')
        .eq('id', profile.organization_id)
        .single()

      if (error) {
        console.error('Error fetching organization:', error)
        return null
      }

      return data
    } catch (error) {
      console.error('Error in getUserOrganization:', error)
      return null
    }
  }

  async createOrganization(name: string, slug: string): Promise<{ organization?: Organization; error?: any }> {
    try {
      const currentUser = supabaseAuth.getCurrentUser()
      if (!currentUser) {
        return { error: { message: 'Not authenticated' } }
      }

      const { data, error } = await supabase
        .from('organizations')
        .insert({
          name,
          slug,
          plan: 'free',
          settings: {
            max_users: 5,
            max_repositories: 10,
            scan_frequency_limit: 100,
            retention_days: 30,
            sso_enabled: false,
            audit_logs_enabled: false
          },
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString()
        })
        .select()
        .single()

      if (error) {
        console.error('Error creating organization:', error)
        return { error }
      }

      // Update user profile to link to organization
      await this.updateUserProfile({
        organization_id: data.id,
        role: 'admin' // Creator becomes admin
      })

      return { organization: data }
    } catch (error) {
      console.error('Error in createOrganization:', error)
      return { error }
    }
  }

  // User Activity and Audit
  async getUserActivity(limit = 50): Promise<any[]> {
    try {
      const currentUser = supabaseAuth.getCurrentUser()
      if (!currentUser) {
        return []
      }

      const { data, error } = await supabase
        .from('user_activity_logs')
        .select('*')
        .eq('user_id', currentUser.id)
        .order('created_at', { ascending: false })
        .limit(limit)

      if (error) {
        console.error('Error fetching user activity:', error)
        return []
      }

      return data || []
    } catch (error) {
      console.error('Error in getUserActivity:', error)
      return []
    }
  }

  async logUserActivity(action: string, details?: any): Promise<void> {
    try {
      const currentUser = supabaseAuth.getCurrentUser()
      if (!currentUser) {
        return
      }

      await supabase
        .from('user_activity_logs')
        .insert({
          user_id: currentUser.id,
          action,
          details: details || {},
          ip_address: await this.getUserIP(),
          user_agent: navigator.userAgent,
          created_at: new Date().toISOString()
        })
    } catch (error) {
      console.error('Error logging user activity:', error)
    }
  }

  // User Verification and Onboarding
  async sendEmailVerification(): Promise<{ error?: any }> {
    try {
      const { error } = await supabase.auth.resend({
        type: 'signup',
        email: supabaseAuth.getCurrentUser()?.email || ''
      })

      if (error) {
        console.error('Error sending email verification:', error)
        return { error }
      }

      return {}
    } catch (error) {
      console.error('Error in sendEmailVerification:', error)
      return { error }
    }
  }

  async completeOnboarding(onboardingData: {
    full_name: string
    organization_name?: string
    role_in_organization?: string
    use_case?: string
    team_size?: string
  }): Promise<{ error?: any }> {
    try {
      const currentUser = supabaseAuth.getCurrentUser()
      if (!currentUser) {
        return { error: { message: 'Not authenticated' } }
      }

      // Update user profile with onboarding data
      const { error: profileError } = await this.updateUserProfile({
        full_name: onboardingData.full_name
      })

      if (profileError) {
        return { error: profileError }
      }

      // Create organization if provided
      if (onboardingData.organization_name) {
        const slug = onboardingData.organization_name
          .toLowerCase()
          .replace(/[^a-z0-9]/g, '-')
          .replace(/-+/g, '-')
          .replace(/^-|-$/g, '')

        await this.createOrganization(onboardingData.organization_name, slug)
      }

      // Log onboarding completion
      await this.logUserActivity('onboarding_completed', onboardingData)

      return {}
    } catch (error) {
      console.error('Error in completeOnboarding:', error)
      return { error }
    }
  }

  // Utility methods
  private async getUserIP(): Promise<string> {
    try {
      const response = await fetch('https://api.ipify.org?format=json')
      const data = await response.json()
      return data.ip
    } catch {
      return 'unknown'
    }
  }

  // Account Management
  async deactivateAccount(): Promise<{ error?: any }> {
    try {
      const currentUser = supabaseAuth.getCurrentUser()
      if (!currentUser) {
        return { error: { message: 'Not authenticated' } }
      }

      // Update profile to mark as inactive
      const { error } = await this.updateUserProfile({
        is_active: false
      })

      if (error) {
        return { error }
      }

      // Log account deactivation
      await this.logUserActivity('account_deactivated')

      // Sign out user
      await supabaseAuth.signOut()

      return {}
    } catch (error) {
      console.error('Error in deactivateAccount:', error)
      return { error }
    }
  }

  async deleteAccount(): Promise<{ error?: any }> {
    try {
      const currentUser = supabaseAuth.getCurrentUser()
      if (!currentUser) {
        return { error: { message: 'Not authenticated' } }
      }

      // This would typically involve a more complex process
      // including data anonymization, cleanup, etc.
      
      // For now, we'll just mark the account for deletion
      const { error } = await supabase
        .from('account_deletion_requests')
        .insert({
          user_id: currentUser.id,
          requested_at: new Date().toISOString(),
          reason: 'user_requested'
        })

      if (error) {
        console.error('Error requesting account deletion:', error)
        return { error }
      }

      // Log deletion request
      await this.logUserActivity('account_deletion_requested')

      return {}
    } catch (error) {
      console.error('Error in deleteAccount:', error)
      return { error }
    }
  }
}

export const userManagement = new UserManagementService()