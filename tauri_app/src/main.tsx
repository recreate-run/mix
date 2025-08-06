import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import { initPostHog, safeTrackEvent } from './lib/posthog';

// Record app start time
const appStartTime = performance.now();

// Initialize PostHog
initPostHog();

// Calculate and track app load time when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
  const loadTime = performance.now() - appStartTime;
  safeTrackEvent('app_load_time', {
    load_time_ms: loadTime,
    timestamp: new Date().toISOString()
  });
});

ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
