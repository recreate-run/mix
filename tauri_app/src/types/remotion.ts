// Basic Remotion types for compatibility
export interface VideoConfig {
  width: number;
  height: number;
  durationInFrames: number;
  fps: number;
}

export interface VideoElement {
  type: string;
  content: string;
  from?: number;
  durationInFrames?: number;
  position?: { x: number; y: number };
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
