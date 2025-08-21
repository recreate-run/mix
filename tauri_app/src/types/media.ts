export type MediaOutput = {
  path: string;
  type: 'image' | 'video' | 'audio' | 'remotion_title';
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
