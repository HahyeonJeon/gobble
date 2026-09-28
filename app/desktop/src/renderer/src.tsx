import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { WorkspaceApp } from './workspace/WorkspaceApp';
import { ErrorBoundary } from './workspace/ErrorBoundary';
import './style.css';

const root = document.getElementById('root');
if (!root) throw new Error('App root is missing.');
createRoot(root).render(
  <StrictMode>
    <ErrorBoundary>
      <WorkspaceApp />
    </ErrorBoundary>
  </StrictMode>,
);
