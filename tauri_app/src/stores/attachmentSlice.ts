import { stat } from '@tauri-apps/plugin-fs';
import {
  createFileAttachment as createFileAttachmentUtil,
  createFolderAttachment as createFolderAttachmentUtil,
} from '@/utils/attachmentUtils';

export type Attachment = {
  id: string;
  name: string;
  type: 'image' | 'video' | 'audio' | 'text' | 'folder' | 'app';
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

export interface AttachmentSlice {
  attachments: Attachment[];
  referenceMap: Map<string, string>;
  addAttachment: (attachment: Attachment) => void;
  removeAttachment: (index: number) => void;
  clearAttachments: () => void;
  addReference: (displayName: string, path: string) => void;
  removeReference: (displayName: string) => void;
  syncWithText: (text: string) => void;
  setHistoryState: (
    attachments: Attachment[],
    referenceMap: Map<string, string>
  ) => void;
  getMediaFiles: () => Attachment[];
}


export const createAttachmentSlice = (
  set: (
    partial: (
      state: AttachmentSlice
    ) => Partial<AttachmentSlice> | AttachmentSlice
  ) => void,
  get: () => AttachmentSlice
): AttachmentSlice => ({
  attachments: [],
  referenceMap: new Map(),

  addAttachment: (attachment: Attachment) => {
    const state = get();

    // Skip if attachment already exists
    if (
      state.attachments.some(
        (existing: Attachment) => existing.id === attachment.id
      )
    ) {
      return;
    }

    set((state) => {
      const newAttachments = [...state.attachments, attachment];
      if (newAttachments.length > 10) {
        console.warn('Maximum 10 attachments allowed');
        return { attachments: newAttachments.slice(0, 10) };
      }
      return { attachments: newAttachments };
    });
  },

  removeAttachment: (index: number) => {
    set((state) => ({
      attachments: state.attachments.filter((_, i) => i !== index),
    }));
  },

  clearAttachments: () => {
    set(() => ({ attachments: [], referenceMap: new Map() }));
  },

  addReference: (displayName: string, path: string) => {
    set((state) => {
      const newMap = new Map(state.referenceMap);
      newMap.set(displayName, path);
      return { referenceMap: newMap };
    });
  },

  removeReference: (displayName: string) => {
    set((state) => {
      const newMap = new Map(state.referenceMap);
      newMap.delete(displayName);
      return { referenceMap: newMap };
    });
  },

  syncWithText: (text: string) => {
    const state = get();
    const referencedAttachments = getReferencedAttachments(
      text,
      state.attachments
    );

    // Deep comparison to prevent unnecessary updates
    const hasChanged =
      referencedAttachments.length !== state.attachments.length ||
      referencedAttachments.some(
        (attachment, index) => attachment.id !== state.attachments[index]?.id
      );

    if (hasChanged) {
      set(() => ({ attachments: referencedAttachments }));
    }
  },

  setHistoryState: (
    attachments: Attachment[],
    referenceMap: Map<string, string>
  ) => {
    // Apply 10-attachment limit
    const limitedAttachments =
      attachments.length > 10 ? attachments.slice(0, 10) : attachments;
    if (attachments.length > 10) {
      console.warn('Maximum 10 attachments allowed, truncating');
    }

    // Atomic update of both attachments and referenceMap
    set(() => ({
      attachments: limitedAttachments,
      referenceMap: new Map(referenceMap),
    }));
  },

  getMediaFiles: () => {
    const state = get();
    return state.attachments.filter(
      (attachment) =>
        attachment.type === 'folder' ||
        attachment.type === 'image' ||
        attachment.type === 'video' ||
        attachment.type === 'audio'
    );
  },
});

export const getParentPath = (path: string): string | null => {
  const parts = path.split('/');
  parts.pop();
  return parts.length > 0 ? parts.join('/') : null;
};


// Get attachments that are still referenced in text
export const getReferencedAttachments = (
  text: string,
  attachments: Attachment[]
): Attachment[] => {
  return attachments.filter((attachment) => {
    return (
      text.includes(`@${attachment.name}`) ||
      text.includes(`@../${attachment.name}`)
    );
  });
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
          attachment = await createFolderAttachmentUtil(mediaPath);
        } else {
          attachment = createFileAttachmentUtil(mediaPath);
        }
      } catch (statError) {
        // If stat fails, try to create as file based on file extension
        attachment = createFileAttachmentUtil(mediaPath);
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
