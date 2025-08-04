import posthog from 'posthog-js';

/**
 * Initialize PostHog analytics
 * Called once at app startup
 */
export function initPostHog() {
  try {
    // Generate a unique identifier for this specific app installation
    const clientId = generateClientId();
    
    // Initialize PostHog with the correct settings
    posthog.init(
      'phc_M2rmsW9YkY5KVfxFZxbhT7TnEpHxKL9kPVML0dMEn4o',
      {
        api_host: 'https://eu.i.posthog.com',
        defaults: '2025-05-24',
        autocapture: false,
        capture_pageview: true,
        persistence: 'localStorage',
        bootstrap: { 
          distinctID: clientId
        },
        // Properties to identify the Tauri app
        properties: {
          app_type: 'tauri_desktop',
          app_platform: 'desktop',
          app_version: '0.1.0'
        },
        debug: false // Set to false in production
      }
    );
    
    // Identify the user with the client ID
    posthog.identify(clientId);
    
    // Send initialization event
    posthog.capture("tauri_app_initialized", {
      version: "0.1.0",
      timestamp: new Date().toISOString()
    });
  } catch (error) {
    console.error("PostHog initialization error:", error);
  }
}

/**
 * Generate a unique client ID or retrieve existing one
 */
function generateClientId() {
  const existingId = localStorage.getItem('client_id');
  if (existingId) return existingId;
  
  const newId = `client_${Math.random().toString(36).substring(2, 15)}`;
  localStorage.setItem('client_id', newId);
  return newId;
}

/**
 * Track events with standardized properties
 */
export function trackEvent(eventName: string, properties: Record<string, any> = {}) {
  posthog.capture(eventName, {
    ...properties,
    app_version: '0.1.0',
    timestamp: new Date().toISOString()
  });
}

// Alias for tracking events
export const safeTrackEvent = trackEvent;