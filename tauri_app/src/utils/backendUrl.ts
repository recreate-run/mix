/**
 * Utility to get the backend URL with a default fallback
 */

export const getBackendUrl = (): string => {
  return import.meta.env.VITE_BACKEND_URL;
};