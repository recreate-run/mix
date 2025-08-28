import { useEffect, useState, useRef } from 'react';
import type { MediaOutput } from '@/types/media';
import { VideoPlayer } from './video-player';

interface LazyVideoPlayerProps {
  media: MediaOutput;
  workingDirectory: string;
}

export const LazyVideoPlayer = ({ 
  media, 
  workingDirectory 
}: LazyVideoPlayerProps) => {
  const [isVisible, setIsVisible] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    // Reset visibility state when media changes
    setIsVisible(false);
    
    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsVisible(true);
          observer.disconnect(); // Stop observing once loaded
        }
      },
      { threshold: 0.1 } // Trigger when 10% visible
    );

    if (containerRef.current) {
      observer.observe(containerRef.current);
    }

    return () => observer.disconnect();
  }, [media.path]); // Reset when media path changes

  return (
    <div ref={containerRef} className="min-h-[200px]">
      {isVisible ? (
        <VideoPlayer
          duration={media.duration}
          path={media.path}
          startTime={media.startTime}
          title=""
          workingDirectory={workingDirectory}
        />
      ) : (
        <div className="flex items-center justify-center h-48 bg-stone-700/30 rounded-md">
          <div className="animate-pulse">
            <div className="h-6 w-32 bg-stone-600 rounded mb-2"></div>
            <div className="h-4 w-48 bg-stone-600 rounded"></div>
          </div>
        </div>
      )}
    </div>
  );
};