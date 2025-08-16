import { resolve } from '@tauri-apps/api/path';

/**
 * Normalizes a file path for consistent comparison and storage.
 * Converts to absolute path and removes trailing slashes.
 */
export const normalizePath = async (inputPath: string): Promise<string> => {
  try {
    // Convert to absolute path using Tauri's resolve
    const absolutePath = await resolve(inputPath);
    
    // Remove trailing slashes but preserve root slash
    return absolutePath.replace(/\/+$/, '') || '/';
  } catch (error) {
    console.warn('Failed to normalize path:', inputPath, error);
    // Fallback: just remove trailing slashes from input
    return inputPath.replace(/\/+$/, '') || inputPath;
  }
};