import { X } from 'lucide-react';
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip';
import { type Attachment } from '@/stores/attachmentStore';
import { 
  ImagePreview, 
  VideoPreview, 
  AudioPreview, 
  FolderPreview, 
  AppPreview, 
  DefaultPreview 
} from './attachment-item-preview';

interface AttachmentPreviewProps {
  attachments: Attachment[];
  onRemoveItem: (index: number) => void;
}


export const AttachmentPreview = ({ attachments, onRemoveItem }: AttachmentPreviewProps) => {
  if (attachments.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-wrap gap-2 p-2 bg-gray-50 dark:bg-gray-800/30 rounded-lg">
      {attachments.map((attachment, index) => (
        <div key={attachment.id} className="relative group flex-shrink-0">
          {attachment.type === 'app' ? (
            // App attachments have different styling - no tooltip, inline layout
            <div className="relative">
              <AppPreview attachment={attachment} />
              <button
                onClick={() => onRemoveItem(index)}
                className="absolute top-1 right-1 p-[2px] bg-red-500/80 hover:bg-red-600 rounded-full flex items-center justify-center transition-colors z-10"
                title="Remove app"
              >
                <X className="size-3 text-white" />
              </button>
            </div>
          ) : (
            // File/folder/media attachments use tooltip and grid layout
            <Tooltip>
              <TooltipTrigger asChild>
                <div className="relative">
                  {attachment.type === 'image' ? (
                    <ImagePreview attachment={attachment} />
                  ) : attachment.type === 'video' ? (
                    <VideoPreview attachment={attachment} />
                  ) : attachment.type === 'audio' ? (
                    <AudioPreview attachment={attachment} />
                  ) : attachment.type === 'folder' ? (
                    <FolderPreview attachment={attachment} />
                  ) : (
                    <DefaultPreview attachment={attachment} />
                  )}
                </div>
              </TooltipTrigger>
              <TooltipContent>
                <p>
                  {attachment.type === 'folder' && attachment.mediaCount ? (
                    (() => {
                      const { images, videos, audios } = attachment.mediaCount;
                      const total = images + videos + audios;
                      if (total === 0) {
                        return `${attachment.name} - no media files`;
                      }
                      const parts = [];
                      if (images > 0) parts.push(`${images}i`);
                      if (videos > 0) parts.push(`${videos}v`);
                      if (audios > 0) parts.push(`${audios}a`);
                      return `${attachment.name} ${parts.join('/')}`;
                    })()
                  ) : (
                    attachment.name
                  )}
                </p>
              </TooltipContent>
              <button
                onClick={() => onRemoveItem(index)}
                className="absolute top-1 right-1 p-[2px] bg-red-500/80 hover:bg-red-600 rounded-full flex items-center justify-center transition-colors z-10"
              >
                <X className="size-3 text-white" />
              </button>
            </Tooltip>
          )}
        </div>
      ))}
    </div>
  );
};