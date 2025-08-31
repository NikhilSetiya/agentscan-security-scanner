import { useCallback, useMemo } from 'react';
import { useAuth } from '../contexts/AuthContext';
import { observeLogger } from '../services/observeLogger';

/**
 * Enhanced authentication hook with additional utilities
 */
export const useAuthState = () => {
  const { state, signIn, signUp, signOut, resetPassword, clearError } = useAuth();

  // Check if user has specific permissions
  const hasPermission = useCallback((permission: string): boolean => {
    if (!state.user) return false;
    
    // For now, all authenticated users have basic permissions
    // This can be extended based on user roles/permissions
    const basicPermissions = ['read', 'scan', 'view_findings'];
    return basicPermissions.includes(permission);
  }, [state.user]);

  // Check if user is admin
  const isAdmin = useMemo(() => {
    // This would check user role from the backend
    // For now, return false as we don't have role management yet
    return false;
  }, []);

  // Enhanced sign in with logging
  const enhancedSignIn = useCallback(async (credentials: { email: string; password: string }) => {
    observeLogger.logUserAction('sign_in_attempt', { email: credentials.email });
    
    const success = await signIn(credentials);
    
    if (success) {
      observeLogger.logUserAction('sign_in_success', { 
        email: credentials.email,
        userId: state.user?.id 
      });
    } else {
      observeLogger.logUserAction('sign_in_failed', { 
        email: credentials.email,
        error: state.error 
      });
    }
    
    return success;
  }, [signIn, state.user?.id, state.error]);

  // Enhanced sign up with logging
  const enhancedSignUp = useCallback(async (credentials: { email: string; password: string; name?: string }) => {
    observeLogger.logUserAction('sign_up_attempt', { email: credentials.email });
    
    const success = await signUp(credentials);
    
    if (success) {
      observeLogger.logUserAction('sign_up_success', { email: credentials.email });
    } else {
      observeLogger.logUserAction('sign_up_failed', { 
        email: credentials.email,
        error: state.error 
      });
    }
    
    return success;
  }, [signUp, state.error]);

  // Enhanced sign out with logging
  const enhancedSignOut = useCallback(async () => {
    const userId = state.user?.id;
    const email = state.user?.email;
    
    observeLogger.logUserAction('sign_out_attempt', { userId, email });
    
    await signOut();
    
    observeLogger.logUserAction('sign_out_success', { userId, email });
  }, [signOut, state.user?.id, state.user?.email]);

  // Get user display name
  const userDisplayName = useMemo(() => {
    if (!state.user) return '';
    return state.user.name || state.user.email.split('@')[0];
  }, [state.user]);

  // Get user initials for avatar
  const userInitials = useMemo(() => {
    if (!state.user) return '';
    const name = state.user.name || state.user.email;
    return name
      .split(' ')
      .map(part => part.charAt(0).toUpperCase())
      .slice(0, 2)
      .join('');
  }, [state.user]);

  // Check if session is valid
  const isSessionValid = useMemo(() => {
    if (!state.session) return false;
    
    const now = Math.floor(Date.now() / 1000);
    return state.session.expiresAt > now;
  }, [state.session]);

  // Get time until session expires
  const sessionTimeRemaining = useMemo(() => {
    if (!state.session) return 0;
    
    const now = Math.floor(Date.now() / 1000);
    return Math.max(0, state.session.expiresAt - now);
  }, [state.session]);

  return {
    // Original auth state and methods
    ...state,
    signIn: enhancedSignIn,
    signUp: enhancedSignUp,
    signOut: enhancedSignOut,
    resetPassword,
    clearError,
    
    // Enhanced utilities
    hasPermission,
    isAdmin,
    userDisplayName,
    userInitials,
    isSessionValid,
    sessionTimeRemaining,
    
    // Computed states
    isReady: !state.isLoading,
    needsAuthentication: !state.isAuthenticated && !state.isLoading,
  };
};

export default useAuthState;