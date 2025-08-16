// Basic Remotion types for compatibility
export interface VideoConfig {
  width: number;
  height: number;
  durationInFrames: number;
  fps: number;
}

export interface VideoElement {
  type: string;
  content?: string;
  props: Record<string, any>;
  animation?: {
    type: string;
    duration: number;
  };
}

export interface RemotionVideoConfig {
  composition: {
    width: number;
    height: number;
    durationInFrames: number;
    fps: number;
  };
  elements: VideoElement[];
}