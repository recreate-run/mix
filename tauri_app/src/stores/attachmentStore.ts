import { create } from 'zustand';
import { convertFileSrc } from '@tauri-apps/api/core';
import { readDir } from '@tauri-apps/plugin-fs';
import { IMAGE_EXTENSIONS, VIDEO_EXTENSIONS, AUDIO_EXTENSIONS, ALL_MEDIA_EXTENSIONS, getFileType, isMediaFile } from '@/utils/fileTypes';

export type Attachment = {
  id: string;
  name: string;
  type: 'image' | 'video' | 'audio' | 'folder' | 'app';
  // File/folder specific
  path?: string;
  preview?: string;
  extension?: string;
  mediaCount?: {
    images: number;
    videos: number;
    audios: number;
  };
  isDirectory?: boolean;
  // App specific
  icon?: string; // base64
  isOpen?: boolean;
  bundleId?: string;
};

interface AttachmentState {
  attachments: Attachment[];
  referenceMap: Map<string, string>;
  addAttachment: (attachment: Attachment) => void;
  removeAttachment: (index: number) => void;
  clearAttachments: () => void;
  addReference: (displayName: string, path: string) => void;
  removeReference: (displayName: string) => void;
  syncWithText: (text: string) => void;
  getMediaFiles: () => Attachment[];
}




const countMediaFilesInFolder = async (folderPath: string): Promise<{ images: number; videos: number; audios: number }> => {
  try {
    const entries = await readDir(folderPath);
    let images = 0, videos = 0, audios = 0;
    
    for (const entry of entries) {
      if (entry.isFile) {
        const extension = entry.name.split('.').pop()?.toLowerCase();
        if (extension) {
          if (IMAGE_EXTENSIONS.includes(extension)) images++;
          else if (VIDEO_EXTENSIONS.includes(extension)) videos++;
          else if (AUDIO_EXTENSIONS.includes(extension)) audios++;
        }
      }
    }
    
    return { images, videos, audios };
  } catch (error) {
    console.warn('Failed to count media files in folder:', folderPath, error);
    return { images: 0, videos: 0, audios: 0 };
  }
};

export const useAttachmentStore = create<AttachmentState>((set, get) => ({
  attachments: [],
  referenceMap: new Map(),

  addAttachment: (attachment: Attachment) => {
    const state = get();
    
    // Skip if attachment already exists
    if (state.attachments.some(existing => existing.id === attachment.id)) {
      return;
    }

    set(state => {
      const newAttachments = [...state.attachments, attachment];
      if (newAttachments.length > 10) {
        console.warn('Maximum 10 attachments allowed');
        return { attachments: newAttachments.slice(0, 10) };
      }
      return { attachments: newAttachments };
    });
  },

  removeAttachment: (index: number) => {
    set(state => ({
      attachments: state.attachments.filter((_, i) => i !== index)
    }));
  },

  clearAttachments: () => {
    set({ attachments: [], referenceMap: new Map() });
  },

  addReference: (displayName: string, path: string) => {
    set(state => {
      const newMap = new Map(state.referenceMap);
      newMap.set(displayName, path);
      return { referenceMap: newMap };
    });
  },

  removeReference: (displayName: string) => {
    set(state => {
      const newMap = new Map(state.referenceMap);
      newMap.delete(displayName);
      return { referenceMap: newMap };
    });
  },

  syncWithText: (text: string) => {
    const state = get();
    const referencedAttachments = getReferencedAttachments(text, state.attachments);
    
    // Deep comparison to prevent unnecessary updates
    const hasChanged = referencedAttachments.length !== state.attachments.length ||
      referencedAttachments.some((attachment, index) => attachment.id !== state.attachments[index]?.id);
    
    if (hasChanged) {
      set({ attachments: referencedAttachments });
    }
  },


  getMediaFiles: () => {
    const state = get();
    return state.attachments.filter(attachment => 
      attachment.type === 'folder' || 
      (attachment.extension && ALL_MEDIA_EXTENSIONS.includes(attachment.extension as any))
    );
  }
}));

// Utility functions
export const createFileAttachment = (filePath: string): Attachment | null => {
  const fileName = filePath.split('/').pop() || filePath;
  const fileType = getFileType(fileName);
  
  if (!fileType) {
    console.warn(`Unsupported file type: ${fileName}`);
    return null;
  }

  return {
    id: `file:${filePath}`,
    name: fileName,
    type: fileType,
    path: filePath,
    preview: convertFileSrc(filePath),
    extension: fileName.split('.').pop()?.toLowerCase()
  };
};

export const createFolderAttachment = async (folderPath: string): Promise<Attachment> => {
  const folderName = folderPath.split('/').pop() || folderPath;
  const mediaCount = await countMediaFilesInFolder(folderPath);

  return {
    id: `folder:${folderPath}`,
    name: folderName,
    type: 'folder',
    path: folderPath,
    mediaCount,
    isDirectory: true
  };
};


export const isImageFile = (filename: string): boolean => {
  const ext = filename.split('.').pop()?.toLowerCase();
  return ext ? IMAGE_EXTENSIONS.includes(ext as any) : false;
};

export const filterAndSortEntries = (entries: any[], basePath = ''): Attachment[] => {
  return entries
    .filter(entry => {
      if (entry.name.startsWith('.')) return false;
      if (entry.isDirectory) return true;
      const extension = entry.name.split('.').pop()?.toLowerCase();
      return extension && ALL_MEDIA_EXTENSIONS.includes(extension as any);
    })
    .map(entry => {
      const path = basePath ? `${basePath}/${entry.name}` : entry.name;
      const extension = entry.isFile ? entry.name.split('.').pop()?.toLowerCase() : undefined;
      const fileType = extension ? getFileType(entry.name) : null;
      
      return {
        id: entry.isDirectory ? `folder:${path}` : `file:${path}`,
        name: entry.name,
        path,
        type: entry.isDirectory ? 'folder' as const : fileType!,
        isDirectory: entry.isDirectory,
        extension,
        preview: !entry.isDirectory && fileType ? convertFileSrc(path) : undefined
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

// Text reference utilities
export const expandFileReferences = (text: string, referenceMap: Map<string, string>): string => {
  let expandedText = text;
  
  for (const [displayName, fullPath] of referenceMap) {
    // Handle app references by just using the app name
    if (fullPath.startsWith('app:')) {
      const appName = fullPath.substring(4); // Remove 'app:' prefix
      expandedText = expandedText.replace(displayName, appName);
    } else {
      // Handle file/folder references as before
      expandedText = expandedText.replace(displayName, fullPath);
    }
  }
  
  // Check for any remaining unresolved references and throw exception
  const unresolvedMatches = expandedText.match(/@[^\s]+/g);
  if (unresolvedMatches) {
    throw new Error(`Unresolved file references: ${unresolvedMatches.join(', ')}`);
  }
  
  return expandedText;
};

export const removeFileReferences = (text: string, referenceMap: Map<string, string>, fullPath: string): string => {
  let updatedText = text;
  
  for (const [displayName, mappedPath] of referenceMap) {
    if (mappedPath === fullPath) {
      updatedText = updatedText.replace(new RegExp(`${displayName.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}\\s*`, 'g'), '');
    }
  }
  
  return updatedText;
};

// Get attachments that are still referenced in text
export const getReferencedAttachments = (text: string, attachments: Attachment[]): Attachment[] => {
  return attachments.filter(attachment => {
    return text.includes(`@${attachment.name}`) || text.includes(`@../${attachment.name}`);
  });
};