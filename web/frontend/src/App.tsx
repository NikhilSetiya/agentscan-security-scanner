
import { BrowserRouter as Router, Routes, Route } from 'react-router-dom';
import { Layout } from './components/layout/Layout';
import { Dashboard } from './pages/Dashboard';
import { Scans } from './pages/Scans';
import { ScanResults } from './pages/ScanResults';
import { Findings } from './pages/Findings';
import { Reports } from './pages/Reports';
import { Security } from './pages/Security';
import { Activity } from './pages/Activity';
import { Settings } from './pages/Settings';
import { OnboardingFlow, useOnboarding } from './components/onboarding/OnboardingFlow';
import { GlobalShortcutsHelp } from './components/ui/KeyboardShortcutsHelp';
import { useGlobalShortcuts } from './hooks/useKeyboardShortcuts';
import { PageTransition } from './components/ui/Transitions';
import { ErrorBoundary } from './components/ErrorBoundary';
import { AuthProvider, ProtectedRoute, useAuth } from './contexts/AuthContext';
import { LoginForm } from './components/auth/LoginForm';
import './styles/globals.css';

function AppContent() {
  const { state } = useAuth();
  const { shortcuts } = useGlobalShortcuts();
  const {
    isOnboardingOpen,
    completeOnboarding,
    closeOnboarding,
  } = useOnboarding();

  // Show login form if not authenticated
  if (!state.isAuthenticated && !state.isLoading) {
    return <LoginForm />;
  }

  // Show loading while checking authentication
  if (state.isLoading) {
    return (
      <div className="loading-container">
        <div className="loading-spinner">Loading...</div>
      </div>
    );
  }

  return (
    <Router>
      <Layout>
        <PageTransition>
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
        </PageTransition>
      </Layout>
      
      {/* Global components */}
      <OnboardingFlow
        isOpen={isOnboardingOpen}
        onClose={closeOnboarding}
        onComplete={completeOnboarding}
      />
      
      <GlobalShortcutsHelp shortcuts={shortcuts} />
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