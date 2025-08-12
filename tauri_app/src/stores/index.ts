import { create } from 'zustand';
import { createAttachmentSlice, type AttachmentSlice } from './attachmentSlice';

type BoundStore = AttachmentSlice;

export const useBoundStore = create<BoundStore>((...a) => ({
  ...createAttachmentSlice(...a),
}));

