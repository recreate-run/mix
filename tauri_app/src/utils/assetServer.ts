import type { Attachment } from '@/stores/attachmentSlice';

/**
 * Convert absolute file path to HTTP asset server URL
 * Requires working directory - fails fast if not provided
 */
export const convertToAssetServerUrl = (absolutePath: string, workingDirectory: string): string => {
  const workingDirNormalized = workingDirectory.endsWith('/')
    ? workingDirectory.slice(0, -1)
    : workingDirectory;

  if (!absolutePath.startsWith(workingDirNormalized + '/')) {
    throw new Error(`File path "${absolutePath}" is not within working directory "${workingDirectory}"`);
  }

  const relativePath = absolutePath.substring(workingDirNormalized.length + 1);
  return `${import.meta.env.VITE_BACKEND_URL}/${relativePath}`;
};

/**
 * Generate preview URL for media attachments with error handling
 * Supports configurable thumbnail size for images and videos
 */
export const generatePreviewUrl = (
  attachment: Attachment | { path?: string; type: string },
  workingDirectory: string,
  thumbnailSize = 200
): string | undefined => {
  if (!attachment.path) return undefined;

  try {
    const baseUrl = convertToAssetServerUrl(attachment.path, workingDirectory);
    // For videos and images, request thumbnail with specified max dimension (maintains aspect ratio)
    if (attachment.type === 'video' || attachment.type === 'image') {
      return `${baseUrl}?thumb=${thumbnailSize}`;
    }
    return baseUrl;
  } catch (error) {
    console.error('Failed to generate preview URL:', error);
    return undefined;
  }
};