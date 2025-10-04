/**
 * Utility to get the backend URL and GSAP URL
 */

export const getBackendUrl = (): string => {
  const backendUrl = import.meta.env.VITE_BACKEND_URL;
  if (!backendUrl) {
    throw new Error(
      'VITE_BACKEND_URL environment variable is not set. Please configure it in your .env file.'
    );
  }
  return backendUrl;
};

export const getGsapUrl = (): string => {
  const gsapUrl = import.meta.env.VITE_GSAP_URL;
  if (!gsapUrl) {
    throw new Error(
      'VITE_GSAP_URL environment variable is not set. Please configure it in your .env file.'
    );
  }
  return gsapUrl;
};
