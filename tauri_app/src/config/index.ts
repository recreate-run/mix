// Client-side configuration for Tauri app
export const appConfig = {
  services: {
    // GSAP Animation Studio URL - default to localhost:3000
    // This can be overridden by build-time environment variables
    gsapAnimationStudio: import.meta.env?.VITE_GSAP_SERVER_URL,
  },
} as const;

// Export individual service URLs for convenience
export const { gsapAnimationStudio } = appConfig.services;