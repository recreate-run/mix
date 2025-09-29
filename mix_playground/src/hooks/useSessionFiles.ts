import { useQuery } from '@tanstack/react-query';
import { mix } from '@/lib/mix-sdk';
import { CACHE_KEYS } from '@/lib/cache-keys';
import type { Attachment } from '@/stores/attachmentSlice';
import { getFileTypeFromExtension } from '@/utils/fileTypes';

interface FileInfo {
  name: string;
  size: number;
  modified: number;
  isDir: boolean;
}

// Transform SDK FileInfo to Attachment format
const transformFileInfoToAttachment = (fileInfo: FileInfo): Attachment => {
  const extension = fileInfo.name.split('.').pop()?.toLowerCase();

  // Determine attachment type - handle folders first, then use centralized detection
  const type: Attachment['type'] = fileInfo.isDir ? 'folder' : getFileTypeFromExtension(fileInfo.name);

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
  });
};

