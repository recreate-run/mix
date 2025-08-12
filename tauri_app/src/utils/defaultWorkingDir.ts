import * as path from '@tauri-apps/api/path';

/**
 * Get the default working directory for the current platform.
 * Creates a platform-appropriate directory for new projects.
 * 
 * @returns Promise<string> The default working directory path
 */
export const getDefaultWorkingDir = async (): Promise<string> => {
  try {
    // Get the user's home directory
    const homeDir = await path.homeDir();
    
    // Create a sensible default directory name
    const defaultDir = await path.join(homeDir, 'CreativeAgentProjects');
    
    return defaultDir;
  } catch (error) {
    console.error('Failed to get default working directory:', error);
    // Fallback to a basic directory structure
    return '~/CreativeAgentProjects';
  }
};