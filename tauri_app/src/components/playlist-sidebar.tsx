import { convertFileSrc } from '@tauri-apps/api/core';
import { Image, Music, Play, Video } from 'lucide-react';
import { ScrollArea, ScrollBar } from '@/components/ui/scroll-area';
import type { MediaOutput } from '@/types/media';

const formatTime = (seconds: number): string => {
  const minutes = Math.floor(seconds / 60);
  const remainingSeconds = Math.floor(seconds % 60);
  return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
};

interface PlaylistSidebarProps {
  mediaOutputs: MediaOutput[];
  selectedIndex: number;
  onSelect: (index: number) => void;
}

export const PlaylistSidebar = ({
  mediaOutputs,
  selectedIndex,
  onSelect,
}: PlaylistSidebarProps) => {
  const getMediaIcon = (type: string) => {
    switch (type) {
      case 'video':
      case 'remotion_title':
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
      return (
        <div className="h-12 w-16 flex-shrink-0 overflow-hidden rounded bg-stone-800">
          <img
            alt=""
            className="h-full w-full object-cover"
            onError={(e) => {
              console.error('Image failed to load:', media.path);
              e.currentTarget.style.display = 'none';
              const parent = e.currentTarget.parentElement;
              if (parent) {
                parent.className =
                  'w-16 h-12 rounded bg-stone-700 flex-shrink-0 flex items-center justify-center';
                parent.innerHTML =
                  '<svg class="w-4 h-4 text-stone-400" fill="currentColor" viewBox="0 0 24 24"><path d="M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z"/></svg>';
              }
            }}
            onLoad={() => {}}
            src={convertFileSrc(media.path)}
          />
        </div>
      );
    }

    if (media.type === 'video') {
      // Use sourceVideo for highlights, fallback to path for regular videos
      const videoSrc = convertFileSrc(media.sourceVideo || media.path);
      // Use startTime for highlights, fallback to 1 second for regular videos
      const thumbnailTime = media.startTime !== undefined ? media.startTime : 1;

      return (
        <div className="relative h-12 w-16 flex-shrink-0 overflow-hidden rounded bg-stone-800">
          <video
            className="h-full w-full object-cover"
            onError={(e) => {
              console.error(
                'Video failed to load:',
                media.sourceVideo || media.path,
                e
              );
              e.currentTarget.style.display = 'none';
              const parent = e.currentTarget.parentElement;
              if (parent) {
                parent.className =
                  'w-16 h-12 rounded bg-stone-700 flex-shrink-0 flex items-center justify-center';
                parent.innerHTML =
                  '<svg class="w-4 h-4 text-stone-400" fill="currentColor" viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg>';
              }
            }}
            onLoadedMetadata={(e) => {
              // Seek to startTime for highlights or 1 second for regular videos
              try {
                e.currentTarget.currentTime = thumbnailTime;
              } catch (err) {
                console.error('Error seeking video:', err);
              }
            }}
            poster=""
            preload="metadata"
            src={videoSrc}
          />
          <div className="absolute inset-0 flex items-center justify-center bg-black/20">
            <Play className="h-3 w-3 text-white drop-shadow-sm" />
          </div>
        </div>
      );
    }

    // Fallback for audio/remotion_title - show icon in colored box
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
              className={`min-w-32 rounded-md bg-stone-700/30 p-2 text-left transition-colors ${
                selectedIndex === index
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
