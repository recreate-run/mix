import type { Attachment } from '@/stores/attachmentSlice';
import { getBackendUrl } from './backendUrl';

/**
 * Convert filename to session-based HTTP asset server URL
 * Files are served from /api/sessions/{sessionId}/files/{filename}
 */
export const convertToAssetServerUrl = (
  filename: string,
  sessionId: string
): string => {
  if (!sessionId) {
    throw new Error('Session ID is required for asset server URL');
  }

  if (!filename) {
    throw new Error('Filename is required for asset server URL');
  }

  // Extract filename from path if full path is provided
  const cleanFilename = filename.includes('/')
    ? filename.split('/').pop()!
    : filename;

  return `${getBackendUrl()}/api/sessions/${sessionId}/files/${encodeURIComponent(cleanFilename)}`;
};

/**
 * Generate preview URL for media attachments with error handling
 * Supports configurable thumbnail size for images and videos
 */
export const generatePreviewUrl = (
  attachment: Attachment | { path?: string; type: string },
  sessionId: string,
  thumbnailSize = 200
): string | undefined => {
  if (!attachment.path || !sessionId) return undefined;

  try {
    const baseUrl = convertToAssetServerUrl(attachment.path, sessionId);
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
