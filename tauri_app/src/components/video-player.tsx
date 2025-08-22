import { convertToAssetServerUrl } from '@/utils/assetServer';
import { Pause, PictureInPicture, Play } from 'lucide-react';
import { useEffect, useRef, useState, useCallback } from 'react';
import { useThrottledCallback } from '@tanstack/react-pacer';
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
  const [currentTime, setCurrentTime] = useState(startTime || 0);
  const [hasError, setHasError] = useState(false);
  const [isVertical, setIsVertical] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const videoRef = useRef<HTMLVideoElement>(null);

  // Use refs for internal state to avoid re-render cycles
  const hasInitialized = useRef(false);
  const isSegment = startTime !== undefined && duration !== undefined;

  // Refs for smooth timeline animation
  const animationFrameRef = useRef<number>();
  const progressBarRef = useRef<HTMLDivElement>(null);
  const progressDotRef = useRef<HTMLDivElement>(null);
  const progressFillRef = useRef<HTMLDivElement>(null);
  const isDraggingRef = useRef(false);


  useEffect(() => {
    setIsLoading(true);
    setHasError(false);
    setIsVertical(false);
    hasInitialized.current = false;
    // Reset timeline position when video changes
    if (progressFillRef.current) {
      progressFillRef.current.style.width = '0%';
    }
    if (progressDotRef.current) {
      progressDotRef.current.style.left = '0%';
    }
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
      // Sync timeline after segment change
      syncTimelinePosition();
    }
  }, [startTime, duration, isSegment]);

  // Cleanup animation frame on unmount
  useEffect(() => {
    return () => {
      if (animationFrameRef.current) {
        cancelAnimationFrame(animationFrameRef.current);
      }
    };
  }, []);


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

  // Unified function to sync timeline position from video element
  const syncTimelinePosition = () => {
    const video = videoRef.current;
    if (!video || !isFinite(video.duration)) return;

    const displayDuration = getDisplayDuration();
    if (displayDuration === 0) return;

    const displayTime = getDisplayTime(video.currentTime);
    const progress = Math.min(100, (displayTime / displayDuration) * 100);

    // Update DOM directly
    if (progressFillRef.current) {
      progressFillRef.current.style.width = `${progress}%`;
    }
    if (progressDotRef.current) {
      progressDotRef.current.style.left = `${progress}%`;
    }
  };

  // Smooth animation loop for timeline
  const updateTimeline = () => {
    const video = videoRef.current;
    if (!video || video.paused || video.ended) {
      animationFrameRef.current = undefined;
      return;
    }

    // Use unified sync function
    syncTimelinePosition();

    // Continue animation loop
    animationFrameRef.current = requestAnimationFrame(updateTimeline);
  };

  // Start/stop animation based on play state
  const handlePlayStateChange = (playing: boolean) => {
    setIsPlaying(playing);

    if (playing && !animationFrameRef.current) {
      // Sync position before starting animation
      syncTimelinePosition();
      animationFrameRef.current = requestAnimationFrame(updateTimeline);
    } else if (!playing && animationFrameRef.current) {
      cancelAnimationFrame(animationFrameRef.current);
      animationFrameRef.current = undefined;
      // Sync position after stopping animation
      syncTimelinePosition();
    }
  };

  const handlePlayPause = () => {
    if (!videoRef.current) return;

    if (isPlaying) {
      videoRef.current.pause();
    } else {
      videoRef.current.play();
    }
  };

  // Throttled seek for smooth scrubbing (20fps)
  const throttledSeek = useThrottledCallback(
    (progress: number) => {
      const video = videoRef.current;
      if (!video?.duration || !isFinite(video.duration)) return;

      let newTime: number;
      if (isSegment && startTime !== undefined && duration !== undefined) {
        newTime = startTime + (progress / 100) * duration;
        newTime = Math.max(startTime, Math.min(startTime + duration, newTime));
      } else {
        newTime = (progress / 100) * video.duration;
      }

      video.currentTime = newTime;
    },
    { wait: 50 } // 50ms = 20fps updates
  );

  const handleProgressClick = (e: React.MouseEvent<HTMLDivElement>) => {
    // Don't handle click if it was a drag
    if (isDraggingRef.current) return;

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
    // Immediately sync visual position for responsive feedback
    syncTimelinePosition();
  };

  const handleProgressMouseDown = (e: React.MouseEvent<HTMLDivElement>) => {
    const video = videoRef.current;
    if (!video?.duration || !isFinite(video.duration)) return;

    setIsDragging(true);
    isDraggingRef.current = true;

    // Calculate initial position
    const rect = progressBarRef.current?.getBoundingClientRect();
    if (!rect) return;

    const clickX = e.clientX - rect.left;
    const progress = Math.max(0, Math.min(100, (clickX / rect.width) * 100));

    // Update visual position immediately
    if (progressFillRef.current) {
      progressFillRef.current.style.width = `${progress}%`;
    }
    if (progressDotRef.current) {
      progressDotRef.current.style.left = `${progress}%`;
    }

    // Throttled video seek
    throttledSeek(progress);

    // Prevent text selection during drag
    e.preventDefault();
  };

  // Stable timeline drag handler that uses throttled seek
  const handleTimelineDrag = useCallback((e: MouseEvent) => {
    if (!isDraggingRef.current) return;

    const rect = progressBarRef.current?.getBoundingClientRect();
    if (!rect) return;

    const clickX = e.clientX - rect.left;
    const progress = Math.max(0, Math.min(100, (clickX / rect.width) * 100));

    // Update visual position immediately (not throttled)
    if (progressFillRef.current) {
      progressFillRef.current.style.width = `${progress}%`;
    }
    if (progressDotRef.current) {
      progressDotRef.current.style.left = `${progress}%`;
    }

    // Throttled video seek
    throttledSeek(progress);
  }, [throttledSeek]);

  const handleProgressMouseUp = (e: MouseEvent) => {
    if (!isDraggingRef.current) return;

    setIsDragging(false);
    isDraggingRef.current = false;

    // Sync final position
    syncTimelinePosition();

    // Prevent the click event from firing after drag
    e.stopPropagation();
  };

  // Add document-level mouse event listeners for dragging
  useEffect(() => {
    if (isDragging) {
      document.addEventListener('mousemove', handleTimelineDrag);
      document.addEventListener('mouseup', handleProgressMouseUp);

      return () => {
        document.removeEventListener('mousemove', handleTimelineDrag);
        document.removeEventListener('mouseup', handleProgressMouseUp);
      };
    }
  }, [isDragging, handleTimelineDrag]);


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

  // Handle keyboard shortcuts
  const handleKeyDown = (e: React.KeyboardEvent<HTMLDivElement>) => {
    if (e.code === 'Space') {
      e.preventDefault(); // Prevent page scroll
      handlePlayPause();
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
        className={`relative rounded-md ${isVertical ? 'max-w-64' : 'max-w-4xl'} mx-auto focus:outline-none`}
        tabIndex={0}
        onKeyDown={handleKeyDown}
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

            // Sync timeline position after metadata loads
            syncTimelinePosition();
          }}
          onPause={() => handlePlayStateChange(false)}
          onPlay={() => handlePlayStateChange(true)}
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
            // Sync timeline position after seek completes
            syncTimelinePosition();
          }}
          onTimeUpdate={(e) => {
            const video = e.currentTarget;
            setCurrentTime(video.currentTime);

            // Update timeline if not playing (for manual seeks)
            if (video.paused || video.ended) {
              syncTimelinePosition();
            }

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
        {!isLoading && (
          <>
            {isVertical ? (
              /* Vertical Layout: Single Row Controls Above Timeline */
              <div className="absolute bottom-0 left-0 right-0 p-3 ">
                {/* Control Row */}
                <div className="flex items-center justify-between gap-2 mb-1 hover:bg-black/30 transition-opacity rounded-2xl">
                  {/* Play/Pause Button */}
                  <Button
                    onClick={handlePlayPause}
                    size="icon"
                    variant="ghost"
                    className="hover:bg-white/20 text-white border-0 rounded-full"
                  >
                    {isPlaying ? (
                      <IconPlayerPauseFilled className="size-6" />
                    ) : (
                      <IconPlayerPlayFilled className="size-6" />
                    )}
                  </Button>

                  {/* Time Display */}
                  <div className="flex-1 text-center">
                    <span className="text-white text-sm font-medium">
                      {formatTime(getDisplayTime(currentTime))} / {formatTime(getDisplayDuration())}
                    </span>
                  </div>

                  {/* Picture-in-Picture Button */}
                  <Button
                    onClick={handlePictureInPicture}
                    size="icon"
                    variant={"ghost"}
                    title="Picture-in-Picture"
                    className=" border-0 rounded-full"
                  >
                    <PictureInPicture className="size-4" />
                  </Button>
                </div>

                {/* Progress Bar */}
                <div
                  ref={progressBarRef}
                  className={`relative h-[5px] w-full rounded-full bg-white  ${isDragging ? 'cursor-grabbing' : 'cursor-grab'
                    }`}
                  onClick={handleProgressClick}
                  onMouseDown={handleProgressMouseDown}
                >
                  <div
                    ref={progressFillRef}
                    className="h-full rounded-full bg-white"
                    style={{ width: '0%' }}
                  />
                  <div
                    ref={progressDotRef}
                    className={`absolute top-1/2 left-0 w-3 h-3 bg-white rounded-full shadow-lg ${isDragging ? 'scale-125' : ''
                      } transition-transform`}
                    style={{
                      transform: `translateY(-50%) ${isDragging ? 'scale(1.25)' : ''}`,
                      left: '0%',
                      marginLeft: '-6px',
                    }}
                  />
                </div>
              </div>
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
                    ref={progressBarRef}
                    className={`flex-1 relative h-1 rounded-full bg-neutral-600/40 backdrop-blur-sm rounded-lg py-[5px] ${isDragging ? 'cursor-grabbing' : 'cursor-grab'
                      }`}
                    onClick={handleProgressClick}
                    onMouseDown={handleProgressMouseDown}
                  >
                    <div
                      ref={progressFillRef}
                      className="h-full rounded-full bg-white"
                      style={{ width: '0%' }}
                    />
                    <div
                      ref={progressDotRef}
                      className={`absolute top-1/2 left-0 size-3 bg-white rounded-full shadow-lg ${isDragging ? 'scale-125' : ''
                        } transition-transform`}
                      style={{
                        transform: `translateY(-50%) ${isDragging ? 'scale(1.25)' : ''}`,
                        left: '0%',
                        marginLeft: '-6px',
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
