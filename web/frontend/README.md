# AgentScan Dashboard

A modern React-based dashboard for the AgentScan security scanning platform.

## Features Implemented

### Design System
- **Typography**: Inter font with system fallbacks
- **Grid System**: 8pt grid system for consistent spacing
- **Color Palette**: Comprehensive color system with semantic colors
- **Components**: Reusable UI components (Button, Card, Table, Modal)
- **Responsive Design**: Mobile-first approach with breakpoints

### Dashboard Overview Page
- **Statistics Cards**: Display scan metrics (total scans, high/medium/low severity)
- **Recent Scans Table**: Shows repository, status, findings, and time
- **Findings Trend Chart**: Line chart showing findings over time using Recharts
- **Responsive Layout**: Works on mobile, tablet, and desktop
- **Interactive Elements**: Hover states and smooth transitions

### Scan Results Page
- **Scan Header**: Repository, branch, commit info with scan status
- **Findings Table**: Severity, rule, file, line, and tools columns
- **Filtering & Sorting**: Filter by severity/status, search functionality
- **Real-time Updates**: WebSocket connection simulation for live updates
- **Export Functionality**: PDF and JSON export buttons
- **Expandable Details**: Click to view code snippets and fix suggestions
- **Finding Management**: Mark as fixed, ignore, or view in editor

### Layout Components
- **Top Navigation**: 64px height with logo and user actions
- **Sidebar**: 240px width with navigation items
- **Responsive Behavior**: Collapsible sidebar on mobile

## Technology Stack

- **React 18** with TypeScript
- **Vite** for build tooling
- **React Router** for navigation
- **Recharts** for data visualization
- **Lucide React** for icons
- **Vitest** for testing
- **CSS Custom Properties** for theming

## Getting Started

```bash
# Install dependencies
npm install

# Start development server
npm run dev

# Run tests
npm run test

# Build for production
npm run build
```

## Project Structure

```
src/
├── components/
│   ├── layout/          # Layout components (TopNavigation, Sidebar, Layout)
│   └── ui/              # Reusable UI components (Button, Card, Table, Modal)
├── pages/               # Page components (Dashboard, ScanResults, etc.)
├── styles/              # Global styles and design system
└── test/                # Test utilities and setup
```

## Design Philosophy

The dashboard follows a "quiet luxury" design philosophy inspired by Linear, Vercel, and Superhuman:

- **Clean Lines**: Minimal visual noise with purposeful interactions
- **Consistent Spacing**: 8pt grid system for visual harmony
- **Subtle Interactions**: Smooth hover states and transitions
- **Accessibility**: Focus management and keyboard navigation
- **Performance**: Optimized components and lazy loading

## Testing

The project includes comprehensive tests for:
- UI components (Button, Card, Table, Modal)
- Page components (Dashboard, ScanResults)
- User interactions and state management
- Responsive behavior

## Future Enhancements

- WebSocket integration for real-time updates
- Advanced filtering and search capabilities
- Data export functionality
- User preferences and settings
- Dark mode support
- Progressive Web App features