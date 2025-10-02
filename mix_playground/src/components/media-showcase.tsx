import { isYouTubeUrl } from '@/utils/videoUrlDetection';
import type { MediaOutput } from '@/types/media';
import { useState } from 'react';
import { MediaDownloadButton } from './media-download-button';
import { GsapAnimationPreview } from './gsap/GsapAnimationPreview';
import { LazyVideoPlayer } from './LazyVideoPlayer';
import { CsvViewer } from './CsvViewer';
import { PlaylistSidebar } from './playlist-sidebar';

// Main Media Player Component
const MainMediaPlayer = ({
  media,
  sessionId,
  getMediaSrc,
}: {
  media: MediaOutput;
  sessionId: string;
  getMediaSrc: (path: string, sessionId: string) => string;
}) => {
  return (
    <div className="mb-2 space-y-2">
      <div className="flex items-start justify-between gap-2">
        <div className="flex-1">
          <h3 className="font-semibold">{media.title}</h3>
          {media.description && (
            <p className="mt-1 text-muted-foreground text-sm">
              {media.description}
            </p>
          )}
        </div>
        <MediaDownloadButton media={media} sessionId={sessionId} getMediaSrc={getMediaSrc} />
      </div>

      {media.type === 'image' && (
        <img
          alt={media.title}
          className="aspect-auto max-h-120  object-contain"
          onError={(e) => {
            e.currentTarget.style.display = 'none';
            const fallback = e.currentTarget
              .nextElementSibling as HTMLElement;
            if (fallback) fallback.style.display = 'block';
          }}
          src={getMediaSrc(media.path, sessionId)}
        />
      )}

      {media.type === 'video' && (
        <>
          {isYouTubeUrl(media.path) ? (
            <div className="overflow-hidden rounded-md">
              <iframe
                src={(() => {
                  const baseUrl = getMediaSrc(media.path, sessionId);
                  if (media.startTime !== undefined || media.duration !== undefined) {
                    try {
                      const url = new URL(baseUrl);
                      if (media.startTime !== undefined) {
                        url.searchParams.set('start', media.startTime.toString());
                      }
                      if (media.duration !== undefined && media.startTime !== undefined) {
                        url.searchParams.set('end', (media.startTime + media.duration).toString());
                      }
                      return url.toString();
                    } catch {
                      return baseUrl;
                    }
                  }
                  return baseUrl;
                })()}
                title={media.title}
                frameBorder="0"
                allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
                referrerPolicy="strict-origin-when-cross-origin"
                allowFullScreen
                className="aspect-video w-full bg-black"
                onError={(e) => {
                  e.currentTarget.style.display = 'none';
                  const fallback = e.currentTarget
                    .nextElementSibling as HTMLElement;
                  if (fallback) fallback.style.display = 'block';
                }}
              />
              <div
                className="flex h-48 items-center justify-center bg-stone-700 text-stone-400"
                style={{ display: 'none' }}
              >
                Failed to load YouTube video: {media.path}
              </div>
            </div>
          ) : (
            <LazyVideoPlayer
              media={media}
              sessionId={sessionId}
            />
          )}
        </>
      )}

      {media.type === 'audio' && (
        <div className="rounded-md bg-stone-700/30 p-4">
          <audio
            className="w-full"
            controls
            onError={(e) => {
              e.currentTarget.style.display = 'none';
              const fallback = e.currentTarget
                .nextElementSibling as HTMLElement;
              if (fallback) fallback.style.display = 'block';
            }}
            preload="metadata"
            src={getMediaSrc(media.path, sessionId)}
          >
            Your browser does not support the audio tag.
          </audio>
          <div
            className="mt-2 text-center text-stone-400"
            style={{ display: 'none' }}
          >
            Failed to load audio: {media.path}
          </div>
        </div>
      )}

      {media.type === 'gsap_animation' && media.config && (
        <GsapAnimationPreview config={media.config as any} />
      )}

      {media.type === 'pdf' && (
        <div className="overflow-hidden rounded-md">
          <iframe
            src={getMediaSrc(media.path, sessionId)}
            title={media.title}
            frameBorder="0"
            className="aspect-[4/5] w-full bg-white"
            onError={(e) => {
              e.currentTarget.style.display = 'none';
              const fallback = e.currentTarget
                .nextElementSibling as HTMLElement;
              if (fallback) fallback.style.display = 'block';
            }}
          />
          <div
            className="flex h-48 items-center justify-center bg-stone-700 text-stone-400"
            style={{ display: 'none' }}
          >
            Failed to load PDF document: {media.path}
          </div>
        </div>
      )}

      {media.type === 'csv' && (
        <CsvViewer
          url={getMediaSrc(media.path, sessionId)}
          title={media.title}
        />
      )}
    </div>
  );
};

// Media Showcase Component
export const MediaShowcase = ({
  mediaOutputs,
  sessionId,
  getMediaSrc,
}: {
  mediaOutputs: MediaOutput[];
  sessionId: string;
  getMediaSrc: (path: string, sessionId: string) => string;
}) => {
  const [selectedIndex, setSelectedIndex] = useState(0);

  if (!mediaOutputs || mediaOutputs.length === 0) return null;

  // Single media file - show directly
  if (mediaOutputs.length === 1) {
    return <MainMediaPlayer media={mediaOutputs[0]} sessionId={sessionId} getMediaSrc={getMediaSrc} />;
  }

  // Multiple media files - show player + playlist
  return (
    <div className="space-y-4">
      <MainMediaPlayer media={mediaOutputs[selectedIndex]} sessionId={sessionId} getMediaSrc={getMediaSrc} />
      <PlaylistSidebar
        mediaOutputs={mediaOutputs}
        onSelect={setSelectedIndex}
        selectedIndex={selectedIndex}
        sessionId={sessionId}
      />
    </div>
  );
};
