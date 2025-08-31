/**
 * Utility to get the backend URL with a default fallback
 * 
 * This ensures that even if the environment variable is not set,
 * the application will use a sensible default value.
 */

// Default backend URL if environment variable is not set
export const DEFAULT_BACKEND_URL = 'http://localhost:8088';

/**
 * Returns the backend URL from the environment variable with a fallback
 * @returns The backend URL string
 */
export const getBackendUrl = (): string => {
  return import.meta.env.VITE_BACKEND_URL || DEFAULT_BACKEND_URL;
};