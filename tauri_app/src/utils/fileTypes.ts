// Types matching the Go backend structures
export interface FileTypeInfo {
  extensions: string[];
  mime_types: Record<string, number>;
  size_limit: number;
}

export interface SupportedFileTypes {
  image: FileTypeInfo;
  video: FileTypeInfo;
  audio: FileTypeInfo;
}

export type FileType = 'image' | 'video' | 'audio' | 'text';

// Text extensions are still frontend-only for now
const TEXT_EXTENSIONS = ['md', 'txt'] as const;

export function getFileType(fileName: string, supportedTypes?: SupportedFileTypes): FileType | null {
  const extension = '.' + fileName.split('.').pop()?.toLowerCase();
  if (!extension || extension === '.') return null;

  // Handle text files (frontend-only logic)
  const textExt = fileName.split('.').pop()?.toLowerCase();
  if (textExt && TEXT_EXTENSIONS.includes(textExt as any)) return 'text';

  // Return null if no supported types provided (loading state)
  if (!supportedTypes) return null;

  if (supportedTypes.image.extensions.includes(extension)) return 'image';
  if (supportedTypes.video.extensions.includes(extension)) return 'video';  
  if (supportedTypes.audio.extensions.includes(extension)) return 'audio';

  return null;
}

export function isMediaFile(fileName: string, supportedTypes?: SupportedFileTypes): boolean {
  return getFileType(fileName, supportedTypes) !== null;
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
