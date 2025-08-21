// Basic Remotion types for compatibility
export type VideoFormat = 'horizontal' | 'vertical';

export interface VideoConfig {
  format: VideoFormat;
  durationInFrames: number;
  fps: number;
}

export interface VideoElement {
  type: 'text' | 'shape' | 'image' | 'video';
  content: string;
  from?: number;
  durationInFrames?: number;
  layout?: 'top-center' | 'bottom-center';
  style?: {
    fontSize?: number;
    color?: string;
    backgroundColor?: string;
    opacity?: number;
    objectFit?: 'cover' | 'contain' | 'fill' | 'scale-down' | 'none';
  };
  animation?: {
    type: 'fadeIn' | 'fadeOut' | 'slideIn' | 'slideOut' | 'typing' | 'tiktokEntrance';
    duration: number;
  };
  wordTimings?: { word: string; start: number; end: number }[];
  stroke?: { width: number; color: string };
}

export interface RemotionVideoConfig {
  composition: VideoConfig;
  elements: VideoElement[];
}
