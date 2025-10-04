import { useCallback, useState } from 'react';
import { isYouTubeUrl } from '@/utils/videoUrlDetection';
import type { MediaOutput } from '@/types/media';
import { toast } from 'sonner';

// Helper to get file extension from media type and path
const getFileExtension = (media: MediaOutput): string => {
  // Try to extract extension from path first
  const pathMatch = media.path.match(/\.([^./?#]+)(?:[?#]|$)/);
  if (pathMatch) {
    return `.${pathMatch[1]}`;
  }

  // Fallback to media type
  const extensionMap: Record<string, string> = {
    image: '.jpg',
    video: '.mp4',
    audio: '.mp3',
    pdf: '.pdf',
    csv: '.csv',
    gsap_animation: '.json',
  };

  return extensionMap[media.type] || '';
};

export const useMediaDownload = (
  media: MediaOutput,
  sessionId: string,
  getMediaSrc: (path: string, sessionId: string) => string
) => {
  const [isDownloading, setIsDownloading] = useState(false);

  const downloadMedia = useCallback(async () => {
    // For YouTube videos, open in new tab instead of downloading
    if (media.type === 'video' && isYouTubeUrl(media.path)) {
      window.open(media.path, '_blank');
      return;
    }

    setIsDownloading(true);

    try {
      // For GSAP animations, download the config as JSON
      if (media.type === 'gsap_animation' && media.config) {
        const configJson = JSON.stringify(media.config, null, 2);
        const blob = new Blob([configJson], { type: 'application/json' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        const filename = `${media.title || 'animation'}.json`;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);

        toast.success('Download complete', {
          description: filename,
        });

        setIsDownloading(false);
        return;
      }

      // For all other media types, fetch as blob and trigger download
      const mediaUrl = getMediaSrc(media.path, sessionId);
      const response = await fetch(mediaUrl);

      if (!response.ok) {
        throw new Error(`Failed to fetch media: ${response.statusText}`);
      }

      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;

      // Create filename with proper extension
      const extension = getFileExtension(media);
      const filename = media.title
        ? media.title.includes('.')
          ? media.title
          : `${media.title}${extension}`
        : `media${extension}`;

      a.download = filename;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);

      toast.success('Download complete', {
        description: filename,
      });
    } catch (error) {
      console.error('Download failed:', error);
      toast.error('Download failed', {
        description: 'Opening in new tab instead',
      });
      // Fallback: try opening in new tab
      const mediaUrl = getMediaSrc(media.path, sessionId);
      window.open(mediaUrl, '_blank');
    } finally {
      setIsDownloading(false);
    }
  }, [media, sessionId, getMediaSrc]);

  return {
    isDownloading,
    downloadMedia,
  };
};
