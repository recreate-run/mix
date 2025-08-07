import { type Attachment } from '@/stores/attachmentStore';
import { 
  ImagePreview, 
  VideoPreview, 
  AudioPreview, 
  FolderPreview, 
  AppPreview, 
  DefaultPreview 
} from './attachment-item-preview';

interface MessageAttachmentDisplayProps {
  attachments: Attachment[];
}

export function MessageAttachmentDisplay({ attachments }: MessageAttachmentDisplayProps) {
  if (!attachments || attachments.length === 0) return null;

  return (
    <div className="flex flex-wrap gap-2 mb-2">
      {attachments.map((attachment, index) => {
        const renderPreview = () => {
          switch (attachment.type) {
            case 'image':
              return <ImagePreview attachment={attachment} />;
            case 'video':
              return <VideoPreview attachment={attachment} />;
            case 'audio':
              return <AudioPreview attachment={attachment} />;
            case 'folder':
              return <FolderPreview attachment={attachment} />;
            case 'app':
              return <AppPreview attachment={attachment} />;
            default:
              return <DefaultPreview attachment={attachment} />;
          }
        };

        return (
          <div key={`${attachment.id}-${index}`} className="flex-shrink-0">
            {renderPreview()}
          </div>
        );
      })}
    </div>
  );
}