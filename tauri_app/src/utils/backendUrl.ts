/**
 * Utility to get the backend URL with a default fallback
 */
export const DEFAULT_BACKEND_URL = 'http://localhost:8088';

export const getBackendUrl = (): string => {
  return import.meta.env.VITE_BACKEND_URL || DEFAULT_BACKEND_URL;
};