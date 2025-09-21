import { useGSAP } from '@gsap/react';
import gsap from 'gsap';
import { useEffect, useRef, useState } from 'react';

import { DotLoader } from './dot-loader';

export type DotFlowProps = {
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

  const { contextSafe } = useGSAP();

  useEffect(() => {
    if (!(containerRef.current && textRef.current)) return;

    const newWidth = textRef.current.offsetWidth + 1;

    gsap.to(containerRef.current, {
      width: newWidth,
      duration: 0.5,
      ease: 'power2.out',
    });
  }, [textIndex, items]);

  useEffect(() => {
    setIndex(0);
    setTextIndex(0);
  }, [items]);

  const next = contextSafe(() => {
    const el = containerRef.current;
    if (!el) return;
    gsap.to(el, {
      y: 20,
      opacity: 0,
      filter: 'blur(8px)',
      duration: 0.5,
      ease: 'power2.in',
      onComplete: () => {
        setTextIndex((prev) => (prev + 1) % items.length);
        gsap.fromTo(
          el,
          { y: -20, opacity: 0, filter: 'blur(4px)' },
          {
            y: 0,
            opacity: 1,
            filter: 'blur(0px)',
            duration: 0.7,
            ease: 'power2.out',
          }
        );
      },
    });

    setIndex((prev) => (prev + 1) % items.length);
  });

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
      <div className="relative" ref={containerRef}>
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
