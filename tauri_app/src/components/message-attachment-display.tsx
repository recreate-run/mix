import type { Attachment } from '@/stores';
import {
  AppPreview,
  AudioPreview,
  DefaultPreview,
  FolderPreview,
  ImagePreview,
  TextPreview,
  VideoPreview,
} from './attachment-item-preview';

interface MessageAttachmentDisplayProps {
  attachments: Attachment[];
}

export function MessageAttachmentDisplay({
  attachments,
}: MessageAttachmentDisplayProps) {
  if (!attachments || attachments.length === 0) return null;

  return (
    <div className="mb-2 flex flex-wrap gap-2">
      {attachments.map((attachment, index) => {
        const renderPreview = () => {
          switch (attachment.type) {
            case 'image':
              return <ImagePreview attachment={attachment} />;
            case 'video':
              return <VideoPreview attachment={attachment} />;
            case 'audio':
              return <AudioPreview attachment={attachment} />;
            case 'text':
              return <TextPreview attachment={attachment} />;
            case 'folder':
              return <FolderPreview attachment={attachment} />;
            case 'app':
              return <AppPreview attachment={attachment} />;
            default:
              return <DefaultPreview attachment={attachment} />;
          }
        };

        return (
          <div className="flex-shrink-0" key={`${attachment.id}-${index}`}>
            {renderPreview()}
          </div>
        );
      })}
    </div>
  );
}
