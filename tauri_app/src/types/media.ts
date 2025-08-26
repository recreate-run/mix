export type MediaOutput = {
  path: string;
  type: 'image' | 'video' | 'audio' | 'gsap_animation';
  title: string;
  description?: string;
  startTime?: number;
  duration?: number;
  config?: any;
  sourceVideo?: string;
};

export interface VideoPlayerProps {
  path: string;
  title: string;
  description?: string;
  startTime?: number;
  duration?: number;
  workingDirectory: string;
}
