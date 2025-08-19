import React, { createContext, useContext, useReducer, useEffect, ReactNode } from 'react';
import { supabaseAuth, AuthUser, AuthSession, SignInData, SignUpData, ResetPasswordData } from '../services/supabaseAuth';
import type { AuthChangeEvent, Session } from '@supabase/supabase-js';

// Auth State Types
interface AuthState {
  user: AuthUser | null;
  session: AuthSession | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
}

// Auth Actions
type AuthAction =
  | { type: 'AUTH_START' }
  | { type: 'AUTH_SUCCESS'; payload: { user: AuthUser; session: AuthSession } }
  | { type: 'AUTH_FAILURE'; payload: string }
  | { type: 'AUTH_LOGOUT' }
  | { type: 'AUTH_CLEAR_ERROR' }
  | { type: 'AUTH_SESSION_UPDATE'; payload: { user: AuthUser; session: AuthSession } };

// Auth Context Type
interface AuthContextType {
  state: AuthState;
  signIn: (credentials: SignInData) => Promise<boolean>;
  signUp: (credentials: SignUpData) => Promise<boolean>;
  signOut: () => Promise<void>;
  resetPassword: (data: ResetPasswordData) => Promise<boolean>;
  clearError: () => void;
}

// Initial State
const initialState: AuthState = {
  user: null,
  session: null,
  isAuthenticated: false,
  isLoading: true, // Start with loading true to check existing session
  error: null,
};

// Auth Reducer
function authReducer(state: AuthState, action: AuthAction): AuthState {
  switch (action.type) {
    case 'AUTH_START':
      return {
        ...state,
        isLoading: true,
        error: null,
      };
    case 'AUTH_SUCCESS':
      return {
        ...state,
        user: action.payload.user,
        session: action.payload.session,
        isAuthenticated: true,
        isLoading: false,
        error: null,
      };
    case 'AUTH_SESSION_UPDATE':
      return {
        ...state,
        user: action.payload.user,
        session: action.payload.session,
        isAuthenticated: true,
        isLoading: false,
        error: null,
      };
    case 'AUTH_FAILURE':
      return {
        ...state,
        user: null,
        session: null,
        isAuthenticated: false,
        isLoading: false,
        error: action.payload,
      };
    case 'AUTH_LOGOUT':
      return {
        ...state,
        user: null,
        session: null,
        isAuthenticated: false,
        isLoading: false,
        error: null,
      };
    case 'AUTH_CLEAR_ERROR':
      return {
        ...state,
        error: null,
      };
    default:
      return state;
  }
}

// Create Context
const AuthContext = createContext<AuthContextType | undefined>(undefined);

// Auth Provider Props
interface AuthProviderProps {
  children: ReactNode;
}

// Auth Provider Component
export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const [state, dispatch] = useReducer(authReducer, initialState);

  // Check for existing session on mount and listen for auth changes
  useEffect(() => {
    let mounted = true;

    const initializeAuth = async () => {
      try {
        console.log('[AUTH] Initializing authentication...');
        
        // Get current session
        const { session, error } = await supabaseAuth.getSession();
        
        if (!mounted) return;

        if (error) {
          console.error('[AUTH] Session check error:', error);
          dispatch({ type: 'AUTH_FAILURE', payload: error.message });
          return;
        }

        if (session) {
          console.log('[AUTH] Found existing session');
          dispatch({ 
            type: 'AUTH_SUCCESS', 
            payload: { user: session.user, session } 
          });
        } else {
          console.log('[AUTH] No existing session found');
          dispatch({ type: 'AUTH_LOGOUT' });
        }
      } catch (error) {
        if (!mounted) return;
        console.error('[AUTH] Initialize auth exception:', error);
        dispatch({ 
          type: 'AUTH_FAILURE', 
          payload: error instanceof Error ? error.message : 'Authentication initialization failed' 
        });
      }
    };

    // Set up auth state change listener
    const { data: { subscription } } = supabaseAuth.onAuthStateChange(
      async (event: AuthChangeEvent, session: Session | null) => {
        if (!mounted) return;

        console.log('[AUTH] Auth state changed:', event, session?.user?.email);

        switch (event) {
          case 'SIGNED_IN':
            if (session) {
              try {
                const { session: authSession, error } = await supabaseAuth.getSession();
                if (error) {
                  dispatch({ type: 'AUTH_FAILURE', payload: error.message });
                } else if (authSession) {
                  dispatch({ 
                    type: 'AUTH_SUCCESS', 
                    payload: { user: authSession.user, session: authSession } 
                  });
                }
              } catch (error) {
                dispatch({ 
                  type: 'AUTH_FAILURE', 
                  payload: error instanceof Error ? error.message : 'Sign in failed' 
                });
              }
            }
            break;
          
          case 'SIGNED_OUT':
            dispatch({ type: 'AUTH_LOGOUT' });
            break;
          
          case 'TOKEN_REFRESHED':
            if (session) {
              try {
                const { session: authSession, error } = await supabaseAuth.getSession();
                if (error) {
                  console.warn('[AUTH] Token refresh session error:', error);
                } else if (authSession) {
                  dispatch({ 
                    type: 'AUTH_SESSION_UPDATE', 
                    payload: { user: authSession.user, session: authSession } 
                  });
                }
              } catch (error) {
                console.warn('[AUTH] Token refresh exception:', error);
              }
            }
            break;
          
          case 'PASSWORD_RECOVERY':
            // Handle password recovery if needed
            break;
          
          default:
            break;
        }
      }
    );

    // Initialize auth
    initializeAuth();

    // Cleanup
    return () => {
      mounted = false;
      subscription.unsubscribe();
    };
  }, []);

  // Sign in function
  const signIn = async (credentials: SignInData): Promise<boolean> => {
    console.log('[AUTH] Starting sign in process...');
    dispatch({ type: 'AUTH_START' });

    try {
      const { user, session, error } = await supabaseAuth.signIn(credentials);

      if (error) {
        console.error('[AUTH] Sign in failed:', error);
        dispatch({ type: 'AUTH_FAILURE', payload: error.message });
        return false;
      }

      if (user && session) {
        console.log('[AUTH] Sign in successful:', user.email);
        dispatch({ type: 'AUTH_SUCCESS', payload: { user, session } });
        return true;
      }

      console.error('[AUTH] Sign in failed - no user or session returned');
      dispatch({ type: 'AUTH_FAILURE', payload: 'Sign in failed' });
      return false;
    } catch (error) {
      console.error('[AUTH] Sign in exception:', error);
      const errorMessage = error instanceof Error ? error.message : 'Sign in failed';
      dispatch({ type: 'AUTH_FAILURE', payload: errorMessage });
      return false;
    }
  };

  // Sign up function
  const signUp = async (credentials: SignUpData): Promise<boolean> => {
    console.log('[AUTH] Starting sign up process...');
    dispatch({ type: 'AUTH_START' });

    try {
      const { user, error } = await supabaseAuth.signUp(credentials);

      if (error) {
        console.error('[AUTH] Sign up failed:', error);
        dispatch({ type: 'AUTH_FAILURE', payload: error.message });
        return false;
      }

      if (user) {
        console.log('[AUTH] Sign up successful:', user.email);
        // Note: User will need to verify email before they can sign in
        dispatch({ type: 'AUTH_LOGOUT' }); // Clear loading state
        return true;
      }

      console.error('[AUTH] Sign up failed - no user returned');
      dispatch({ type: 'AUTH_FAILURE', payload: 'Sign up failed' });
      return false;
    } catch (error) {
      console.error('[AUTH] Sign up exception:', error);
      const errorMessage = error instanceof Error ? error.message : 'Sign up failed';
      dispatch({ type: 'AUTH_FAILURE', payload: errorMessage });
      return false;
    }
  };

  // Sign out function
  const signOut = async (): Promise<void> => {
    console.log('[AUTH] Starting sign out process...');
    
    try {
      const { error } = await supabaseAuth.signOut();
      if (error) {
        console.error('[AUTH] Sign out error:', error);
        // Still clear local state even if server sign out fails
      }
    } catch (error) {
      console.error('[AUTH] Sign out exception:', error);
      // Still clear local state even if server sign out fails
    }
    
    // Auth state change listener will handle the logout dispatch
  };

  // Reset password function
  const resetPassword = async (data: ResetPasswordData): Promise<boolean> => {
    console.log('[AUTH] Starting password reset process...');
    
    try {
      const { error } = await supabaseAuth.resetPassword(data);

      if (error) {
        console.error('[AUTH] Password reset failed:', error);
        dispatch({ type: 'AUTH_FAILURE', payload: error.message });
        return false;
      }

      console.log('[AUTH] Password reset email sent successfully');
      return true;
    } catch (error) {
      console.error('[AUTH] Password reset exception:', error);
      const errorMessage = error instanceof Error ? error.message : 'Password reset failed';
      dispatch({ type: 'AUTH_FAILURE', payload: errorMessage });
      return false;
    }
  };

  // Clear error function
  const clearError = (): void => {
    dispatch({ type: 'AUTH_CLEAR_ERROR' });
  };

  const contextValue: AuthContextType = {
    state,
    signIn,
    signUp,
    signOut,
    resetPassword,
    clearError,
  };

  return (
    <AuthContext.Provider value={contextValue}>
      {children}
    </AuthContext.Provider>
  );
};

// Custom hook to use auth context
export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};

// Higher-order component for protected routes
interface ProtectedRouteProps {
  children: ReactNode;
  fallback?: ReactNode;
}

export const ProtectedRoute: React.FC<ProtectedRouteProps> = ({ 
  children, 
  fallback = <div>Please log in to access this page.</div> 
}) => {
  const { state } = useAuth();

  if (state.isLoading) {
    return <div>Loading...</div>;
  }

  if (!state.isAuthenticated) {
    return <>{fallback}</>;
  }

  return <>{children}</>;
};