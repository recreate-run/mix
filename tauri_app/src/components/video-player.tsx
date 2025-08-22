import { convertToAssetServerUrl } from '@/utils/assetServer';
import { Pause, PictureInPicture, Play } from 'lucide-react';
import { useEffect, useRef, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import type { VideoPlayerProps } from '@/types/media';
import { IconPlayerPauseFilled, IconPlayerPlayFilled } from '@tabler/icons-react';

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
  const [showControls, setShowControls] = useState(true);
  const [controlsTimeout, setControlsTimeout] = useState<NodeJS.Timeout | null>(null);
  const [isVertical, setIsVertical] = useState(false);
  const videoRef = useRef<HTMLVideoElement>(null);

  // Use refs for internal state to avoid re-render cycles
  const hasInitialized = useRef(false);
  const isSegment = startTime !== undefined && duration !== undefined;

  // Auto-hide controls after inactivity
  const resetControlsTimeout = () => {
    if (controlsTimeout) {
      clearTimeout(controlsTimeout);
    }
    setShowControls(true);
    const timeout = setTimeout(() => {
      setShowControls(false);
    }, 3000); // Hide after 3 seconds
    setControlsTimeout(timeout);
  };

  // Handle video container click to toggle controls
  const handleVideoClick = () => {
    if (showControls) {
      setShowControls(false);
      if (controlsTimeout) {
        clearTimeout(controlsTimeout);
        setControlsTimeout(null);
      }
    } else {
      resetControlsTimeout();
    }
  };

  useEffect(() => {
    setIsLoading(true);
    setHasError(false);
    setIsVertical(false);
    hasInitialized.current = false;
    resetControlsTimeout();
  }, [path]);

  // Handle startTime/duration changes for segments
  // Note: This assumes the same video file for all segments (path doesn't change)
  // If path changes to a different video, the existing useEffect with [path] dependency handles that
  useEffect(() => {
    const video = videoRef.current;
    if (video && video.readyState >= 1 && isSegment && startTime !== undefined) {
      // Update current time when segment parameters change
      video.currentTime = startTime;
      hasInitialized.current = true; // Mark as initialized to prevent duplicate seeks
    }
  }, [startTime, duration, isSegment]);

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (controlsTimeout) {
        clearTimeout(controlsTimeout);
      }
    };
  }, [controlsTimeout]);

  // Handle mouse interactions to control auto-hide behavior
  const handleMouseEnter = () => {
    if (controlsTimeout) {
      clearTimeout(controlsTimeout);
      setControlsTimeout(null);
    }
    setShowControls(true);
  };

  const handleMouseLeave = () => {
    resetControlsTimeout();
  };

  // Simple URL without media fragments (more reliable)
  const getVideoSrc = () => {
    return convertToAssetServerUrl(path, workingDirectory);
  };

  // Format time for display
  const formatTime = (seconds: number) => {
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = Math.floor(seconds % 60);
    return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
  };

  // Get display time (segment-relative for segments)
  const getDisplayTime = (videoTime: number) => {
    if (!isSegment || startTime === undefined) return videoTime;
    return Math.max(0, videoTime - startTime);
  };

  // Get display duration (segment duration for segments)
  const getDisplayDuration = () => {
    if (!isSegment) {
      return videoRef.current?.duration || 0;
    }
    return duration || 0;
  };

  // Get display progress (0-100%)
  const getDisplayProgress = () => {
    const displayDuration = getDisplayDuration();
    if (displayDuration === 0) return 0;

    const displayTime = getDisplayTime(currentTime);
    return Math.min(100, (displayTime / displayDuration) * 100);
  };

  const handlePlayPause = () => {
    if (!videoRef.current) return;

    if (isPlaying) {
      videoRef.current.pause();
    } else {
      videoRef.current.play();
    }
    if (isVertical) {
      resetControlsTimeout();
    }
  };

  const handleProgressClick = (e: React.MouseEvent<HTMLDivElement>) => {
    const video = videoRef.current;
    if (!video?.duration || !isFinite(video.duration)) return;

    const rect = e.currentTarget.getBoundingClientRect();
    const clickX = e.clientX - rect.left;
    const progress = Math.max(0, Math.min(100, (clickX / rect.width) * 100));

    let newTime: number;

    if (isSegment && startTime !== undefined && duration !== undefined) {
      // Pre-clamp to segment boundaries
      newTime = startTime + (progress / 100) * duration;
      newTime = Math.max(startTime, Math.min(startTime + duration, newTime));
    } else {
      newTime = (progress / 100) * video.duration;
    }

    video.currentTime = newTime;
    if (isVertical) {
      resetControlsTimeout();
    }
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
    if (isVertical) {
      resetControlsTimeout();
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

      <div
        className={`relative rounded-md ${isVertical ? 'max-w-64' : 'max-w-4xl'} mx-auto`}
        {...(isVertical && {
          onClick: handleVideoClick,
          onMouseEnter: handleMouseEnter,
          onMouseLeave: handleMouseLeave
        })}
      >
        {isLoading && <Skeleton className="aspect-auto" />}
        <video
          className="aspect-auto rounded-xl bg-black w-full"
          onError={() => {
            setHasError(true);
            setIsLoading(false);
          }}
          onLoadedMetadata={() => {
            setIsLoading(false);
            setHasError(false);

            // Detect video orientation
            const video = videoRef.current;
            if (video) {
              setIsVertical(video.videoHeight > video.videoWidth);
            }

            // Set initial position for segments (one-time setup)
            if (isSegment && startTime !== undefined && !hasInitialized.current) {
              if (video) {
                video.currentTime = startTime;
                hasInitialized.current = true;
              }
            }
          }}
          onPause={() => setIsPlaying(false)}
          onPlay={() => setIsPlaying(true)}
          onSeeked={() => {
            // Gentle boundary correction after seeking (no loops)
            if (isSegment && startTime !== undefined && duration !== undefined) {
              const video = videoRef.current;
              if (video) {
                const endTime = startTime + duration;
                if (video.currentTime < startTime) {
                  video.currentTime = startTime;
                } else if (video.currentTime > endTime) {
                  video.currentTime = endTime;
                }
              }
            }
          }}
          onTimeUpdate={(e) => {
            const video = e.currentTarget;
            setCurrentTime(video.currentTime);

            // Only pause at segment end, don't modify time during update
            if (isSegment && startTime !== undefined && duration !== undefined) {
              const endTime = startTime + duration;
              if (video.currentTime >= endTime && !video.paused) {
                video.pause();
              }
            }
          }}
          preload="auto"
          ref={videoRef}
          src={getVideoSrc()}
        >
          Your browser does not support the video tag.
        </video>

        {/* Controls */}
        {!isLoading && (isVertical ? showControls : true) && (
          <>
            {isVertical ? (
              /* Vertical Layout: Distributed Controls */
              <>
                {/* Top-Left Time Display */}
                <div className="absolute top-0 left-0 right-0 w-full flex justify-between items-center px-2">
                  <div className="bg-black/60 backdrop-blur-sm rounded-md px-2 py-1">
                    <span className="text-white text-sm">
                      {formatTime(getDisplayTime(currentTime))} / {formatTime(getDisplayDuration())}
                    </span>
                  </div>

                  <Button
                    onClick={handlePictureInPicture}
                    size="icon"
                    title="Picture-in-Picture"
                    className="bg-black/60 backdrop-blur-sm hover:bg-black/80 text-white border-0 h-11 w-11 rounded-full"
                  >
                    <PictureInPicture />
                  </Button>
                </div>

                {/* Center Play/Pause Button */}
                <div className="absolute inset-0 flex items-center justify-center">
                  <Button
                    onClick={handlePlayPause}
                    size="lg"
                    variant="ghost"
                    className="bg-black/60 backdrop-blur-sm hover:bg-black/80 text-white border-0 h-16 w-16 rounded-full"
                  >
                    {isPlaying ? (
                      <IconPlayerPauseFilled className="size-8" />
                    ) : (
                      <IconPlayerPlayFilled className="size-8 ml-1" />
                    )}
                  </Button>
                </div>

                {/* Bottom Progress Bar */}
                <div className="absolute bottom-0 left-0 right-0 p-4">
                  <div
                    className="relative h-1 w-full cursor-pointer rounded-full bg-white/30 backdrop-blur-sm"
                    onClick={handleProgressClick}
                  >
                    <div
                      className="h-full rounded-full bg-white transition-all duration-150"
                      style={{
                        width: `${getDisplayProgress()}%`,
                      }}
                    />
                    <div
                      className="absolute top-1/2 -translate-y-1/2 w-3 h-3 bg-white rounded-full shadow-lg transition-all duration-150"
                      style={{
                        left: `calc(${getDisplayProgress()}% - 6px)`,
                      }}
                    />
                  </div>
                </div>
              </>
            ) : (
              /* Horizontal Layout: Single Bottom Row */
              <div className="absolute bottom-0 left-0 right-0 p-2">
                <div className="flex items-center gap-2">
                  {/* Play/Pause Button */}
                  <Button
                    onClick={handlePlayPause}
                    size="icon"
                    variant="ghost"
                    className="hover:bg-white/20 text-white border-0  rounded-full shrink-0"
                  >
                    {isPlaying ? (
                      <IconPlayerPauseFilled className="size-6" />
                    ) : (
                      <IconPlayerPlayFilled className="size-6" />
                    )}
                  </Button>

                  {/* Progress Bar */}
                  <div
                    className="flex-1 relative h-1 cursor-pointer rounded-full bg-neutral-600/40 backdrop-blur-sm rounded-lg py-[5px]"
                    onClick={handleProgressClick}
                  >
                    <div
                      className="h-full rounded-full bg-white transition-all duration-150"
                      style={{
                        width: `${getDisplayProgress()}%`,
                      }}
                    />
                    <div
                      className="absolute top-1/2 -translate-y-1/2 size-3 bg-white rounded-full shadow-lg transition-all duration-150"
                      style={{
                        left: `calc(${getDisplayProgress()}% - 6px)`,
                      }}
                    />
                  </div>

                  {/* Time Display */}
                  <div className="bg-black/40 p-1 rounded-md text-white text-xs font-medium shrink-0">
                    {formatTime(getDisplayTime(currentTime))} / {formatTime(getDisplayDuration())}
                  </div>

                  {/* Picture-in-Picture Button */}
                  <Button
                    onClick={handlePictureInPicture}
                    size="icon"
                    variant={"ghost"} title="Picture-in-Picture"
                    className="hover:bg-white/20 text-white border-0 h-8 w-8 rounded-full shrink-0"
                  >
                    <PictureInPicture className="size-4" />
                  </Button>
                </div>
              </div>
            )}
          </>
        )}

        {/* Error overlay - only show if video failed and not loading */}
        {hasError && !isLoading && (
          <div className="absolute inset-0 flex items-center justify-center bg-stone-800/80 backdrop-blur-sm text-stone-300 text-sm rounded-md">
            <div className="text-center">
              <p>Video temporarily unavailable</p>
              <Button
                onClick={() => {
                  setHasError(false);
                  setIsLoading(true);
                  if (videoRef.current) {
                    videoRef.current.load();
                  }
                  if (isVertical) {
                    resetControlsTimeout();
                  }
                }}
                variant="ghost"
                className="mt-3 text-white underline hover:no-underline hover:bg-transparent"
              >
                Retry
              </Button>
            </div>
          </div>
        )}
      </div>
    </div >
  );
};
