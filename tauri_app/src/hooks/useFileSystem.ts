import { useQuery } from '@tanstack/react-query';
import * as path from '@tauri-apps/api/path';
import { BaseDirectory, readDir } from '@tauri-apps/plugin-fs';
import { useCallback } from 'react';
import {
  type Attachment,
  filterAndSortEntries,
} from '@/stores/attachmentStore';

export type MediaItem = Attachment;

interface FileSystemData {
  files: Attachment[];
  currentDirectory: string;
}

const fetchFileSystemData = async (
  customBasePath?: string
): Promise<FileSystemData> => {
  try {
    if (customBasePath) {
      // Use custom path directly
      const entries = await readDir(customBasePath);
      const fileEntries = filterAndSortEntries(entries, customBasePath);

      return {
        files: fileEntries,
        currentDirectory: customBasePath,
      };
    }
    // Fallback to Desktop
    const homeDir = await path.homeDir();
    const entries = await readDir('.', {
      baseDir: BaseDirectory.Desktop,
    });

    const fileEntries = filterAndSortEntries(entries);

    return {
      files: fileEntries,
      currentDirectory: homeDir,
    };
  } catch (error) {
    console.error('Failed to load directory:', error);
    throw error;
  }
};

const fetchDirectoryContents = async (
  dirPath: string,
  customBasePath?: string
): Promise<Attachment[]> => {
  try {
    if (customBasePath) {
      // Use absolute path when we have a custom base
      const entries = await readDir(dirPath);
      return filterAndSortEntries(entries, dirPath);
    }
    // Fallback to Desktop base directory
    const entries = await readDir(dirPath, {
      baseDir: BaseDirectory.Desktop,
    });
    return filterAndSortEntries(entries, dirPath);
  } catch (error) {
    console.error('Failed to load directory contents:', error);
    throw error;
  }
};

export const useFileSystem = (customBasePath?: string) => {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['fileSystem', customBasePath],
    queryFn: () => fetchFileSystemData(customBasePath),
    enabled: false, // Don't fetch automatically
    staleTime: 0, // Always refetch when manually triggered
    refetchOnWindowFocus: false, // Don't refetch on window focus
  });

  const fetchFiles = useCallback(() => {
    return refetch();
  }, [refetch]);

  const fetchDirectoryContentsWrapper = useCallback(
    async (dirPath: string) => {
      return await fetchDirectoryContents(dirPath, customBasePath);
    },
    [customBasePath]
  );

  return {
    currentFiles: data?.files ?? [],
    isLoading,
    error: error?.message ?? null,
    currentDirectory: data?.currentDirectory ?? '',
    refresh: refetch,
    fetchFiles, // New method to trigger on-demand fetch
    fetchDirectoryContents: fetchDirectoryContentsWrapper, // Expose directory contents function
  };
};
