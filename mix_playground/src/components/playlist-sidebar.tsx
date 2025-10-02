import { Image, Music, Play, Video } from 'lucide-react';
import { ScrollArea, ScrollBar } from '@/components/ui/scroll-area';
import type { MediaOutput } from '@/types/media';
import { convertToAssetServerUrl } from '@/utils/assetServer';

// Helper function to detect URLs
const isURL = (path: string): boolean => {
  return path.startsWith('http://') || path.startsWith('https://');
};

// Helper function to detect local asset server URLs (localhost URLs that need thumbnail params)
const isLocalAssetServerURL = (path: string): boolean => {
  return path.startsWith('http://localhost:') || path.startsWith('https://localhost:');
};

// Helper function to get media source URL
const getMediaSrc = (path: string, sessionId: string): string => {
  return isURL(path) ? path : convertToAssetServerUrl(path, sessionId);
};

// Helper function to extract YouTube video ID
const getYouTubeVideoId = (url: string): string | null => {
  const patterns = [
    /(?:https?:\/\/)?(?:www\.)?(?:youtube\.com\/watch\?v=|youtu\.be\/)([a-zA-Z0-9_-]+)/,
    /(?:https?:\/\/)?(?:www\.)?youtube\.com\/embed\/([a-zA-Z0-9_-]+)/
  ];

  for (const pattern of patterns) {
    const match = url.match(pattern);
    if (match && match[1]) {
      return match[1];
    }
  }
  return null;
};

// Helper function to get YouTube thumbnail URL
const getYouTubeThumbnail = (url: string): string | null => {
  const videoId = getYouTubeVideoId(url);
  if (videoId) {
    return `https://img.youtube.com/vi/${videoId}/maxresdefault.jpg`;
  }
  return null;
};

const formatTime = (seconds: number): string => {
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = Math.floor(seconds % 60);
  return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
};

interface PlaylistSidebarProps {
  mediaOutputs: MediaOutput[];
  selectedIndex: number;
  onSelect: (index: number) => void;
  sessionId: string;
}

export const PlaylistSidebar = ({
  mediaOutputs,
  selectedIndex,
  onSelect,
  sessionId,
}: PlaylistSidebarProps) => {
  const getMediaIcon = (type: string) => {
    switch (type) {
      case 'video':
        return <Video className="h-4 w-4" />;
      case 'audio':
        return <Music className="h-4 w-4" />;
      case 'image':
        return <Image className="h-4 w-4" />;
      default:
        return <Play className="h-4 w-4" />;
    }
  };

  const renderThumbnail = (media: MediaOutput) => {

    if (media.type === 'image') {
      const imageUrl = getMediaSrc(media.path, sessionId);
      // Only add thumbnail parameter for local files, use URL directly for remote images
      const thumbnailUrl = isURL(media.path) ? imageUrl : `${imageUrl}?thumb=100`;


      return (
        <div className="h-12 w-16 flex-shrink-0 overflow-hidden rounded bg-stone-800">
          <img
            alt=""
            className="h-full w-full object-cover"
            onError={(e) => {
              console.error('Image thumbnail failed to load:', {
                src: e.currentTarget.src,
                media: media,
                sessionId: sessionId,
                error: e
              });
            }}
            src={thumbnailUrl}
          />
        </div>
      );
    }

    if (media.type === 'video') {
      // Check if it's a YouTube URL first
      const youtubeThumbnail = getYouTubeThumbnail(media.path);
      if (youtubeThumbnail) {
        return (
          <div className="relative h-12 w-16 flex-shrink-0 overflow-hidden rounded bg-stone-800">
            <img
              alt={`${media.title} thumbnail`}
              className="h-full w-full object-cover"
              onError={(e) => {
                console.error('YouTube thumbnail failed to load:', {
                  src: e.currentTarget.src,
                  media: media,
                  thumbnailUrl: youtubeThumbnail,
                  error: e
                });
              }}
              src={youtubeThumbnail}
            />
            <div className="absolute inset-0 flex items-center justify-center bg-black/20">
              <Play className="h-3 w-3 text-white drop-shadow-sm" />
            </div>
          </div>
        );
      }

      // Regular video handling
      // Use sourceVideo for highlights, fallback to path for regular videos
      const videoPath = media.sourceVideo || media.path;
      const videoUrl = getMediaSrc(videoPath, sessionId);

      // Add thumbnail and time parameters for local files (including localhost asset server URLs)
      let thumbnailUrl = videoUrl;
      if (!isURL(videoPath) || isLocalAssetServerURL(videoPath)) {
        thumbnailUrl = `${videoUrl}?thumb=100`;
        if (media.startTime !== undefined && typeof media.startTime === 'number' && media.startTime >= 0) {
          thumbnailUrl += `&time=${media.startTime}`;
        }
      }

      return (
        <div className="relative h-12 w-16 flex-shrink-0 overflow-hidden rounded bg-stone-800">
          <img
            alt={`${media.title} thumbnail`}
            className="h-full w-full object-cover"
            onError={(e) => {
              console.error('Video thumbnail failed to load:', JSON.stringify({
                src: e.currentTarget.src,
                media: media,
                sessionId: sessionId,
                videoPath: media.sourceVideo || media.path,
                startTime: media.startTime,
                thumbnailUrl: thumbnailUrl,
                errorType: e.type,
                errorTarget: e.target
              }, null, 2));
            }}
            src={thumbnailUrl}
          />
          <div className="absolute inset-0 flex items-center justify-center bg-black/20">
            <Play className="h-3 w-3 text-white drop-shadow-sm" />
          </div>
        </div>
      );
    }

    // Fallback  - show icon in colored box
    return (
      <div className="flex h-12 w-16 flex-shrink-0 items-center justify-center rounded bg-stone-700/50">
        {getMediaIcon(media.type)}
      </div>
    );
  };

  return (
    <div>
      <h4 className="mb-3 font-medium text-muted-foreground text-sm">
        Playlist ({mediaOutputs.length})
      </h4>
      <ScrollArea className="w-full">
        <div className="flex gap-3 pb-2">
          {mediaOutputs.map((media, index) => (
            <button
              className={`min-w-32 rounded-md bg-stone-700/30 p-2 text-left transition-colors ${selectedIndex === index
                ? ' border border-primary/30'
                : 'hover:bg-stone-700/30'
                }`}
              key={index}
              onClick={() => onSelect(index)}
            >
              <div className="flex items-center gap-3">
                {renderThumbnail(media)}
                <div className="min-w-0 flex-1">
                  <div className="mb-1 font-medium text-sm">{media.title}</div>
                  {media.sourceVideo &&
                    media.startTime !== undefined &&
                    media.duration !== undefined && (
                      <div className="mb-1 text-muted-foreground text-xs">
                        {formatTime(media.startTime)} -{' '}
                        {formatTime(media.startTime + media.duration)}
                      </div>
                    )}
                  {media.description && (
                    <div className="line-clamp-2 text-muted-foreground text-xs">
                      {media.description}
                    </div>
                  )}
                </div>
              </div>
            </button>
          ))}
        </div>
        <ScrollBar orientation="horizontal" />
      </ScrollArea>
    </div>
  );
};
