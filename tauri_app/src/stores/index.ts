import { create } from 'zustand';
import { createAttachmentSlice, type AttachmentSlice, type Attachment } from './attachmentSlice';

type BoundStore = AttachmentSlice;

export const useBoundStore = create<BoundStore>((set, get) => ({
  ...createAttachmentSlice(set, get),
}));

export type { Attachment };

