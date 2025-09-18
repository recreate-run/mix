/**
 * Utility to get the backend URL and GSAP URL
 */

export const getBackendUrl = (): string => {
  return import.meta.env.VITE_BACKEND_URL;
};

export const getGsapUrl = (): string => {
  return import.meta.env.VITE_GSAP_URL;
};