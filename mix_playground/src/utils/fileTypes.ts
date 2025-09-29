// Types matching the Go backend structures
interface FileTypeInfo {
  extensions: string[];
  mime_types: Record<string, number>;
  size_limit: number;
}

export interface SupportedFileTypes {
  image: FileTypeInfo;
  video: FileTypeInfo;
  audio: FileTypeInfo;
}

type FileType = 'image' | 'video' | 'audio' | 'text';

// Static extension arrays - single source of truth
export const IMAGE_EXTENSIONS = ['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp', 'tiff'] as const;
export const VIDEO_EXTENSIONS = ['mp4', 'webm', 'mov', 'avi', 'mkv', 'wmv', 'flv', 'm4v'] as const;
export const AUDIO_EXTENSIONS = ['mp3', 'wav', 'flac', 'aac', 'm4a', 'ogg'] as const;
const TEXT_EXTENSIONS = ['md', 'txt'] as const;

/**
 * Simple file type detection based on extension only (no backend dependency)
 * Use this when supportedTypes data isn't available yet
 */
export function getFileTypeFromExtension(fileName: string): FileType {
  const extension = fileName.split('.').pop()?.toLowerCase();
  if (!extension) return 'text';

  if (IMAGE_EXTENSIONS.includes(extension as any)) return 'image';
  if (VIDEO_EXTENSIONS.includes(extension as any)) return 'video';
  if (AUDIO_EXTENSIONS.includes(extension as any)) return 'audio';
  if (TEXT_EXTENSIONS.includes(extension as any)) return 'text';
  
  return 'text'; // Default fallback
}

export function getFileType(fileName: string, supportedTypes?: SupportedFileTypes): FileType | null {
  const extension = '.' + fileName.split('.').pop()?.toLowerCase();
  if (!extension || extension === '.') return null;

  // Handle text files (frontend-only logic)
  const textExt = fileName.split('.').pop()?.toLowerCase();
  if (textExt && TEXT_EXTENSIONS.includes(textExt as any)) return 'text';

  // If no supported types provided, use fallback detection
  if (!supportedTypes) {
    return getFileTypeFromExtension(fileName);
  }

  if (supportedTypes.image.extensions.includes(extension)) return 'image';
  if (supportedTypes.video.extensions.includes(extension)) return 'video';  
  if (supportedTypes.audio.extensions.includes(extension)) return 'audio';

  return null;
}


// Helper functions for backward compatibility
export function getImageExtensions(supportedTypes?: SupportedFileTypes): string[] {
  return supportedTypes?.image.extensions.map(ext => ext.slice(1)) || [];
}

export function getVideoExtensions(supportedTypes?: SupportedFileTypes): string[] {
  return supportedTypes?.video.extensions.map(ext => ext.slice(1)) || [];
}

export function getAudioExtensions(supportedTypes?: SupportedFileTypes): string[] {
  return supportedTypes?.audio.extensions.map(ext => ext.slice(1)) || [];
}
