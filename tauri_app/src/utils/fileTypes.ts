export const IMAGE_EXTENSIONS = [
  'jpg',
  'jpeg',
  'png',
  'gif',
  'webp',
  'bmp',
  'svg',
] as const;
export const VIDEO_EXTENSIONS = [
  'mp4',
  'mov',
  'avi',
  'mkv',
  'webm',
  'wmv',
] as const;
export const AUDIO_EXTENSIONS = [
  'mp3',
  'wav',
  'ogg',
  'm4a',
  'aac',
  'flac',
  'wma',
] as const;
export const TEXT_EXTENSIONS = ['md', 'txt'] as const;

export const ALL_MEDIA_EXTENSIONS = [
  ...IMAGE_EXTENSIONS,
  ...VIDEO_EXTENSIONS,
  ...AUDIO_EXTENSIONS,
  ...TEXT_EXTENSIONS,
] as const;

export type FileType = 'image' | 'video' | 'audio' | 'text';

export function getFileType(fileName: string): FileType | null {
  const extension = fileName.split('.').pop()?.toLowerCase();
  if (!extension) return null;

  if (IMAGE_EXTENSIONS.includes(extension as any)) return 'image';
  if (VIDEO_EXTENSIONS.includes(extension as any)) return 'video';
  if (AUDIO_EXTENSIONS.includes(extension as any)) return 'audio';
  if (TEXT_EXTENSIONS.includes(extension as any)) return 'text';

  return null;
}

export function isMediaFile(fileName: string): boolean {
  return getFileType(fileName) !== null;
}
