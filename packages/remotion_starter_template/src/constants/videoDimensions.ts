/**
 * Predefined video dimensions for different platforms and orientations
 */

export type VideoFormat = 'horizontal' | 'vertical';

export interface VideoDimensions {
  width: number;
  height: number;
}

export const VIDEO_DIMENSIONS: Record<VideoFormat, VideoDimensions> = {
  horizontal: {
    width: 1920,
    height: 1080,
  },
  vertical: {
    width: 1080,
    height: 1920,
  },
};

/**
 * Get dimensions for a specific video format
 */
export function getDimensionsForFormat(format: VideoFormat): VideoDimensions {
  return VIDEO_DIMENSIONS[format];
}