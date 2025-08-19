
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { Suspense, lazy, useState, useEffect } from 'react';
import { Layout } from './components/layout/Layout';
import { OnboardingFlow, useOnboarding } from './components/onboarding/OnboardingFlow';
import { GlobalShortcutsHelp } from './components/ui/KeyboardShortcutsHelp';
import { useGlobalShortcuts } from './hooks/useKeyboardShortcuts';
import { PageTransition } from './components/ui/Transitions';
import { ErrorBoundary } from './components/ErrorBoundary';
import { AuthProvider, ProtectedRoute, useAuth } from './contexts/AuthContext';
import { LoginForm } from './components/auth/LoginForm';
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
  const { state } = useAuth();
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

  // Show login form if not authenticated
  if (!state.isAuthenticated && !state.isLoading) {
    return <LoginForm />;
  }

  // Show loading while checking authentication
  if (state.isLoading) {
    return (
      <div className="loading-container">
        <div className="loading-spinner">Loading AgentScan...</div>
      </div>
    );
  }

  return (
    <Router>
      <Layout>
        <PageTransition>
          <Suspense fallback={<PageLoader />}>
            <Routes>
              <Route path="/" element={
                <ProtectedRoute>
                  <Dashboard />
                </ProtectedRoute>
              } />
              <Route path="/scans" element={
                <ProtectedRoute>
                  <Scans />
                </ProtectedRoute>
              } />
              <Route path="/scans/:id" element={
                <ProtectedRoute>
                  <ScanResults />
                </ProtectedRoute>
              } />
              <Route path="/findings" element={
                <ProtectedRoute>
                  <Findings />
                </ProtectedRoute>
              } />
              <Route path="/reports" element={
                <ProtectedRoute>
                  <Reports />
                </ProtectedRoute>
              } />
              <Route path="/security" element={
                <ProtectedRoute>
                  <Security />
                </ProtectedRoute>
              } />
              <Route path="/activity" element={
                <ProtectedRoute>
                  <Activity />
                </ProtectedRoute>
              } />
              <Route path="/settings" element={
                <ProtectedRoute>
                  <Settings />
                </ProtectedRoute>
              } />
            </Routes>
          </Suspense>
        </PageTransition>
      </Layout>
      
      {/* Global components */}
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