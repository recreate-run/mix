import React from 'react';
import { AbsoluteFill, Sequence, useCurrentFrame, interpolate, getInputProps } from 'remotion';

export interface VideoElement {
  type: 'text' | 'shape';
  content: string;
  from: number;
  durationInFrames: number;
  position: { x: number; y: number };
  style?: {
    fontSize?: number;
    color?: string;
    backgroundColor?: string;
  };
  animation?: {
    type: 'fadeIn' | 'fadeOut' | 'slideIn' | 'slideOut' | 'typing';
    duration: number;
  };
}

export interface VideoConfig {
  composition: {
    durationInFrames: number;
    fps: number;
    width: number;
    height: number;
  };
  elements: VideoElement[];
}

export const DynamicVideoComposition: React.FC = () => {
  const inputProps = getInputProps() as { config?: VideoConfig };
  const config = inputProps.config;
  
  if (!config) {
    throw new Error('No configuration provided. Video configuration is required to render the composition.');
  }

  if (!config.composition || !config.elements) {
    throw new Error('Invalid configuration structure. Expected config with "composition" and "elements" properties.');
  }

  if (config.elements.length === 0) {
    throw new Error('No elements defined in configuration. At least one element is required to render the video.');
  }

  return (
    <AbsoluteFill style={{ backgroundColor: '#000' }}>
      {config.elements.map((element, index) => (
        <Sequence
          key={index}
          from={element.from}
          durationInFrames={element.durationInFrames}
        >
          <ElementRenderer element={element} />
        </Sequence>
      ))}
    </AbsoluteFill>
  );
};

export const ElementRenderer: React.FC<{ element: VideoElement }> = ({ element }) => {
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
        opacity = interpolate(frame, [element.durationInFrames - element.animation.duration, element.durationInFrames], [1, 0], { extrapolateRight: 'clamp' });
        break;
      case 'slideIn':
        translateX = interpolate(frame, [0, element.animation.duration], [-200, 0], { extrapolateRight: 'clamp' });
        break;
      case 'slideOut':
        translateX = interpolate(frame, [element.durationInFrames - element.animation.duration, element.durationInFrames], [0, 200], { extrapolateRight: 'clamp' });
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
    transform: `translate(calc(-50% + ${element.position.x}px + ${translateX}px), calc(-50% + ${element.position.y}px + ${translateY}px))`,
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