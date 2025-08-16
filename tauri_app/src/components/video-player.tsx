import { useState, useEffect, useRef } from 'react';
import { convertFileSrc } from '@tauri-apps/api/core';
import { Button } from '@/components/ui/button';
import { Skeleton } from '@/components/ui/skeleton';
import { Play, Pause, PictureInPicture, RotateCcw } from 'lucide-react';
import type { VideoPlayerProps } from '@/types/media';

export const VideoPlayer = ({ path, title, description, startTime, duration }: VideoPlayerProps) => {
  const [isLoading, setIsLoading] = useState(true);
  const [isPlaying, setIsPlaying] = useState(false);
  const [currentTime, setCurrentTime] = useState(0);
  const videoRef = useRef<HTMLVideoElement>(null);

  useEffect(() => {
    setIsLoading(true);
  }, [path, startTime, duration]);

  // Reset video position when segment changes
  useEffect(() => {
    if (videoRef.current && startTime !== undefined) {
      videoRef.current.currentTime = startTime;
    }
  }, [startTime, duration]);

  // Create URL with media fragment for segments
  const getVideoSrc = () => {
    const baseSrc = convertFileSrc(path);
    
    // For video segments, add media fragment to constrain playback
    if (startTime !== undefined && duration !== undefined) {
      const endTime = startTime + duration;
      return `${baseSrc}#t=${startTime},${endTime}`;
    }
    
    return baseSrc;
  };

  // Get segment info
  const segmentStartTime = startTime || 0;
  const segmentDuration = duration || 0;
  const isSegment = startTime !== undefined && duration !== undefined;

  // Format time for display (segment time, not absolute time)
  const formatTime = (seconds: number) => {
    const minutes = Math.floor(seconds / 60);
    const remainingSeconds = Math.floor(seconds % 60);
    return `${minutes}:${remainingSeconds.toString().padStart(2, '0')}`;
  };

  // Get segment time (0-based from start of segment)
  const getSegmentTime = (videoCurrentTime: number) => {
    if (!isSegment) return videoCurrentTime;
    return Math.max(0, videoCurrentTime - segmentStartTime);
  };

  // Convert segment progress (0-100%) to actual video time
  const segmentProgressToVideoTime = (progress: number) => {
    if (!isSegment) return 0;
    return segmentStartTime + (progress / 100) * segmentDuration;
  };

  // Get current segment progress (0-100%)
  const getSegmentProgress = () => {
    if (!isSegment || segmentDuration === 0) return 0;
    const segmentTime = getSegmentTime(currentTime);
    return Math.min(100, (segmentTime / segmentDuration) * 100);
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
    const startTime = isSegment ? segmentStartTime : 0;
    videoRef.current.currentTime = startTime;
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
            <p className="text-sm text-muted-foreground mt-1">{description}</p>
          )}
        </div>
      )}

      <div className="rounded-md relative max-w-xl">
        {isLoading && (
          <Skeleton className="aspect-video" />
        )}
        <video
          ref={videoRef}
          src={getVideoSrc()}
          className={`aspect-video bg-black rounded-md ${isLoading ? 'hidden' : ''}`}
          preload="auto"
          onLoadedData={() => {
            setIsLoading(false);
          }}
          onPlay={() => setIsPlaying(true)}
          onPause={() => setIsPlaying(false)}
          onTimeUpdate={(e) => {
            const video = e.currentTarget;
            setCurrentTime(video.currentTime);
            
            // Additional safeguard to keep playback within bounds
            if (isSegment) {
              const endTime = segmentStartTime + segmentDuration;
              if (video.currentTime >= endTime) {
                video.pause();
                video.currentTime = endTime;
              }
            }
          }}
          onSeeking={(e) => {
            // Prevent seeking outside the segment range
            if (isSegment) {
              const video = e.currentTarget;
              const endTime = segmentStartTime + segmentDuration;
              
              if (video.currentTime < segmentStartTime) {
                video.currentTime = segmentStartTime;
              } else if (video.currentTime > endTime) {
                video.currentTime = endTime;
              }
            }
          }}
          onError={(e) => {
            e.currentTarget.style.display = 'none';
            const fallback = e.currentTarget.nextElementSibling as HTMLElement;
            if (fallback) fallback.style.display = 'block';
          }}
        >
          Your browser does not support the video tag.
        </video>
        
        {/* Custom Controls */}
        {!isLoading && (
          <div className="absolute bottom-0 left-0 right-0 p-2">
            <div className="flex items-center gap-3">
              {/* Play/Pause Button */}

              <div className='flex items-center'>
              <Button
                variant="ghost" 
                size="sm"
                onClick={handlePlayPause}
              >
                {isPlaying ? <Pause className="size-4" /> : <Play className="size-4" />}
              </Button>
              
              {/* Replay Button */}
              <Button
                variant="ghost" 
                size="sm"
                onClick={handleReplay}
                title="Replay"
              >
                <RotateCcw className="size-4" />
              </Button>

              </div>

              {/* Progress Bar - show for all videos */}
              <div className="flex-1 flex items-center gap-2">
                <span className="text-white text-xs font-mono">
                  {formatTime(isSegment ? getSegmentTime(currentTime) : currentTime)}
                </span>
                
                <div 
                  className="flex-1 h-1 bg-white/30 rounded cursor-pointer relative"
                  onClick={handleProgressClick}
                >
                  <div 
                    className="h-full bg-white rounded"
                    style={{ width: `${isSegment ? getSegmentProgress() : (videoRef.current?.duration ? (currentTime / videoRef.current.duration) * 100 : 0)}%` }}
                  />
                </div>
                
                <span className="text-white text-xs font-mono">
                  {formatTime(isSegment ? segmentDuration : (videoRef.current?.duration || 0))}
                </span>
              </div>
              
              {/* Picture-in-Picture Button */}
              <Button
                variant="ghost" 
                size="sm"
                onClick={handlePictureInPicture}
                title="Picture-in-Picture"
              >
                <PictureInPicture className="size-4" />
              </Button>
            </div>
          </div>
        )}
        
        <div
          className="flex items-center justify-center h-48 bg-stone-700 text-stone-400"
          style={{ display: 'none' }}
        >
          Failed to load video: {path}
        </div>
      </div>
    </div>
  );
};