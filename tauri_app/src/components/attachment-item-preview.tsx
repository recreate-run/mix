import { ImageIcon, VideoIcon, Play, FolderIcon } from 'lucide-react';
import { type Attachment } from '@/stores/attachmentStore';
import { AudioWaveform } from './audio-waveform';

export interface AttachmentItemPreviewProps {
  attachment: Attachment;
}

export const ImagePreview = ({ attachment }: AttachmentItemPreviewProps) => {
  return (
    <div className="relative">
      <img 
        src={attachment.preview} 
        alt={attachment.name}
        className="size-14 object-cover rounded-lg border border-stone-600"
        onError={(e) => {
          console.error('❌ [Attachment Debug] Image failed to load:', { 
            name: attachment.name, 
            src: attachment.preview,
            error: e 
          });
          e.currentTarget.style.display = 'none';
          const fallback = e.currentTarget.nextElementSibling as HTMLElement;
          if (fallback) fallback.style.display = 'block';
        }}
      />
      <ImageIcon 
        className="size-14 text-stone-400 absolute top-0 left-0 rounded-lg border border-stone-600 bg-stone-700/50 p-2" 
        style={{ display: 'none' }}
      />
    </div>
  );
};

export const VideoPreview = ({ attachment }: AttachmentItemPreviewProps) => {
  return (
    <div className="relative">
      <video 
        src={attachment.preview}
        className="size-14 object-cover rounded-lg border border-stone-600"
        preload="metadata"
        onLoadedMetadata={(e) => {
          e.currentTarget.currentTime = 1;
        }}
        onError={(e) => {
          console.error('❌ [Attachment Debug] Video failed to load:', { 
            name: attachment.name, 
            src: attachment.preview,
            error: e 
          });
          e.currentTarget.style.display = 'none';
          const fallback = e.currentTarget.nextElementSibling as HTMLElement;
          if (fallback) fallback.style.display = 'block';
        }}
      />
      <Play className="absolute bottom-1 left-1 w-3 h-3 text-white bg-black/50 rounded-full p-0.5" />
      <VideoIcon 
        className="size-14 text-stone-400 absolute top-0 left-0 rounded-lg border border-stone-600 bg-stone-700/50 p-2" 
        style={{ display: 'none' }}
      />
    </div>
  );
};

export const AudioPreview = ({ attachment }: AttachmentItemPreviewProps) => {
  return (
    <div className="size-14 bg-stone-700/50 border border-stone-600 rounded-lg flex items-center justify-center">
      <AudioWaveform className="h-8 w-10" small />
    </div>
  );
};

export const FolderPreview = ({ attachment }: AttachmentItemPreviewProps) => {
  return (
    <div className="rounded-lg flex items-center justify-center relative">
      <FolderIcon className="size-16 stroke-1 text-stone-400" />
      <div className="absolute inset-0 flex items-center justify-center">
        <span className="text-[10px] text-white font-medium truncate max-w-12">
          {attachment.name}
        </span>
      </div>
    </div>
  );
};

export const AppPreview = ({ attachment }: AttachmentItemPreviewProps) => {
  return (
    <div className="flex items-center gap-2 px-3 py-2 bg-white dark:bg-gray-700 rounded-md border border-gray-200 dark:border-gray-600 group hover:border-gray-300 dark:hover:border-gray-500 transition-colors min-w-0">
      <div className="flex-shrink-0 p-1 rounded-md bg-gray-50 dark:bg-gray-600 shadow-sm">
        <img 
          src={`data:image/png;base64,${attachment.icon}`} 
          alt={`${attachment.name} icon`}
          className="size-4 rounded-sm"
        />
      </div>
      
      <div className="flex-1 min-w-0">
        <div className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
          {attachment.name}
        </div>
        <div className="text-xs text-gray-500 dark:text-gray-400">
          Application
        </div>
      </div>
    </div>
  );
};

export const DefaultPreview = ({ attachment }: AttachmentItemPreviewProps) => {
  return (
    <div className="size-14 bg-stone-700/50 border border-stone-600 rounded-lg flex items-center justify-center">
      <ImageIcon className="w-6 h-6 text-stone-400" />
    </div>
  );
};