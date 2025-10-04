import { readDir } from '@tauri-apps/plugin-fs';
import type { Attachment } from '@/stores/attachmentSlice';
import {
  getFileType,
  getImageExtensions,
  getVideoExtensions,
  getAudioExtensions,
  type SupportedFileTypes,
} from '@/utils/fileTypes';

// Helper function for folder attachment creation
const countMediaFilesInFolder = async (
  folderPath: string,
  supportedTypes?: SupportedFileTypes
): Promise<{ images: number; videos: number; audios: number }> => {
  try {
    const entries = await readDir(folderPath);
    let images = 0,
      videos = 0,
      audios = 0;

    // Return zeros if file types not loaded yet
    if (!supportedTypes) {
      return { images: 0, videos: 0, audios: 0 };
    }

    const imageExts = getImageExtensions(supportedTypes);
    const videoExts = getVideoExtensions(supportedTypes);
    const audioExts = getAudioExtensions(supportedTypes);

    for (const entry of entries) {
      if (entry.isFile) {
        const extension = entry.name.split('.').pop()?.toLowerCase();
        if (extension) {
          if (imageExts.includes(extension)) images++;
          else if (videoExts.includes(extension)) videos++;
          else if (audioExts.includes(extension)) audios++;
        }
      }
    }

    return { images, videos, audios };
  } catch (error) {
    console.warn('Failed to count media files in folder:', folderPath, error);
    return { images: 0, videos: 0, audios: 0 };
  }
};

// Attachment creation utilities
export const createFileAttachment = (
  filePath: string,
  supportedTypes?: SupportedFileTypes
): Attachment | null => {
  const fileName = filePath.split('/').pop() || filePath;
  const fileType = getFileType(fileName, supportedTypes);

  if (!fileType) {
    console.warn(`Unsupported file type: ${fileName}`);
    return null;
  }

  return {
    id: `file:${filePath}`,
    name: fileName,
    type: fileType,
    path: filePath,
    // Note: Preview URL will be generated when sessionStorageDirectory is available
    extension: fileName.split('.').pop()?.toLowerCase(),
  };
};

export const createFolderAttachment = async (
  folderPath: string,
  supportedTypes?: SupportedFileTypes
): Promise<Attachment> => {
  const folderName = folderPath.split('/').pop() || folderPath;
  const mediaCount = await countMediaFilesInFolder(folderPath, supportedTypes);

  return {
    id: `folder:${folderPath}`,
    name: folderName,
    type: 'folder',
    path: folderPath,
    mediaCount,
    isDirectory: true,
  };
};

// URL building utilities
export const buildSessionFileUrl = (
  sessionId: string,
  fileName: string
): string => {
  const backendUrl = import.meta.env.VITE_BACKEND_URL;
  return `${backendUrl}/api/sessions/${sessionId}/files/${fileName}`;
};

// Text reference utilities
export const expandFileReferences = (
  text: string,
  referenceMap: Map<string, string>
): string => {
  let expandedText = text;

  for (const [displayName, fullUrl] of referenceMap) {
    // Handle file/folder references
    expandedText = expandedText.replace(displayName, fullUrl);
  }

  // Check for any remaining unresolved references and throw exception
  const unresolvedMatches = expandedText.match(/@[^\s]+/g);
  if (unresolvedMatches) {
    throw new Error(
      `Unresolved file references: ${unresolvedMatches.join(', ')}`
    );
  }

  return expandedText;
};

export const removeFileReferences = (
  text: string,
  referenceMap: Map<string, string>,
  fullPath: string
): string => {
  let updatedText = text;

  for (const [displayName, mappedPath] of referenceMap) {
    if (mappedPath === fullPath) {
      updatedText = updatedText.replace(
        new RegExp(
          `${displayName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\s*`,
          'g'
        ),
        ''
      );
    }
  }

  return updatedText;
};
