// GSAP Animation Studio API utilities
export interface AnimationSchema {
  title: string;
  description: string;
  file: string;
  parameters: ParameterSchema[];
}

export interface ParameterSchema {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'color';
  description?: string;
  default?: any;
  min?: number;
  max?: number;
  options?: string[];
}

// Accept any configuration format from the endpoint

const DEFAULT_GSAP_SERVER = import.meta.env.VITE_BACKEND_URL;

// Fetch list of available animations
export async function fetchAnimationList(serverUrl: string = DEFAULT_GSAP_SERVER): Promise<string[]> {
  try {
    console.log(`[GSAP API] Fetching animation list from: ${serverUrl}/api/gsap_animations`);
    const response = await fetch(`${serverUrl}/api/gsap_animations`);
    console.log(`[GSAP API] Animation list response status: ${response.status}`);
    if (!response.ok) {
      throw new Error(`Failed to fetch animations: ${response.statusText}`);
    }
    const animations = await response.json();
    console.log(`[GSAP API] Animation list response:`, animations);
    return animations.map((anim: any) => anim.name);
  } catch (error) {
    console.error(`[GSAP API] Error fetching animation list:`, error);
    return [];
  }
}

// Fetch schema for a specific animation
export async function fetchAnimationSchema(
  animationName: string,
  serverUrl: string
): Promise<AnimationSchema | null> {
  try {
    const url = `${serverUrl}/api/gsap_animations/${encodeURIComponent(animationName)}`;
    console.log(`[GSAP API] Fetching animation schema from: ${url}`);
    const response = await fetch(url);
    console.log(`[GSAP API] Animation schema response status: ${response.status}`);
    if (!response.ok) {
      console.error(`[GSAP API] Failed to fetch animation schema: ${response.status} ${response.statusText}`);
      throw new Error(`Failed to fetch animation schema: ${response.statusText}`);
    }
    const schema = await response.json();
    console.log(`[GSAP API] Animation schema response:`, schema);
    return schema;
  } catch (error) {
    console.error(`[GSAP API] Error fetching animation schema for ${animationName}:`, error);
    return null;
  }
}

// Build animation URL with parameters
export function buildAnimationUrl(
  serverUrl: string,
  animationName: string,
  parameters: Record<string, any>
): string {
  const baseUrl = `${serverUrl}/${encodeURIComponent(animationName)}`;

  // Handle null/undefined parameters
  if (!parameters || typeof parameters !== 'object') {
    return baseUrl;
  }

  // Convert parameters to URL search params
  const searchParams = new URLSearchParams();

  // Safely handle null/undefined parameters
  Object.entries(parameters || {}).forEach(([key, value]) => {
    if (value !== undefined && value !== null && value !== '') {
      // Convert boolean to string
      if (typeof value === 'boolean') {
        searchParams.set(key, value.toString());
      } else {
        searchParams.set(key, String(value));
      }
    }
  });

  const queryString = searchParams.toString();
  return queryString ? `${baseUrl}?${queryString}` : baseUrl;
}

// Check if GSAP server is available
export async function checkGsapServerHealth(serverUrl: string = DEFAULT_GSAP_SERVER): Promise<boolean> {
  try {
    console.log(`[GSAP API] Checking server health: ${serverUrl}/api/gsap_animations`);
    const response = await fetch(`${serverUrl}/api/gsap_animations`, {
      method: 'GET',
      // Add timeout
      signal: AbortSignal.timeout(3000)
    });
    console.log(`[GSAP API] Health check response status: ${response.status}`);
    return response.ok;
  } catch (error) {
    console.error(`[GSAP API] Health check failed:`, error);
    return false;
  }
}