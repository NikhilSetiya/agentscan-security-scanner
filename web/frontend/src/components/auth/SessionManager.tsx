import React, { useEffect, useCallback } from 'react';
import { useAuth } from '../../contexts/AuthContext';
import { supabaseAuth } from '../../services/supabaseAuth';
import { observeLogger } from '../../services/observeLogger';

interface SessionManagerProps {
  children: React.ReactNode;
}

/**
 * SessionManager component handles automatic token refresh and session management
 */
export const SessionManager: React.FC<SessionManagerProps> = ({ children }) => {
  const { state, dispatch } = useAuth();

  // Check if token is close to expiry (within 5 minutes)
  const isTokenNearExpiry = useCallback((expiresAt: number): boolean => {
    const now = Math.floor(Date.now() / 1000);
    const fiveMinutes = 5 * 60;
    return (expiresAt - now) <= fiveMinutes;
  }, []);

  // Refresh token if needed
  const refreshTokenIfNeeded = useCallback(async () => {
    if (!state.session || !state.isAuthenticated) {
      return;
    }

    try {
      if (isTokenNearExpiry(state.session.expiresAt)) {
        console.log('[SESSION] Token near expiry, refreshing...');
        observeLogger.logUserAction('token_refresh_attempt', {
          expiresAt: state.session.expiresAt,
          timeUntilExpiry: state.session.expiresAt - Math.floor(Date.now() / 1000)
        });

        // Get fresh session (Supabase handles token refresh automatically)
        const { session, error } = await supabaseAuth.getSession();
        
        if (error) {
          console.error('[SESSION] Token refresh failed:', error);
          observeLogger.logError('token_refresh_failed', error);
          // Don't sign out immediately, let the auth state change handler deal with it
          return;
        }

        if (session) {
          console.log('[SESSION] Token refreshed successfully');
          observeLogger.logUserAction('token_refresh_success', {
            newExpiresAt: session.expiresAt
          });
          
          dispatch({ 
            type: 'AUTH_SESSION_UPDATE', 
            payload: { user: session.user, session } 
          });
        }
      }
    } catch (error) {
      console.error('[SESSION] Token refresh exception:', error);
      observeLogger.logError('token_refresh_exception', error);
    }
  }, [state.session, state.isAuthenticated, isTokenNearExpiry, dispatch]);

  // Set up periodic token refresh check
  useEffect(() => {
    if (!state.isAuthenticated || !state.session) {
      return;
    }

    // Check token expiry every minute
    const interval = setInterval(refreshTokenIfNeeded, 60 * 1000);

    // Also check immediately if token is already near expiry
    if (isTokenNearExpiry(state.session.expiresAt)) {
      refreshTokenIfNeeded();
    }

    return () => clearInterval(interval);
  }, [state.isAuthenticated, state.session, refreshTokenIfNeeded, isTokenNearExpiry]);

  // Handle page visibility change to refresh token when page becomes visible
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible' && state.isAuthenticated) {
        console.log('[SESSION] Page became visible, checking token...');
        refreshTokenIfNeeded();
      }
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    return () => document.removeEventListener('visibilitychange', handleVisibilityChange);
  }, [state.isAuthenticated, refreshTokenIfNeeded]);

  // Handle window focus to refresh token
  useEffect(() => {
    const handleFocus = () => {
      if (state.isAuthenticated) {
        console.log('[SESSION] Window focused, checking token...');
        refreshTokenIfNeeded();
      }
    };

    window.addEventListener('focus', handleFocus);
    return () => window.removeEventListener('focus', handleFocus);
  }, [state.isAuthenticated, refreshTokenIfNeeded]);

  return <>{children}</>;
};

export default SessionManager;