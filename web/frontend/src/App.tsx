
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
import './styles/globals.css';

function App() {
  const { shortcuts } = useGlobalShortcuts();
  const {
    isOnboardingOpen,
    completeOnboarding,
    closeOnboarding,
  } = useOnboarding();

  return (
    <ErrorBoundary>
      <Router>
        <Layout>
          <PageTransition>
            <Routes>
              <Route path="/" element={<Dashboard />} />
              <Route path="/scans" element={<Scans />} />
              <Route path="/scans/:id" element={<ScanResults />} />
              <Route path="/findings" element={<Findings />} />
              <Route path="/reports" element={<Reports />} />
              <Route path="/security" element={<Security />} />
              <Route path="/activity" element={<Activity />} />
              <Route path="/settings" element={<Settings />} />
            </Routes>
          </PageTransition>
        </Layout>
        
        {/* Global components */}
        <OnboardingFlow
          isOpen={isOnboardingOpen}
          onClose={closeOnboarding}
          onComplete={completeOnboarding}
          steps={[]} // Use default steps
        />
        
        <GlobalShortcutsHelp shortcuts={shortcuts} />
      </Router>
    </ErrorBoundary>
  );
}

export default App;