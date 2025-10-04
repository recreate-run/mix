// GSAP Animation Studio API utilities
export interface AnimationSchema {
  title: string;
  description: string;
  file: string;
  parameters: ParameterSchema[];
}

interface ParameterSchema {
  name: string;
  type: 'string' | 'number' | 'boolean' | 'color';
  description?: string;
  default?: any;
  min?: number;
  max?: number;
  options?: string[];
}

// Fetch schema for a specific animation
export async function fetchAnimationSchema(
  animationName: string,
  serverUrl: string
): Promise<AnimationSchema | null> {
  try {
    // Use the correct GSAP server endpoint format
    const url = `${serverUrl}/animations/${encodeURIComponent(animationName)}/schema`;

    const response = await fetch(url);

    if (!response.ok) {
      console.error(
        `[GSAP API] Failed to fetch animation schema: ${response.status} ${response.statusText}`
      );
      throw new Error(
        `Failed to fetch animation schema: ${response.statusText}`
      );
    }

    const schema = await response.json();
    return schema;
  } catch (error) {
    console.error(
      `[GSAP API] Error fetching animation schema for ${animationName}:`,
      error
    );
    return null;
  }
}
