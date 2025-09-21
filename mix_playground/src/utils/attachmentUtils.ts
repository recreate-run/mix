import { readDir, stat } from '@tauri-apps/plugin-fs';
import type { Attachment } from '@/stores/attachmentSlice';
import {
  getFileType,
  getImageExtensions,
  getVideoExtensions,
  getAudioExtensions,
  type SupportedFileTypes,
} from '@/utils/fileTypes';

// Helper function for folder attachment creation
export const countMediaFilesInFolder = async (
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

const IGNORED_DIRECTORIES = [
  'node_modules',
  '.git',
  '.next',
  '.nuxt',
  'dist',
  'build',
  'target',     // Rust
  '.cargo',
  'tmp',
  'temp',
  '__pycache__',
  '.pytest_cache',
  'coverage',
  '.nyc_output',
  'vendor',     // PHP/Go
  '.idea',      // JetBrains IDEs
  '.vscode',    // VS Code
  '.gradle',    // Gradle
  'logs'
];

// File system utilities
export const filterAndSortEntries = (
  entries: any[],
  basePath = '',
  supportedTypes?: SupportedFileTypes
): Attachment[] => {
  return entries
    .filter((entry) => {
      if (entry.name.startsWith('.')) return false;
      if (entry.isDirectory && IGNORED_DIRECTORIES.includes(entry.name)) return false;
      if (entry.isDirectory) return true;
      // If file types not loaded yet, don't filter files
      if (!supportedTypes) return true;
      const fileType = getFileType(entry.name, supportedTypes);
      return fileType !== null;
    })
    .map((entry) => {
      const path = basePath ? `${basePath}/${entry.name}` : entry.name;
      const extension = entry.isFile
        ? entry.name.split('.').pop()?.toLowerCase()
        : undefined;
      const fileType = extension ? getFileType(entry.name, supportedTypes) : null;

      return {
        id: entry.isDirectory ? `folder:${path}` : `file:${path}`,
        name: entry.name,
        path,
        type: entry.isDirectory ? ('folder' as const) : fileType!,
        isDirectory: entry.isDirectory,
        extension,
        // Note: Preview URL will be generated when sessionStorageDirectory is available
      };
    })
    .sort((a, b) => {
      if (a.isDirectory && !b.isDirectory) return -1;
      if (!a.isDirectory && b.isDirectory) return 1;
      return a.name.localeCompare(b.name);
    });
};

export const getParentPath = (path: string): string | null => {
  const parts = path.split('/');
  parts.pop();
  return parts.length > 0 ? parts.join('/') : null;
};

// URL building utilities
export const buildSessionFileUrl = (sessionId: string, fileName: string): string => {
  const backendUrl = import.meta.env.VITE_BACKEND_URL;
  return `${backendUrl}/api/sessions/${sessionId}/files/${fileName}`;
};

export const buildFullUrlFromPath = (path: string): string => {
  const backendUrl = import.meta.env.VITE_BACKEND_URL;
  return `${backendUrl}${path}`;
};

// Text reference utilities
export const expandFileReferences = (
  text: string,
  referenceMap: Map<string, string>,
  mediaUrls?: string[]
): string => {
  let expandedText = text;

  // Create a set of media URLs for quick lookup
  const mediaUrlSet = mediaUrls ? new Set(mediaUrls) : null;

  for (const [displayName, fullUrl] of referenceMap) {
    // Handle app references by just using the app name
    if (fullUrl.startsWith('app:')) {
      const appName = fullUrl.substring(4); // Remove 'app:' prefix
      expandedText = expandedText.replace(displayName, appName);
    } else {
      // Handle file/folder references
      // If mediaUrls is provided, ensure we use the exact URL from media array
      let urlToUse = fullUrl;
      if (mediaUrlSet) {
        // Find the matching URL in media array (should be exact match)
        const matchingMediaUrl = mediaUrls?.find(mediaUrl => mediaUrl === fullUrl);
        if (matchingMediaUrl) {
          urlToUse = matchingMediaUrl;
        }
      }
      expandedText = expandedText.replace(displayName, urlToUse);
    }
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

// Reconstruct attachment state from historical message data
export const reconstructAttachmentsFromHistory = async (
  text: string,
  mediaPaths: string[],
  appNames: string[]
): Promise<{
  contractedText: string;
  attachments: Attachment[];
  referenceMap: Map<string, string>;
}> => {
  const attachments: Attachment[] = [];
  const referenceMap = new Map<string, string>();
  let contractedText = text;

  // Process media files/folders
  for (const mediaPath of mediaPaths) {
    try {
      // Use stat to determine if path is file or directory
      let attachment: Attachment | null = null;

      try {
        // Use stat to properly determine if path is file or directory
        const pathStat = await stat(mediaPath);

        if (pathStat.isDirectory) {
          attachment = await createFolderAttachment(mediaPath);
        } else {
          attachment = createFileAttachment(mediaPath);
        }
      } catch (statError) {
        // If stat fails, try to create as file based on file extension
        attachment = createFileAttachment(mediaPath);
      }

      if (attachment) {
        const displayName = `@${attachment.name}`;
        attachments.push(attachment);
        referenceMap.set(displayName, mediaPath);

        // Replace full path with display name in text
        contractedText = contractedText.replace(
          new RegExp(mediaPath.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g'),
          displayName
        );
      }
    } catch (error) {
      console.warn(
        'Failed to create attachment for media path:',
        mediaPath,
        error
      );
    }
  }

  // Process app references
  for (const appName of appNames) {
    const attachment: Attachment = {
      id: `app:${appName}`,
      name: appName,
      type: 'app',
      icon: 'placeholder',
      isOpen: true,
    };

    const displayName = `@${appName}`;
    attachments.push(attachment);
    referenceMap.set(displayName, `app:${appName}`);

    // Replace app name with display name in text (only if it's not already in @ format)
    if (!contractedText.includes(displayName)) {
      contractedText = contractedText.replace(
        new RegExp(
          `\\b${appName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\b`,
          'g'
        ),
        displayName
      );
    }
  }

  return { contractedText, attachments, referenceMap };
};
