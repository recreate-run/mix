import posthog from 'posthog-js';

/**
 * Initialize PostHog analytics
 * Called once at app startup
 */
export function initPostHog() {
  try {
    // Check if analytics are enabled from environment variable
    const analyticsEnabled =
      import.meta.env.VITE_ANALYTICS_ENABLED !== 'false' &&
      import.meta.env.MIX_ANALYTICS_ENABLED !== 'false';

    if (!analyticsEnabled) {
      console.log('Analytics disabled via environment variable');
      return;
    }

    // Generate a unique identifier for this specific app installation
    const clientId = generateClientId();

    // Initialize PostHog with the correct settings
    posthog.init('phc_M2rmsW9YkY5KVfxFZxbhT7TnEpHxKL9kPVML0dMEn4o', {
      api_host: 'https://eu.posthog.com',
      defaults: '2025-05-24',
      autocapture: false,
      capture_pageview: false, // Disabled - too noisy for desktop app
      persistence: 'localStorage',
      bootstrap: {
        distinctID: clientId,
      },
      debug: false, // Set to false in production
    });

    // Identify the user with the client ID and set app properties
    posthog.identify(clientId, {
      app_type: 'tauri_desktop',
      app_platform: 'desktop',
      app_version: '0.1.0',
    });

    // Send initialization event
    posthog.capture('mix_playground_initialized', {
      version: '0.1.0',
      timestamp: new Date().toISOString(),
    });
  } catch (error) {
    console.error('PostHog initialization error:', error);
  }
}

/**
 * Generate a unique client ID or retrieve existing one
 * Checks for USER_NAME environment variable first, then falls back to stored/generated ID
 */
function generateClientId() {
  // Check for USER_NAME environment variable first (matching backend)
  const userName = import.meta.env.POSTHON_USER_NAME;
  if (userName && userName.trim() !== '') {
    return userName.trim();
  }

  // Fall back to stored client ID
  const existingId = localStorage.getItem('client_id');
  if (existingId) return existingId;

  // Generate new anonymous client ID
  const newId = `client_${Math.random().toString(36).substring(2, 15)}`;
  localStorage.setItem('client_id', newId);
  return newId;
}

/**
 * Track slash command usage (frontend-specific UI event)
 */
export function trackSlashCommand(
  commandName: string,
  properties?: Record<string, any>
) {
  try {
    posthog.capture('slash_command_used', {
      command: commandName,
      timestamp: new Date().toISOString(),
      ...properties,
    });
  } catch (error) {
    console.error('Failed to track slash command:', error);
  }
}

/**
 * Track general UI events (frontend-specific)
 */
export function trackUIEvent(
  eventName: string,
  properties?: Record<string, any>
) {
  try {
    posthog.capture(eventName, {
      timestamp: new Date().toISOString(),
      ...properties,
    });
  } catch (error) {
    console.error('Failed to track UI event:', error);
  }
}
