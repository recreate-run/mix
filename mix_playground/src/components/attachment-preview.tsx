import { X } from 'lucide-react';
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';
import { useBoundStore } from '@/stores';
import type { Attachment } from '@/stores/attachmentSlice';
import { removeFileReferences } from '@/utils/attachmentUtils';
import { generatePreviewUrl } from '@/utils/assetServer';
import {
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
  sessionId: string;
  onTextChange?: (newText: string) => void;
}

export const AttachmentPreview = ({
  attachments,
  text,
  referenceMap,
  sessionId,
  onTextChange,
}: AttachmentPreviewProps) => {
  const removeAttachment = useBoundStore((state) => state.removeAttachment);
  const removeReference = useBoundStore((state) => state.removeReference);


  const handleRemoveItem = (index: number) => {
    const attachmentToRemove = attachments[index];
    if (attachmentToRemove) {
      let identifier: string;
      
      if (attachmentToRemove.url) {
        // For URL attachments, remove the URL directly from text
        identifier = attachmentToRemove.url;
        const updatedText = text.replace(new RegExp(attachmentToRemove.url.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'), 'g'), '');
        onTextChange?.(updatedText);
      } else {
        identifier = attachmentToRemove.path!;
      }

      // For non-URL attachments, use the existing file reference removal logic
      if (!attachmentToRemove.url) {
        const updatedText = removeFileReferences(text, referenceMap, identifier);
        onTextChange?.(updatedText);
      }

      // Remove the reference from the map
      for (const [displayName, mappedPath] of referenceMap) {
        if (mappedPath === identifier) {
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
    <div className="flex flex-wrap gap-2 rounded-lg px-2">
      {attachments.map((attachment, index) => (
        <div className="group relative flex-shrink-0" key={attachment.id}>
          {
            // File/folder/media attachments use tooltip and grid layout
            <Tooltip>
              <TooltipTrigger asChild>
                <div className="relative">
                  {attachment.type === 'image' ? (
                    <ImagePreview attachment={attachment} previewUrl={generatePreviewUrl(attachment, sessionId)} />
                  ) : attachment.type === 'video' ? (
                    <VideoPreview attachment={attachment} previewUrl={attachment.url ? undefined : generatePreviewUrl(attachment, sessionId)} />
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
          }
        </div>
      ))}
    </div>
  );
};
