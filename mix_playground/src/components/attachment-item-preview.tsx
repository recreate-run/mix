import {
  FolderIcon,
  ImageIcon,
  NotebookPen,
  Play,
  VideoIcon,
} from 'lucide-react';
import type { Attachment } from '@/stores';
import { AudioWaveform } from './audio-waveform';

export interface AttachmentItemPreviewProps {
  attachment: Attachment;
  previewUrl?: string;
}

export const ImagePreview = ({ attachment, previewUrl }: AttachmentItemPreviewProps) => {
  if (!previewUrl) {
    return (
      <div className="flex size-14 items-center justify-center rounded-lg">
        <ImageIcon className="h-6 w-6 text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className="relative">
      <img
        alt={attachment.name}
        className="size-14 rounded-lg object-cover"
        onError={(e) => {
          e.currentTarget.style.display = 'none';
          const fallback = e.currentTarget.nextElementSibling as HTMLElement;
          if (fallback) fallback.style.display = 'block';
        }}
        src={previewUrl}
      />
      <ImageIcon
        className="absolute top-0 left-0 size-14 rounded-lg p-2 text-muted-foreground"
        style={{ display: 'none' }}
      />
    </div>
  );
};

export const VideoPreview = ({ attachment, previewUrl }: AttachmentItemPreviewProps) => {
  // For URL-based attachments, use thumbnailUrl if available, otherwise use previewUrl
  const thumbnailSrc = attachment.thumbnailUrl || previewUrl;
  
  if (!thumbnailSrc) {
    return (
      <div className="flex size-14 items-center justify-center rounded-lg border-2 border-dashed border-gray-300">
        <VideoIcon className="h-6 w-6 text-muted-foreground" />
      </div>
    );
  }

  return (
    <div className="relative">
      <img
        alt={`${attachment.name} thumbnail`}
        className="size-14 rounded-lg object-cover"
        onError={(e) => {
          e.currentTarget.style.display = 'none';
          const fallback = e.currentTarget.nextElementSibling as HTMLElement;
          if (fallback) fallback.style.display = 'block';
        }}
        src={thumbnailSrc}
      />
      <Play className="absolute bottom-1 left-1 h-3 w-3 rounded-full bg-black/50 p-0.5 text-white" />
      {/* Platform indicator for URL videos */}
      {attachment.platform && (
        <div className="absolute top-1 left-1 rounded bg-black/60 px-1 py-0.5 text-white text-xs font-medium">
          {attachment.platform === 'youtube' ? 'YT' : 
           attachment.platform === 'vimeo' ? 'VM' : 
           attachment.platform === 'direct' ? 'MP4' : 'VID'}
        </div>
      )}
      <VideoIcon
        className="absolute top-0 left-0 size-14 rounded-lg p-2 text-muted-foreground"
        style={{ display: 'none' }}
      />
    </div>
  );
};

export const AudioPreview = ({ attachment: _ }: AttachmentItemPreviewProps) => {
  return (
    <div className="flex size-14 items-center justify-center rounded-lg ">
      <AudioWaveform className="h-8 w-10" small />
    </div>
  );
};

export const TextPreview = ({ attachment: _ }: AttachmentItemPreviewProps) => {
  return (
    <div className="flex size-14 items-center justify-center rounded-lg ">
      <NotebookPen className="h-6 w-6 text-muted-foreground" />
    </div>
  );
};

export const FolderPreview = ({ attachment }: AttachmentItemPreviewProps) => {
  return (
    <div className="relative flex items-center justify-center rounded-lg">
      <FolderIcon className="size-16 stroke-1 text-muted-foreground" />
      <div className="absolute inset-0 flex items-center justify-center">
        <span className="max-w-12 truncate font-medium text-[10px] text-white">
          {attachment.name}
        </span>
      </div>
    </div>
  );
};


export const DefaultPreview = ({
  attachment: _,
}: AttachmentItemPreviewProps) => {
  return (
    <div className="flex size-14 items-center justify-center rounded-lg ">
      <ImageIcon className="h-6 w-6 text-muted-foreground" />
    </div>
  );
};
