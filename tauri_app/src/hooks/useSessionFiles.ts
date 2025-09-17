import { useQuery } from '@tanstack/react-query';
import { mix } from '@/lib/mix-sdk';
import { CACHE_KEYS } from '@/lib/cache-keys';
import type { Attachment } from '@/stores/attachmentSlice';

interface FileInfo {
  name: string;
  size: number;
  modified: number;
  isDir: boolean;
}

// Transform SDK FileInfo to Attachment format
const transformFileInfoToAttachment = (fileInfo: FileInfo): Attachment => {
  const extension = fileInfo.name.split('.').pop()?.toLowerCase();
  
  // Determine attachment type based on file extension
  let type: Attachment['type'] = 'text';
  
  if (fileInfo.isDir) {
    type = 'folder';
  } else if (['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp', 'tiff'].includes(extension || '')) {
    type = 'image';
  } else if (['mp4', 'webm', 'mov', 'avi', 'mkv', 'wmv', 'flv', 'm4v'].includes(extension || '')) {
    type = 'video';
  } else if (['mp3', 'wav', 'flac', 'aac', 'm4a', 'ogg'].includes(extension || '')) {
    type = 'audio';
  }

  return {
    id: `session-file:${fileInfo.name}`,
    name: fileInfo.name,
    type,
    path: fileInfo.name, // Session files use name as path
    extension,
    isDirectory: fileInfo.isDir,
  };
};

const fetchSessionFiles = async (sessionId: string): Promise<Attachment[]> => {
  const response = await mix.files.list({ id: sessionId });
  return response.map(transformFileInfoToAttachment);
};

export const useSessionFiles = (sessionId?: string) => {
  return useQuery({
    queryKey: CACHE_KEYS.sessionFiles(sessionId!),
    queryFn: () => fetchSessionFiles(sessionId!),
    enabled: !!sessionId,
    staleTime: 30000, // 30 seconds
    gcTime: 5 * 60 * 1000, // 5 minutes
  });
};

export default useSessionFiles;