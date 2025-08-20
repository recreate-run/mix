import type React from 'react';
import { AbsoluteFill, interpolate, Sequence, useCurrentFrame, spring, useVideoConfig } from 'remotion';
import type { RemotionVideoConfig, VideoElement } from '@/types/remotion';

interface TemplateAdapterProps {
  config: RemotionVideoConfig;
}

// Full-featured element renderer with animations
const ElementRenderer: React.FC<{
  element: VideoElement;
  compositionDuration: number;
}> = ({ element, compositionDuration }) => {
  const frame = useCurrentFrame();
  const { fps } = useVideoConfig();

  let opacity = 1;
  let translateX = 0;
  let translateY = 0;
  let scaleValue = 1;
  let displayContent = element.content;

  if (element.animation) {
    switch (element.animation.type) {
      case 'fadeIn':
        opacity = interpolate(frame, [0, element.animation.duration], [0, 1], {
          extrapolateRight: 'clamp',
        });
        break;
      case 'fadeOut': {
        const fadeOutDuration = element.durationInFrames || compositionDuration;
        opacity = interpolate(
          frame,
          [fadeOutDuration - element.animation.duration, fadeOutDuration],
          [1, 0],
          { extrapolateRight: 'clamp' }
        );
        break;
      }
      case 'slideIn':
        translateX = interpolate(
          frame,
          [0, element.animation.duration],
          [-200, 0],
          { extrapolateRight: 'clamp' }
        );
        break;
      case 'slideOut': {
        const slideOutDuration =
          element.durationInFrames || compositionDuration;
        translateX = interpolate(
          frame,
          [slideOutDuration - element.animation.duration, slideOutDuration],
          [0, 200],
          { extrapolateRight: 'clamp' }
        );
        break;
      }
      case 'typing': {
        const revealedChars = interpolate(
          frame,
          [0, element.animation.duration],
          [0, element.content.length],
          {
            extrapolateRight: 'clamp',
            extrapolateLeft: 'clamp',
          }
        );
        displayContent = element.content.slice(0, Math.floor(revealedChars));
        break;
      }
      case 'tiktokEntrance': {
        const springValue = spring({
          frame,
          fps,
          config: { damping: 200 },
          durationInFrames: element.animation.duration
        });
        opacity = springValue;
        translateY = interpolate(springValue, [0, 1], [50, 0]);
        scaleValue = interpolate(springValue, [0, 1], [0.8, 1]);
        break;
      }
    }
  }

  const hasBackground = element.style?.backgroundColor && element.style.backgroundColor !== 'transparent';
  
  const style: React.CSSProperties = {
    position: 'absolute',
    left: '50%',
    top: '50%',
    transform: `translate(calc(-50% + ${element.position?.x || 0}px + ${translateX}px), calc(-50% + ${element.position?.y || 0}px + ${translateY}px)) scale(${scaleValue})`,
    opacity,
    fontSize: element.style?.fontSize || 50,
    color: element.style?.color || '#ffffff',
    backgroundColor: element.style?.backgroundColor || 'transparent',
    padding: hasBackground ? '16px 24px' : '0',
    borderRadius: hasBackground ? '60px' : '0',
    display: hasBackground ? 'flex' : 'block',
    alignItems: hasBackground ? 'center' : 'flex-start',
    justifyContent: hasBackground ? 'center' : 'flex-start',
    textAlign: 'center',
    minHeight: hasBackground ? '60px' : 'auto',
    WebkitTextStroke: element.stroke ? `${element.stroke.width}px ${element.stroke.color}` : undefined,
    paintOrder: element.stroke ? 'stroke' : undefined,
    textTransform: element.stroke ? 'uppercase' : undefined,
  };

  if (element.type === 'text') {
    // Handle word-level highlighting for TikTok-style captions
    if (element.wordTimings && element.wordTimings.length > 0) {
      return (
        <div style={style}>
          {element.wordTimings.map((word, i) => {
            const isActive = frame >= word.start && frame < word.end;
            return (
              <span
                key={i}
                style={{
                  color: isActive ? '#39E508' : style.color,
                  display: 'inline',
                  whiteSpace: 'pre',
                }}
              >
                {word.word}
              </span>
            );
          })}
        </div>
      );
    }
    
    // Standard text rendering
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
          durationInFrames={
            element.durationInFrames || config.composition.durationInFrames
          }
          from={element.from || 0}
          key={index}
        >
          <ElementRenderer
            compositionDuration={config.composition.durationInFrames}
            element={element}
          />
        </Sequence>
      ))}
    </AbsoluteFill>
  );
};
