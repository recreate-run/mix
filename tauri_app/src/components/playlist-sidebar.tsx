import { Image, Video, Music, Play } from 'lucide-react';
import { ScrollArea } from '@/components/ui/scroll-area';
import { convertFileSrc } from '@tauri-apps/api/core';

type MediaOutput = {
  path: string;
  type: 'image' | 'video' | 'audio' | 'remotion_title';
  title: string;
  description?: string;
  config?: any;
};

interface PlaylistSidebarProps {
  mediaOutputs: MediaOutput[];
  selectedIndex: number;
  onSelect: (index: number) => void;
}

export const PlaylistSidebar = ({ 
  mediaOutputs, 
  selectedIndex, 
  onSelect 
}: PlaylistSidebarProps) => {
  const getMediaIcon = (type: string) => {
    switch (type) {
      case 'video':
      case 'remotion_title':
        return <Video className="w-4 h-4" />;
      case 'audio':
        return <Music className="w-4 h-4" />;
      case 'image':
        return <Image className="w-4 h-4" />;
      default:
        return <Play className="w-4 h-4" />;
    }
  };

  const renderThumbnail = (media: MediaOutput) => {
    
    if (media.type === 'image') {
      return (
        <div className="w-16 h-12 rounded overflow-hidden bg-stone-800 flex-shrink-0">
          <img
            src={convertFileSrc(media.path)}
            alt=""
            className="w-full h-full object-cover"
            onLoad={() => {}}
            onError={(e) => {
              console.error('Image failed to load:', media.path);
              e.currentTarget.style.display = 'none';
              const parent = e.currentTarget.parentElement;
              if (parent) {
                parent.className = 'w-16 h-12 rounded bg-stone-700 flex-shrink-0 flex items-center justify-center';
                parent.innerHTML = '<svg class="w-4 h-4 text-stone-400" fill="currentColor" viewBox="0 0 24 24"><path d="M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z"/></svg>';
              }
            }}
          />
        </div>
      );
    }
    
    if (media.type === 'video') {
      const videoSrc = convertFileSrc(media.path);
      
      return (
        <div className="w-16 h-12 rounded overflow-hidden bg-stone-800 flex-shrink-0 relative">
          <video
            src={videoSrc}
            className="w-full h-full object-cover"
            preload="metadata"
            poster=""
            onLoadedMetadata={(e) => {
              // Seek to 1 second to get a better thumbnail frame
              try {
                e.currentTarget.currentTime = 1;
              } catch (err) {
                console.error('Error seeking video:', err);
              }
            }}
            onError={(e) => {
              console.error('Video failed to load:', media.path, e);
              e.currentTarget.style.display = 'none';
              const parent = e.currentTarget.parentElement;
              if (parent) {
                parent.className = 'w-16 h-12 rounded bg-stone-700 flex-shrink-0 flex items-center justify-center';
                parent.innerHTML = '<svg class="w-4 h-4 text-stone-400" fill="currentColor" viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg>';
              }
            }}
          />
          <div className="absolute inset-0 flex items-center justify-center bg-black/20">
            <Play className="w-3 h-3 text-white drop-shadow-sm" />
          </div>
        </div>
      );
    }
    
    // Fallback for audio/remotion_title - show icon in colored box
    return (
      <div className="w-16 h-12 rounded bg-stone-700/50 flex-shrink-0 flex items-center justify-center">
        {getMediaIcon(media.type)}
      </div>
    );
  };

  return (
    <div className="py-4">
      <h4 className="text-sm font-medium mb-3 text-muted-foreground">
        Playlist ({mediaOutputs.length})
      </h4>
      <ScrollArea className="h-96">
        <div className="space-y-2 pr-2">
          {mediaOutputs.map((media, index) => (
            <button
              key={index}
              onClick={() => onSelect(index)}
              className={`w-full p-3 rounded-md text-left transition-colors ${
                selectedIndex === index 
                  ? 'bg-stone-700/50 border border-stone-600' 
                  : 'bg-stone-800/30 hover:bg-stone-700/30'
              }`}
            >
              <div className="flex items-center gap-3">
                {renderThumbnail(media)}
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium  mb-1">
                    {media.title}
                  </div>
                  {media.description && (
                    <div className="text-xs text-muted-foreground line-clamp-2">
                      {media.description}
                    </div>
                  )}
                </div>
              </div>
            </button>
          ))}
        </div>
      </ScrollArea>
    </div>
  );
};