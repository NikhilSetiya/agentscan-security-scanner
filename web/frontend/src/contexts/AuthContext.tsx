import React, { createContext, useContext, useReducer, useEffect, ReactNode } from 'react';
import { apiClient, User, LoginRequest } from '../services/api';

// Auth State Types
interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
}

// Auth Actions
type AuthAction =
  | { type: 'AUTH_START' }
  | { type: 'AUTH_SUCCESS'; payload: User }
  | { type: 'AUTH_FAILURE'; payload: string }
  | { type: 'AUTH_LOGOUT' }
  | { type: 'AUTH_CLEAR_ERROR' };

// Auth Context Type
interface AuthContextType {
  state: AuthState;
  login: (credentials: LoginRequest) => Promise<boolean>;
  logout: () => Promise<void>;
  clearError: () => void;
}

// Initial State
const initialState: AuthState = {
  user: null,
  isAuthenticated: false,
  isLoading: false,
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
        user: action.payload,
        isAuthenticated: true,
        isLoading: false,
        error: null,
      };
    case 'AUTH_FAILURE':
      return {
        ...state,
        user: null,
        isAuthenticated: false,
        isLoading: false,
        error: action.payload,
      };
    case 'AUTH_LOGOUT':
      return {
        ...state,
        user: null,
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

  // Check for existing authentication on mount
  useEffect(() => {
    const checkAuth = async () => {
      const token = apiClient.getAuthToken();
      if (token) {
        // Verify token is still valid by making a health check or user info request
        try {
          dispatch({ type: 'AUTH_START' });
          const response = await apiClient.healthCheck();
          if (response.error) {
            // Token is invalid, clear it
            dispatch({ type: 'AUTH_LOGOUT' });
          } else {
            // For now, create a mock user since we don't have a user info endpoint
            // In a real implementation, you'd call a /auth/me endpoint
            const mockUser: User = {
              id: 'current-user',
              username: 'Current User',
              email: 'user@example.com',
              role: 'developer',
            };
            dispatch({ type: 'AUTH_SUCCESS', payload: mockUser });
          }
        } catch (error) {
          dispatch({ type: 'AUTH_LOGOUT' });
        }
      }
    };

    checkAuth();
  }, []);

  // Listen for logout events from API client
  useEffect(() => {
    const handleLogout = () => {
      dispatch({ type: 'AUTH_LOGOUT' });
    };

    window.addEventListener('auth:logout', handleLogout);
    return () => window.removeEventListener('auth:logout', handleLogout);
  }, []);

  // Login function
  const login = async (credentials: LoginRequest): Promise<boolean> => {
    dispatch({ type: 'AUTH_START' });

    try {
      const response = await apiClient.login(credentials);

      if (response.error) {
        dispatch({ type: 'AUTH_FAILURE', payload: response.error.error });
        return false;
      }

      if (response.data) {
        dispatch({ type: 'AUTH_SUCCESS', payload: response.data.user });
        return true;
      }

      dispatch({ type: 'AUTH_FAILURE', payload: 'Login failed' });
      return false;
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Login failed';
      dispatch({ type: 'AUTH_FAILURE', payload: errorMessage });
      return false;
    }
  };

  // Logout function
  const logout = async (): Promise<void> => {
    try {
      await apiClient.logout();
    } catch (error) {
      // Even if logout fails on server, clear local state
      console.warn('Logout request failed:', error);
    } finally {
      dispatch({ type: 'AUTH_LOGOUT' });
    }
  };

  // Clear error function
  const clearError = (): void => {
    dispatch({ type: 'AUTH_CLEAR_ERROR' });
  };

  const contextValue: AuthContextType = {
    state,
    login,
    logout,
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