import { X } from 'lucide-react';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import type { Attachment } from '@/stores/attachmentSlice';
import { removeFileReferences } from '@/stores/attachmentSlice';
import { useBoundStore } from '@/stores';
import {
  AppPreview,
  AudioPreview,
  DefaultPreview,
  FolderPreview,
  ImagePreview,
  TextPreview,
  VideoPreview,
} from './attachment-item-preview';

interface AttachmentPreviewProps {
  attachments: Attachment[];
  text: string;
  referenceMap: Map<string, string>;
  onTextChange?: (newText: string) => void;
}

export const AttachmentPreview = ({
  attachments,
  text,
  referenceMap,
  onTextChange,
}: AttachmentPreviewProps) => {
  const removeAttachment = useBoundStore((state) => state.removeAttachment);
  const removeReference = useBoundStore((state) => state.removeReference);

  const handleRemoveItem = (index: number) => {
    const attachmentToRemove = attachments[index];
    if (attachmentToRemove) {
      const fullPath =
        attachmentToRemove.type === 'app'
          ? `app:${attachmentToRemove.name}`
          : attachmentToRemove.path!;
      const updatedText = removeFileReferences(
        text,
        referenceMap,
        fullPath
      );
      onTextChange?.(updatedText);

      // Remove the reference from the map
      for (const [displayName, mappedPath] of referenceMap) {
        if (mappedPath === fullPath) {
          removeReference(displayName);
          break;
        }
      }
    }
    removeAttachment(index);
  };
  if (attachments.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-wrap gap-2 rounded-lg p-2">
      {attachments.map((attachment, index) => (
        <div className="group relative flex-shrink-0" key={attachment.id}>
          {attachment.type === 'app' ? (
            // App attachments have different styling - no tooltip, inline layout
            <div className="relative">
              <AppPreview attachment={attachment} />
              <button
                className="absolute top-1 right-1 z-10 flex items-center justify-center rounded-full bg-red-500/80 p-[2px] transition-colors hover:bg-red-600"
                onClick={() => handleRemoveItem(index)}
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
                  ) : attachment.type === 'text' ? (
                    <TextPreview attachment={attachment} />
                  ) : attachment.type === 'folder' ? (
                    <FolderPreview attachment={attachment} />
                  ) : (
                    <DefaultPreview attachment={attachment} />
                  )}
                </div>
              </TooltipTrigger>
              <TooltipContent>
                <p>
                  {attachment.type === 'folder' && attachment.mediaCount
                    ? (() => {
                        const { images, videos, audios } =
                          attachment.mediaCount;
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
                    : attachment.name}
                </p>
              </TooltipContent>
              <button
                className="absolute top-1 right-1 z-10 flex items-center justify-center rounded-full bg-red-500/80 p-[2px] transition-colors hover:bg-red-600"
                onClick={() => handleRemoveItem(index)}
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
