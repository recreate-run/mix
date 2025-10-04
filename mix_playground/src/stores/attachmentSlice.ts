import {
  detectVideoUrls,
  createVideoUrlAttachment,
} from '@/utils/videoUrlDetection';

export type Attachment = {
  id: string;
  name: string;
  type: 'image' | 'video' | 'audio' | 'text' | 'folder';
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
  // URL specific (for video/image URLs)
  url?: string;
  thumbnailUrl?: string;
  platform?: 'youtube' | 'vimeo' | 'direct' | 'unknown';
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
  addUrlAttachments: (text: string) => void;
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

  addUrlAttachments: (text: string) => {
    const videoUrls = detectVideoUrls(text);
    const state = get();

    let newAttachments = [...state.attachments];
    const newReferenceMap = new Map(state.referenceMap);
    let hasChanges = false;

    for (const videoInfo of videoUrls) {
      const attachment = createVideoUrlAttachment(videoInfo);

      // Skip if URL attachment already exists
      if (
        state.attachments.some((existing) => existing.url === videoInfo.url)
      ) {
        continue;
      }

      // Add the attachment
      newAttachments.push(attachment);

      // Add reference mapping (URL to itself for direct reference)
      newReferenceMap.set(videoInfo.url, videoInfo.url);
      hasChanges = true;

      // Enforce 10 attachment limit
      if (newAttachments.length > 10) {
        console.warn('Maximum 10 attachments allowed');
        newAttachments = newAttachments.slice(0, 10);
        break;
      }
    }

    if (hasChanges) {
      set(() => ({
        attachments: newAttachments,
        referenceMap: newReferenceMap,
      }));
    }
  },
});

// Get attachments that are still referenced in text
const getReferencedAttachments = (
  text: string,
  attachments: Attachment[]
): Attachment[] => {
  return attachments.filter((attachment) => {
    // Handle file/folder references
    const hasFileReference =
      text.includes(`@${attachment.name}`) ||
      text.includes(`@../${attachment.name}`);

    // Handle URL references (URLs are referenced directly, not with @ prefix)
    const hasUrlReference = attachment.url && text.includes(attachment.url);

    return hasFileReference || hasUrlReference;
  });
};

// This function is now handled in attachmentUtils.ts with simplified signature
