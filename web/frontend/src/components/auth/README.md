# Authentication System

This directory contains the complete Supabase authentication implementation for the AgentScan frontend.

## Components

### Core Components

- **`AuthContext.tsx`** - React context providing authentication state and methods
- **`LoginForm.tsx`** - Comprehensive login/signup/password reset form
- **`SessionManager.tsx`** - Automatic token refresh and session management
- **`PasswordResetForm.tsx`** - Password reset flow with URL token handling
- **`UserProfile.tsx`** - User profile display and management
- **`OAuthCallback.tsx`** - OAuth callback handler for third-party auth

### Services

- **`supabaseAuth.ts`** - Core Supabase authentication service
- **`supabase.ts`** - Supabase client configuration

### Hooks

- **`useAuthState.ts`** - Enhanced authentication hook with utilities

## Features

### ✅ Implemented Features

1. **Email/Password Authentication**
   - User registration with email verification
   - Secure login with password
   - Password strength validation
   - Form validation and error handling

2. **Session Management**
   - Automatic token refresh
   - Session persistence across browser sessions
   - Session validation and expiry handling
   - Automatic logout on token expiration

3. **Password Reset**
   - Email-based password reset flow
   - Secure token validation
   - Password update with confirmation
   - Redirect handling after reset

4. **User Profile Management**
   - User profile display
   - Basic profile information
   - Account management options

5. **Security Features**
   - JWT token validation
   - Secure token storage
   - CSRF protection
   - Input sanitization

6. **Error Handling**
   - Comprehensive error messages
   - User-friendly error display
   - Logging and monitoring integration
   - Graceful fallbacks

7. **Real-time Updates**
   - Auth state change listeners
   - Automatic UI updates
   - Session synchronization across tabs

## Usage

### Basic Setup

```tsx
import { AuthProvider } from './contexts/AuthContext';
import { SessionManager } from './components/auth/SessionManager';

function App() {
  return (
    <AuthProvider>
      <SessionManager>
        {/* Your app content */}
      </SessionManager>
    </AuthProvider>
  );
}
```

### Using Authentication

```tsx
import { useAuth } from './contexts/AuthContext';
// or
import { useAuthState } from './hooks/useAuthState';

function MyComponent() {
  const { state, signIn, signOut } = useAuth();
  
  if (!state.isAuthenticated) {
    return <LoginForm />;
  }
  
  return (
    <div>
      <p>Welcome, {state.user?.name}!</p>
      <button onClick={signOut}>Sign Out</button>
    </div>
  );
}
```

### Protected Routes

```tsx
import { ProtectedRoute } from './contexts/AuthContext';

function App() {
  return (
    <Routes>
      <Route path="/dashboard" element={
        <ProtectedRoute>
          <Dashboard />
        </ProtectedRoute>
      } />
    </Routes>
  );
}
```

## Configuration

### Environment Variables

```env
VITE_SUPABASE_URL=https://your-project.supabase.co
VITE_SUPABASE_ANON_KEY=your-anon-key-here
```

### Supabase Setup

1. Create a Supabase project
2. Configure authentication providers
3. Set up user profiles table
4. Configure email templates
5. Set redirect URLs

## Testing

The authentication system includes comprehensive tests:

```bash
npm test src/services/__tests__/supabaseAuth.test.ts
```

## Security Considerations

1. **Token Security**
   - Tokens are stored securely in browser storage
   - Automatic token refresh prevents expiration
   - Tokens are validated on each request

2. **Input Validation**
   - All user inputs are validated client-side
   - Server-side validation through Supabase
   - XSS protection through input sanitization

3. **Session Management**
   - Sessions expire automatically
   - Concurrent session handling
   - Secure logout across all tabs

## Future Enhancements

- [ ] Multi-factor authentication (MFA)
- [ ] Social login providers (Google, GitHub)
- [ ] Role-based access control (RBAC)
- [ ] Account lockout after failed attempts
- [ ] Advanced password policies
- [ ] Audit logging for security events