
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
import './styles/globals.css';

function App() {
  return (
    <Router>
      <Layout>
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
      </Layout>
    </Router>
  );
}

export default App;