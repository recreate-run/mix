import { isYouTubeUrl } from '@/utils/videoUrlDetection';
import type { MediaOutput } from '@/types/media';
import { Download } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useMediaDownload } from '@/hooks/use-media-download';

export const MediaDownloadButton = ({
  media,
  sessionId,
  getMediaSrc,
}: {
  media: MediaOutput;
  sessionId: string;
  getMediaSrc: (path: string, sessionId: string) => string;
}) => {
  const { isDownloading, downloadMedia } = useMediaDownload(
    media,
    sessionId,
    getMediaSrc
  );

  return (
    <Button
      className="text-muted-foreground hover:text-foreground"
      onClick={downloadMedia}
      size="sm"
      variant="ghost"
      title={
        media.type === 'video' && isYouTubeUrl(media.path)
          ? 'Open in YouTube'
          : 'Download media'
      }
      disabled={isDownloading}
    >
      <Download className="size-4" />
    </Button>
  );
};
