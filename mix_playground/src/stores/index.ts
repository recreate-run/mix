import { create } from 'zustand';
import {
  type Attachment,
  type AttachmentSlice,
  createAttachmentSlice,
} from './attachmentSlice';

type BoundStore = AttachmentSlice;

export const useBoundStore = create<BoundStore>((set, get) => ({
  ...createAttachmentSlice(set, get),
}));

export type { Attachment };
