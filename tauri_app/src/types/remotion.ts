// Basic Remotion types for compatibility
export type VideoFormat = 'horizontal' | 'vertical';

export interface VideoConfig {
  format: VideoFormat;
  durationInFrames: number;
  fps: number;
}

export interface VideoElement {
  type: string;
  content: string;
  from?: number;
  durationInFrames?: number;
  layout?: 'top-center' | 'bottom-center';
  style?: {
    fontSize?: number;
    color?: string;
    backgroundColor?: string;
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
