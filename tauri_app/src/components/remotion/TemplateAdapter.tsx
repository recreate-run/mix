import React from 'react';
import { AbsoluteFill, Sequence, useCurrentFrame, interpolate } from 'remotion';
import type { RemotionVideoConfig, VideoElement } from '@/types/remotion';

interface TemplateAdapterProps {
  config: RemotionVideoConfig;
}

// Full-featured element renderer with animations
const ElementRenderer: React.FC<{ element: VideoElement; compositionDuration: number }> = ({ element, compositionDuration }) => {
  const frame = useCurrentFrame();
  
  let opacity = 1;
  let translateX = 0;
  let translateY = 0;
  let displayContent = element.content;
  
  if (element.animation) {
    switch (element.animation.type) {
      case 'fadeIn':
        opacity = interpolate(frame, [0, element.animation.duration], [0, 1], { extrapolateRight: 'clamp' });
        break;
      case 'fadeOut':
        const fadeOutDuration = element.durationInFrames || compositionDuration;
        opacity = interpolate(frame, [fadeOutDuration - element.animation.duration, fadeOutDuration], [1, 0], { extrapolateRight: 'clamp' });
        break;
      case 'slideIn':
        translateX = interpolate(frame, [0, element.animation.duration], [-200, 0], { extrapolateRight: 'clamp' });
        break;
      case 'slideOut':
        const slideOutDuration = element.durationInFrames || compositionDuration;
        translateX = interpolate(frame, [slideOutDuration - element.animation.duration, slideOutDuration], [0, 200], { extrapolateRight: 'clamp' });
        break;
      case 'typing':
        const revealedChars = interpolate(frame, [0, element.animation.duration], [0, element.content.length], { 
          extrapolateRight: 'clamp',
          extrapolateLeft: 'clamp'
        });
        displayContent = element.content.slice(0, Math.floor(revealedChars));
        break;
    }
  }
  
  const style: React.CSSProperties = {
    position: 'absolute',
    left: '50%',
    top: '50%',
    transform: `translate(calc(-50% + ${element.position?.x || 0}px + ${translateX}px), calc(-50% + ${element.position?.y || 0}px + ${translateY}px))`,
    opacity,
    fontSize: element.style?.fontSize || 50,
    color: element.style?.color || '#ffffff',
    backgroundColor: element.style?.backgroundColor || 'transparent',
    padding: element.style?.backgroundColor ? '10px 20px' : '0',
    borderRadius: element.style?.backgroundColor ? '8px' : '0',
  };

  if (element.type === 'text') {
    return <div style={style}>{displayContent}</div>;
  }
  
  return <div style={style}>Shape: {element.content}</div>;
};

/**
 * Adapter component that renders video elements
 */
export const TemplateAdapter: React.FC<TemplateAdapterProps> = ({ config }) => {
  return (
    <AbsoluteFill style={{ backgroundColor: '#000' }}>
      {config.elements.map((element, index) => (
        <Sequence
          key={index}
          from={element.from || 0}
          durationInFrames={element.durationInFrames || config.composition.durationInFrames}
        >
          <ElementRenderer element={element} compositionDuration={config.composition.durationInFrames} />
        </Sequence>
      ))}
    </AbsoluteFill>
  );
};