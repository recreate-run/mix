import React from 'react';
import { getInputProps } from 'remotion';
// TODO: When converting to separate packages, update this import to use @remotion-shared alias
// or published package dependency. Current cross-package import works via vite.config.ts alias
// but should be formalized during package separation.
import { TemplateAdapter } from '../../../tauri_app/src/components/remotion/TemplateAdapter';
import { VideoFormat } from './constants/videoDimensions';

// Re-export interfaces for compatibility
export interface VideoElement {
  type: 'text' | 'shape' | 'image' | 'video';
  content: string;
  compositionStartFrame: number; // When element appears in composition timeline (was: from)
  compositionDuration: number;   // How long element shows in composition (was: durationInFrames)
  sourceStartFrame?: number;     // Where to start in source video file (video elements only)
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

export interface VideoConfig {
  composition: {
    durationInFrames: number;
    fps: number;
    format: VideoFormat;
  };
  elements: VideoElement[];
}

/**
 * Unified DynamicVideoComposition - now uses TemplateAdapter as single source of truth.
 * This eliminates code duplication and ensures consistent behavior between preview and export.
 */
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

  // Use TemplateAdapter as the single source of truth for rendering
  return <TemplateAdapter config={config} />;
};