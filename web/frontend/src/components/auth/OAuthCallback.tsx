import { useEffect } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { backendAuth, BackendAuthUser } from '../../services/backendAuth';
import { useAuth } from '../../contexts/AuthContext';
import { AuthUser } from '../../services/supabaseAuth';

// Convert backend user format to frontend user format
const convertBackendUser = (backendUser: BackendAuthUser): AuthUser => ({
  id: backendUser.id,
  email: backendUser.email,
  name: backendUser.name,
  avatarUrl: backendUser.avatar_url,
  createdAt: backendUser.created_at,
  updatedAt: backendUser.updated_at,
});

export const OAuthCallback: React.FC = () => {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const { dispatch } = useAuth();

  useEffect(() => {
    const handleOAuthCallback = async () => {
      console.log('[OAUTH_CALLBACK] Processing OAuth callback...');
      
      const code = searchParams.get('code');
      const state = searchParams.get('state');
      const error = searchParams.get('error');
      
      // Handle OAuth error
      if (error) {
        console.error('[OAUTH_CALLBACK] OAuth error:', error);
        dispatch({ 
          type: 'AUTH_FAILURE', 
          payload: `OAuth error: ${error}` 
        });
        navigate('/');
        return;
      }

      // Handle missing code
      if (!code) {
        console.error('[OAUTH_CALLBACK] No authorization code received');
        dispatch({ 
          type: 'AUTH_FAILURE', 
          payload: 'No authorization code received' 
        });
        navigate('/');
        return;
      }

      try {
        // Exchange code for JWT tokens using backend auth
        console.log('[OAUTH_CALLBACK] Exchanging code for tokens...');
        const result = await backendAuth.handleGitHubCallback(code, state || '');
        
        if (result.success && result.data) {
          console.log('[OAUTH_CALLBACK] Authentication successful');
          
          // Convert backend user format to frontend format
          const user = convertBackendUser(result.data.user);
          
          // Update auth context with user data
          dispatch({
            type: 'AUTH_SUCCESS',
            payload: {
              user,
              session: {
                user,
                accessToken: result.data.tokens.access_token,
                refreshToken: result.data.tokens.refresh_token,
                expiresAt: result.data.tokens.expires_at
              }
            }
          });
          
          // Redirect to dashboard
          navigate('/', { replace: true });
        } else {
          throw new Error(result.error || 'Authentication failed');
        }
      } catch (error) {
        console.error('[OAUTH_CALLBACK] Authentication failed:', error);
        dispatch({ 
          type: 'AUTH_FAILURE', 
          payload: error instanceof Error ? error.message : 'Authentication failed' 
        });
        navigate('/', { replace: true });
      }
    };

    handleOAuthCallback();
  }, [searchParams, navigate, dispatch]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50">
      <div className="text-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mx-auto mb-4"></div>
        <p className="text-gray-600">Completing authentication...</p>
      </div>
    </div>
  );
};