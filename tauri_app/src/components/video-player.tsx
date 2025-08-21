import { convertToAssetServerUrl } from '@/utils/assetServer';
import { Pause, PictureInPicture, Play, RotateCcw } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import type { VideoPlayerProps } from '@/types/media';

export const VideoPlayer = ({
  path,
  title,
  description,
  startTime,
  duration,
  workingDirectory,
}: VideoPlayerProps) => {
  const [isLoading, setIsLoading] = useState(true);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const [hasError, setHasError] = useState(false);
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    setIsLoading(true);
    setHasError(false);
  }, [path]);

  // Create URL with media fragment for segments
  const getVideoSrc = () => {
    const baseSrc = convertToAssetServerUrl(path, workingDirectory);

    // For video segments, add media fragment to constrain playback
    if (startTime !== undefined && duration !== undefined) {
      const endTime = startTime + duration;
      return `${baseSrc}#t=${startTime},${endTime}`;
    }

    return baseSrc;
  };

  // Get segment info
  const isSegment = startTime !== undefined && duration !== undefined;

  // Format time for display (segment time, not absolute time)
  const formatTime = (seconds: number) => {
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = Math.floor(seconds % 60);
    return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
  };

  // Get segment time (0-based from start of segment)
  const getSegmentTime = (videoCurrentTime: number) => {
    if (!isSegment || !startTime) return videoCurrentTime;
    return Math.max(0, videoCurrentTime - startTime);
  };

  // Convert segment progress (0-100%) to actual video time
  const segmentProgressToVideoTime = (progress: number) => {
    if (!isSegment || !startTime || !duration) return 0;
    return startTime + (progress / 100) * duration;
  };

  // Get current segment progress (0-100%)
  const getSegmentProgress = () => {
    if (!isSegment || !duration) return 0;
    const segmentTime = getSegmentTime(currentTime);
    return Math.min(100, (segmentTime / duration) * 100);
  };

  const handlePlayPause = () => {
    if (!videoRef.current) return;

    if (isPlaying) {
      videoRef.current.pause();
    } else {
      videoRef.current.play();
    }
  };

  const handleProgressClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (!videoRef.current) return;

    const rect = e.currentTarget.getBoundingClientRect();
    const clickX = e.clientX - rect.left;
    const progress = (clickX / rect.width) * 100;

    let newTime: number;

    if (isSegment) {
      // For segments, use existing logic
      newTime = segmentProgressToVideoTime(progress);
    } else {
      // For non-segments, calculate based on total video duration
      const videoDuration = videoRef.current.duration || 0;
      newTime = (progress / 100) * videoDuration;
    }

    videoRef.current.currentTime = newTime;
  };

  const handleReplay = () => {
    if (!videoRef.current) return;

    // Reset to start (or segment start for segments)
    const resetTime = isSegment && startTime !== undefined ? startTime : 0;
    videoRef.current.currentTime = resetTime;
  };

  const handlePictureInPicture = async () => {
    if (!videoRef.current) return;

    try {
      if (document.pictureInPictureElement) {
        await document.exitPictureInPicture();
      } else {
        await videoRef.current.requestPictureInPicture();
      }
    } catch (error) {
      console.error('Picture-in-Picture failed:', error);
    }
  };

  return (
    <div className="">
      {(title || description) && (
        <div className="mb-3">
          {title && <h3 className="font-semibold">{title}</h3>}
          {description && (
            <p className="mt-1 text-muted-foreground text-sm">{description}</p>
          )}
        </div>
      )}

      <div className="relative max-w-xl rounded-md">
        {isLoading && <Skeleton className="aspect-video" />}
        <video
          className="aspect-video rounded-md bg-black"
          onError={() => {
            setHasError(true);
            setIsLoading(false);
          }}
          onLoadedData={() => {
            setIsLoading(false);
            setHasError(false);
          }}
          onPause={() => setIsPlaying(false)}
          onPlay={() => setIsPlaying(true)}
          onSeeking={(e) => {
            // Prevent seeking outside the segment range
            if (isSegment && startTime !== undefined && duration !== undefined) {
              const video = e.currentTarget;
              const endTime = startTime + duration;

              if (video.currentTime < startTime) {
                video.currentTime = startTime;
              } else if (video.currentTime > endTime) {
                video.currentTime = endTime;
              }
            }
          }}
          onTimeUpdate={(e) => {
            const video = e.currentTarget;
            setCurrentTime(video.currentTime);

            // Additional safeguard to keep playback within bounds
            if (isSegment && startTime !== undefined && duration !== undefined) {
              const endTime = startTime + duration;
              if (video.currentTime >= endTime) {
                video.pause();
                video.currentTime = endTime;
              }
            }
          }}
          preload="auto"
          ref={videoRef}
          src={getVideoSrc()}
        >
          Your browser does not support the video tag.
        </video>

        {/* Custom Controls */}
        {!isLoading && (
          <div className="absolute right-0 bottom-0 left-0 p-2">
            <div className="flex items-center gap-3">
              {/* Play/Pause Button */}

              <div className="flex items-center">
                <Button onClick={handlePlayPause} size="sm" variant="ghost">
                  {isPlaying ? (
                    <Pause className="size-4" />
                  ) : (
                    <Play className="size-4" />
                  )}
                </Button>

                {/* Replay Button */}
                <Button
                  onClick={handleReplay}
                  size="sm"
                  title="Replay"
                  variant="ghost"
                >
                  <RotateCcw className="size-4" />
                </Button>
              </div>

              {/* Progress Bar - show for all videos */}
              <div className="flex flex-1 items-center gap-2">
                <span className="font-mono text-white text-xs">
                  {formatTime(
                    isSegment ? getSegmentTime(currentTime) : currentTime
                  )}
                </span>

                <div
                  className="relative h-1 flex-1 cursor-pointer rounded bg-white/30"
                  onClick={handleProgressClick}
                >
                  <div
                    className="h-full rounded bg-white"
                    style={{
                      width: `${isSegment ? getSegmentProgress() : videoRef.current?.duration ? (currentTime / videoRef.current.duration) * 100 : 0}%`,
                    }}
                  />
                </div>

                <span className="font-mono text-white text-xs">
                  {formatTime(
                    isSegment && duration !== undefined
                      ? duration
                      : videoRef.current?.duration || 0
                  )}
                </span>
              </div>

              {/* Picture-in-Picture Button */}
              <Button
                onClick={handlePictureInPicture}
                size="sm"
                title="Picture-in-Picture"
                variant="ghost"
              >
                <PictureInPicture className="size-4" />
              </Button>
            </div>
          </div>
        )}

        {/* Error overlay - only show if video failed and not loading */}
        {hasError && !isLoading && (
          <div className="absolute inset-0 flex items-center justify-center bg-stone-800/80 text-stone-300 text-sm rounded-md">
            <div className="text-center">
              <p>Video temporarily unavailable</p>
              <button
                onClick={() => {
                  setHasError(false);
                  setIsLoading(true);
                  if (videoRef.current) {
                    videoRef.current.load();
                  }
                }}
                className="mt-2 text-white underline hover:no-underline"
              >
                Retry
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
