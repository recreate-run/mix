import React from 'react';
import { AbsoluteFill, useCurrentFrame, interpolate, spring, useVideoConfig, Sequence } from 'remotion';

export const CreativeShowcase: React.FC = () => {
  const frame = useCurrentFrame();
  const { fps, durationInFrames } = useVideoConfig();

  // Multiple animated elements
  const titleScale = spring({
    fps,
    frame,
    config: { damping: 200, stiffness: 100 },
  });

  const subtitleOpacity = interpolate(frame, [30, 60], [0, 1], {
    extrapolateRight: 'clamp',
  });

  const circleScale = spring({
    fps,
    frame: frame - 60,
    config: { damping: 150, stiffness: 80 },
  });

  const gradientRotation = interpolate(frame, [0, durationInFrames], [0, 360]);

  const floatingY = Math.sin(frame * 0.1) * 20;

  return (
    <AbsoluteFill
      style={{
        background: `linear-gradient(${gradientRotation}deg, #1a1a2e, #16213e, #0f3460)`,
        justifyContent: 'center',
        alignItems: 'center',
        overflow: 'hidden',
      }}
    >
      {/* Background particles */}
      {[...Array(20)].map((_, i) => {
        const particleDelay = i * 5;
        const particleOpacity = interpolate(
          frame,
          [particleDelay, particleDelay + 30],
          [0, 0.6],
          { extrapolateRight: 'clamp' }
        );
        const particleY = interpolate(
          frame,
          [particleDelay, particleDelay + 120],
          [800, -100],
          { extrapolateLeft: 'clamp' }
        );
        
        return (
          <div
            key={i}
            style={{
              position: 'absolute',
              left: `${10 + i * 4}%`,
              top: particleY,
              width: 4,
              height: 4,
              borderRadius: '50%',
              backgroundColor: '#ffffff',
              opacity: particleOpacity,
            }}
          />
        );
      })}

      {/* Main title */}
      <Sequence from={0} durationInFrames={durationInFrames}>
        <div
          style={{
            position: 'absolute',
            top: '30%',
            fontSize: 80,
            fontWeight: 'bold',
            color: '#ffffff',
            transform: `scale(${titleScale}) translateY(${floatingY}px)`,
            textAlign: 'center',
            textShadow: '0 0 20px rgba(255, 255, 255, 0.5)',
            background: 'linear-gradient(45deg, #ff6b6b, #4ecdc4, #45b7d1)',
            backgroundSize: '200% 200%',
            backgroundClip: 'text',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent',
            animation: `gradient 3s ease infinite`,
          }}
        >
          Creative
          <br />
          Showcase
        </div>
      </Sequence>

      {/* Subtitle */}
      <Sequence from={30} durationInFrames={durationInFrames - 30}>
        <div
          style={{
            position: 'absolute',
            top: '60%',
            fontSize: 28,
            color: '#ffffff',
            opacity: subtitleOpacity,
            textAlign: 'center',
            letterSpacing: '2px',
          }}
        >
          Dynamic Video Generation
        </div>
      </Sequence>

      {/* Animated circle */}
      <Sequence from={60} durationInFrames={durationInFrames - 60}>
        <div
          style={{
            position: 'absolute',
            bottom: '20%',
            left: '50%',
            transform: `translateX(-50%) scale(${circleScale})`,
            width: 120,
            height: 120,
            borderRadius: '50%',
            border: '3px solid rgba(255, 255, 255, 0.3)',
            background: 'rgba(255, 255, 255, 0.1)',
            backdropFilter: 'blur(10px)',
          }}
        />
      </Sequence>

      {/* Corner decorations */}
      <div
        style={{
          position: 'absolute',
          top: 40,
          left: 40,
          width: 60,
          height: 60,
          border: '2px solid rgba(255, 107, 107, 0.6)',
          borderRadius: '8px',
          transform: `rotate(${frame * 2}deg)`,
        }}
      />
      
      <div
        style={{
          position: 'absolute',
          bottom: 40,
          right: 40,
          width: 40,
          height: 40,
          backgroundColor: 'rgba(78, 205, 196, 0.6)',
          borderRadius: '50%',
          transform: `scale(${Math.sin(frame * 0.05) * 0.3 + 1})`,
        }}
      />
    </AbsoluteFill>
  );
};