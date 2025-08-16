import React from 'react';
import { AbsoluteFill, Sequence } from 'remotion';
import type { RemotionVideoConfig, VideoElement } from '@/types/remotion';

interface TemplateAdapterProps {
  config: RemotionVideoConfig;
}

// Simple element renderer
const ElementRenderer: React.FC<{ element: VideoElement }> = ({ element }) => {
  if (element.type === 'text') {
    return (
      <div style={{ color: 'white', fontSize: '24px', fontWeight: 'bold' }}>
        {element.content || 'Sample Text'}
      </div>
    );
  }
  return null;
};

/**
 * Adapter component that renders video elements
 */
export const TemplateAdapter: React.FC<TemplateAdapterProps> = ({ config }) => {
  return (
    <AbsoluteFill style={{ backgroundColor: '#000' }}>
      {config.elements.map((element, index) => (
        <Sequence
          key={index}
          from={0}
          durationInFrames={config.composition.durationInFrames}
        >
          <ElementRenderer element={element} />
        </Sequence>
      ))}
    </AbsoluteFill>
  );
};