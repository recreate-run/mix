import React from 'react';
import { AbsoluteFill, Sequence } from 'remotion';
import { VideoElement, ElementRenderer } from '@remotion-shared/DynamicVideoComposition';
import type { RemotionVideoConfig } from '@/types/remotion';

interface TemplateAdapterProps {
  config: RemotionVideoConfig;
}

/**
 * Adapter component that reuses the ElementRenderer from template
 * but accepts config as props instead of getInputProps()
 */

export const TemplateAdapter: React.FC<TemplateAdapterProps> = ({ config }) => {
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