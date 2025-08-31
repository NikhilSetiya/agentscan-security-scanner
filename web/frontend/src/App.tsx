
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { Suspense, lazy, useState, useEffect } from 'react';
import { Layout } from './components/layout/Layout';
import { OnboardingFlow, useOnboarding } from './components/onboarding/OnboardingFlow';
import { GlobalShortcutsHelp } from './components/ui/KeyboardShortcutsHelp';
import { useGlobalShortcuts } from './hooks/useKeyboardShortcuts';
import { PageTransition } from './components/ui/Transitions';
import { ErrorBoundary } from './components/ErrorBoundary';
import { AuthProvider, useAuthContext } from './contexts/AuthContext';
import { ProtectedRoute } from './components/ProtectedRoute';
import { LoginPage } from './pages/auth/LoginPage';
import { SignupPage } from './pages/auth/SignupPage';
import { ForgotPasswordPage } from './pages/auth/ForgotPasswordPage';
import { ResetPasswordPage } from './pages/auth/ResetPasswordPage';
import { AuthCallbackPage } from './pages/auth/AuthCallbackPage';
import { ApiDebugPanel } from './components/debug/ApiDebugPanel';
import { observeLogger } from './services/observeLogger';
import './styles/globals.css';

// Lazy load heavy components
const Dashboard = lazy(() => import('./pages/Dashboard').then(module => ({ default: module.Dashboard })));
const Scans = lazy(() => import('./pages/Scans').then(module => ({ default: module.Scans })));
const ScanResults = lazy(() => import('./pages/ScanResults').then(module => ({ default: module.ScanResults })));
const Findings = lazy(() => import('./pages/Findings').then(module => ({ default: module.Findings })));
const Reports = lazy(() => import('./pages/Reports').then(module => ({ default: module.Reports })));
const Security = lazy(() => import('./pages/Security').then(module => ({ default: module.Security })));
const Activity = lazy(() => import('./pages/Activity').then(module => ({ default: module.Activity })));
const Settings = lazy(() => import('./pages/Settings').then(module => ({ default: module.Settings })));


// Loading fallback component
const PageLoader = () => (
  <div className="flex items-center justify-center min-h-96">
    <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
  </div>
);

function AppContent() {
  const { user, isLoading, isAuthenticated } = useAuthContext();
  const { shortcuts } = useGlobalShortcuts();
  const {
    isOnboardingOpen,
    completeOnboarding,
    closeOnboarding,
  } = useOnboarding();
  const [isDebugPanelOpen, setIsDebugPanelOpen] = useState(false);

  // Add keyboard shortcut for debug panel (Ctrl+Shift+D)
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.ctrlKey && event.shiftKey && event.key === 'D') {
        event.preventDefault();
        setIsDebugPanelOpen(true);
        observeLogger.logUserAction('debug_panel_opened', {
          trigger: 'keyboard_shortcut'
        });
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  // Show loading while checking authentication
  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-screen">
        <div className="animate-spin rounded-full h-32 w-32 border-b-2 border-gray-900"></div>
      </div>
    );
  }

  return (
    <Router>
      <Routes>
        {/* Public routes */}
        <Route path="/login" element={<LoginPage />} />
        <Route path="/signup" element={<SignupPage />} />
        <Route path="/forgot-password" element={<ForgotPasswordPage />} />
        <Route path="/reset-password" element={<ResetPasswordPage />} />
        <Route path="/auth/callback" element={<AuthCallbackPage />} />
        
        {/* Protected routes */}
        <Route path="/*" element={
          <ProtectedRoute>
            <Layout>
              <PageTransition>
                <Suspense fallback={<PageLoader />}>
                  <Routes>
                    <Route path="/" element={<Dashboard />} />
                    <Route path="/dashboard" element={<Dashboard />} />
                    <Route path="/scans" element={<Scans />} />
                    <Route path="/scans/:id" element={<ScanResults />} />
                    <Route path="/findings" element={<Findings />} />
                    <Route path="/reports" element={<Reports />} />
                    <Route path="/security" element={<Security />} />
                    <Route path="/activity" element={<Activity />} />
                    <Route path="/settings" element={<Settings />} />
                  </Routes>
                </Suspense>
              </PageTransition>
            </Layout>
          </ProtectedRoute>
        } />
      </Routes>
      
      {/* Global components - only show when authenticated */}
      {isAuthenticated && (
        <>
          <OnboardingFlow
            isOpen={isOnboardingOpen}
            onClose={closeOnboarding}
            onComplete={completeOnboarding}
          />
          
          <GlobalShortcutsHelp shortcuts={shortcuts} />
          
          {/* Debug Panel */}
          <ApiDebugPanel 
            isOpen={isDebugPanelOpen} 
            onClose={() => setIsDebugPanelOpen(false)} 
          />
        </>
      )}
    </Router>
  );
}

function App() {
  return (
    <ErrorBoundary>
      <AuthProvider>
        <AppContent />
      </AuthProvider>
    </ErrorBoundary>
  );
}

export default App;