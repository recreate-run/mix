import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import { initPostHog } from './lib/posthog';

// Record app start time
const appStartTime = performance.now();

// Initialize PostHog
initPostHog();


ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
