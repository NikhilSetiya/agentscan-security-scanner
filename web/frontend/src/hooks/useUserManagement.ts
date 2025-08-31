import { useState, useEffect, useCallback } from 'react'
import { userManagement, type UserProfile, type UserPreferences, type Organization, type UserRole } from '../services/userManagement'
import { useAuthContext } from '../contexts/AuthContext'

export function useUserProfile() {
  const { user, isAuthenticated } = useAuthContext()
  const [profile, setProfile] = useState<UserProfile | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchProfile = useCallback(async () => {
    if (!isAuthenticated || !user) return

    setIsLoading(true)
    setError(null)

    try {
      const userProfile = await userManagement.getUserProfile()
      setProfile(userProfile)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch profile')
    } finally {
      setIsLoading(false)
    }
  }, [isAuthenticated, user])

  const updateProfile = useCallback(async (updates: Partial<UserProfile>) => {
    setIsLoading(true)
    setError(null)

    try {
      const { error: updateError } = await userManagement.updateUserProfile(updates)
      
      if (updateError) {
        setError(updateError.message || 'Failed to update profile')
        return { success: false, error: updateError }
      }

      // Refresh profile data
      await fetchProfile()
      return { success: true }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to update profile'
      setError(errorMessage)
      return { success: false, error: { message: errorMessage } }
    } finally {
      setIsLoading(false)
    }
  }, [fetchProfile])

  const updatePreferences = useCallback(async (preferences: Partial<UserPreferences>) => {
    setIsLoading(true)
    setError(null)

    try {
      const { error: updateError } = await userManagement.updateUserPreferences(preferences)
      
      if (updateError) {
        setError(updateError.message || 'Failed to update preferences')
        return { success: false, error: updateError }
      }

      // Refresh profile data
      await fetchProfile()
      return { success: true }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to update preferences'
      setError(errorMessage)
      return { success: false, error: { message: errorMessage } }
    } finally {
      setIsLoading(false)
    }
  }, [fetchProfile])

  const uploadAvatar = useCallback(async (file: File) => {
    setIsLoading(true)
    setError(null)

    try {
      const { url, error: uploadError } = await userManagement.uploadAvatar(file)
      
      if (uploadError) {
        setError(uploadError.message || 'Failed to upload avatar')
        return { success: false, error: uploadError }
      }

      // Refresh profile data
      await fetchProfile()
      return { success: true, url }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to upload avatar'
      setError(errorMessage)
      return { success: false, error: { message: errorMessage } }
    } finally {
      setIsLoading(false)
    }
  }, [fetchProfile])

  useEffect(() => {
    fetchProfile()
  }, [fetchProfile])

  return {
    profile,
    isLoading,
    error,
    updateProfile,
    updatePreferences,
    uploadAvatar,
    refetch: fetchProfile
  }
}

export function useUserRoles() {
  const [roles, setRoles] = useState<UserRole[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchRoles = useCallback(async () => {
    setIsLoading(true)
    setError(null)

    try {
      const userRoles = await userManagement.getUserRoles()
      setRoles(userRoles)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch roles')
    } finally {
      setIsLoading(false)
    }
  }, [])

  const assignRole = useCallback(async (userId: string, roleId: string) => {
    setIsLoading(true)
    setError(null)

    try {
      const { error: assignError } = await userManagement.assignUserRole(userId, roleId)
      
      if (assignError) {
        setError(assignError.message || 'Failed to assign role')
        return { success: false, error: assignError }
      }

      return { success: true }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to assign role'
      setError(errorMessage)
      return { success: false, error: { message: errorMessage } }
    } finally {
      setIsLoading(false)
    }
  }, [])

  const removeRole = useCallback(async (userId: string, roleId: string) => {
    setIsLoading(true)
    setError(null)

    try {
      const { error: removeError } = await userManagement.removeUserRole(userId, roleId)
      
      if (removeError) {
        setError(removeError.message || 'Failed to remove role')
        return { success: false, error: removeError }
      }

      return { success: true }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to remove role'
      setError(errorMessage)
      return { success: false, error: { message: errorMessage } }
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchRoles()
  }, [fetchRoles])

  return {
    roles,
    isLoading,
    error,
    assignRole,
    removeRole,
    refetch: fetchRoles
  }
}

export function useOrganization() {
  const { isAuthenticated } = useAuthContext()
  const [organization, setOrganization] = useState<Organization | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchOrganization = useCallback(async () => {
    if (!isAuthenticated) return

    setIsLoading(true)
    setError(null)

    try {
      const org = await userManagement.getUserOrganization()
      setOrganization(org)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch organization')
    } finally {
      setIsLoading(false)
    }
  }, [isAuthenticated])

  const createOrganization = useCallback(async (name: string, slug: string) => {
    setIsLoading(true)
    setError(null)

    try {
      const { organization: newOrg, error: createError } = await userManagement.createOrganization(name, slug)
      
      if (createError) {
        setError(createError.message || 'Failed to create organization')
        return { success: false, error: createError }
      }

      setOrganization(newOrg || null)
      return { success: true, organization: newOrg }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to create organization'
      setError(errorMessage)
      return { success: false, error: { message: errorMessage } }
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchOrganization()
  }, [fetchOrganization])

  return {
    organization,
    isLoading,
    error,
    createOrganization,
    refetch: fetchOrganization
  }
}

export function useUserActivity() {
  const { isAuthenticated } = useAuthContext()
  const [activity, setActivity] = useState<any[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchActivity = useCallback(async (limit = 50) => {
    if (!isAuthenticated) return

    setIsLoading(true)
    setError(null)

    try {
      const userActivity = await userManagement.getUserActivity(limit)
      setActivity(userActivity)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch activity')
    } finally {
      setIsLoading(false)
    }
  }, [isAuthenticated])

  const logActivity = useCallback(async (action: string, details?: any) => {
    try {
      await userManagement.logUserActivity(action, details)
      // Optionally refresh activity log
      await fetchActivity()
    } catch (err) {
      console.error('Failed to log activity:', err)
    }
  }, [fetchActivity])

  useEffect(() => {
    fetchActivity()
  }, [fetchActivity])

  return {
    activity,
    isLoading,
    error,
    logActivity,
    refetch: fetchActivity
  }
}

export function useOnboarding() {
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const completeOnboarding = useCallback(async (onboardingData: {
    full_name: string
    organization_name?: string
    role_in_organization?: string
    use_case?: string
    team_size?: string
  }) => {
    setIsLoading(true)
    setError(null)

    try {
      const { error: onboardingError } = await userManagement.completeOnboarding(onboardingData)
      
      if (onboardingError) {
        setError(onboardingError.message || 'Failed to complete onboarding')
        return { success: false, error: onboardingError }
      }

      return { success: true }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to complete onboarding'
      setError(errorMessage)
      return { success: false, error: { message: errorMessage } }
    } finally {
      setIsLoading(false)
    }
  }, [])

  const sendEmailVerification = useCallback(async () => {
    setIsLoading(true)
    setError(null)

    try {
      const { error: verificationError } = await userManagement.sendEmailVerification()
      
      if (verificationError) {
        setError(verificationError.message || 'Failed to send verification email')
        return { success: false, error: verificationError }
      }

      return { success: true }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to send verification email'
      setError(errorMessage)
      return { success: false, error: { message: errorMessage } }
    } finally {
      setIsLoading(false)
    }
  }, [])

  return {
    isLoading,
    error,
    completeOnboarding,
    sendEmailVerification
  }
}

export function useAccountManagement() {
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const deactivateAccount = useCallback(async () => {
    setIsLoading(true)
    setError(null)

    try {
      const { error: deactivateError } = await userManagement.deactivateAccount()
      
      if (deactivateError) {
        setError(deactivateError.message || 'Failed to deactivate account')
        return { success: false, error: deactivateError }
      }

      return { success: true }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to deactivate account'
      setError(errorMessage)
      return { success: false, error: { message: errorMessage } }
    } finally {
      setIsLoading(false)
    }
  }, [])

  const deleteAccount = useCallback(async () => {
    setIsLoading(true)
    setError(null)

    try {
      const { error: deleteError } = await userManagement.deleteAccount()
      
      if (deleteError) {
        setError(deleteError.message || 'Failed to request account deletion')
        return { success: false, error: deleteError }
      }

      return { success: true }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Failed to request account deletion'
      setError(errorMessage)
      return { success: false, error: { message: errorMessage } }
    } finally {
      setIsLoading(false)
    }
  }, [])

  return {
    isLoading,
    error,
    deactivateAccount,
    deleteAccount
  }
}