import { useEffect, useRef, useState } from 'react';
import { DotLoader } from '../../../components/gsap/dot-loader';

type DotFlowProps = {
  items: {
    title: string;
    frames: number[][];
    duration?: number;
    repeatCount?: number;
  }[];
  isPlaying?: boolean;
};

export const DotFlow = ({ items, isPlaying = true }: DotFlowProps) => {
  const containerRef = useRef<HTMLDivElement>(null);
  const textRef = useRef<HTMLDivElement>(null);
  const [index, setIndex] = useState(0);
  const [textIndex, setTextIndex] = useState(0);
  const [isTransitioning, setIsTransitioning] = useState(false);

  // Handle container width animation with CSS transitions
  useEffect(() => {
    if (!(containerRef.current && textRef.current)) return;

    const newWidth = textRef.current.offsetWidth + 1;
    containerRef.current.style.width = `${newWidth}px`;
  }, [textIndex, items]);

  useEffect(() => {
    setIndex(0);
    setTextIndex(0);
  }, [items]);

  const next = () => {
    if (!containerRef.current) return;

    setIsTransitioning(true);

    // After exit animation completes, update text and start enter animation
    setTimeout(() => {
      setTextIndex((prev) => (prev + 1) % items.length);

      // Small delay before enter animation
      setTimeout(() => {
        setIsTransitioning(false);
      }, 50);
    }, 500); // Match the CSS transition duration

    setIndex((prev) => (prev + 1) % items.length);
  };

  return (
    <div className="flex items-center gap-4 rounded-md px-4">
      <DotLoader
        className="scale-75 gap-px"
        dotClassName="bg-white/15 [&.active]:bg-white size-1"
        duration={items[index]?.duration ?? 150}
        frames={items[index]?.frames ?? []}
        isPlaying={isPlaying}
        onComplete={next}
        repeatCount={items[index]?.repeatCount ?? 1}
      />
      <div
        className="relative transition-all duration-500 ease-out"
        ref={containerRef}
        style={{
          transform: isTransitioning ? 'translateY(20px)' : 'translateY(0px)',
          opacity: isTransitioning ? 0 : 1,
          filter: isTransitioning ? 'blur(8px)' : 'blur(0px)',
        }}
      >
        <div
          className="inline-block whitespace-nowrap font-medium text-lg text-white"
          ref={textRef}
        >
          {items[textIndex]?.title}
        </div>
      </div>
    </div>
  );
};
