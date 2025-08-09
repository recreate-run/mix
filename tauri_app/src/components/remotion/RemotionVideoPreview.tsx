import React, { useState } from 'react';
import { Player } from '@remotion/player';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';
import { TemplateAdapter } from './TemplateAdapter';
import type { RemotionVideoConfig } from '@/types/remotion';

interface RemotionVideoPreviewProps {
  config: RemotionVideoConfig;
  sessionId?: string;
}

export const RemotionVideoPreview: React.FC<RemotionVideoPreviewProps> = ({ 
  config, 
  sessionId 
}) => {
  const [editableConfig, setEditableConfig] = useState<RemotionVideoConfig>(config);
  const [showControls, setShowControls] = useState(false);

  // Get the first text element for editing (keep it simple)
  const firstTextElement = editableConfig.elements.find(el => el.type === 'text');
  
  const updateTextContent = (content: string) => {
    setEditableConfig(prev => ({
      ...prev,
      elements: prev.elements.map(el => 
        el.type === 'text' && el === firstTextElement 
          ? { ...el, content }
          : el
      )
    }));
  };

  const updateAnimationType = (animationType: string) => {
    setEditableConfig(prev => ({
      ...prev,
      elements: prev.elements.map(el => 
        el.type === 'text' && el === firstTextElement 
          ? { ...el, animation: el.animation ? { ...el.animation, type: animationType as any } : undefined }
          : el
      )
    }));
  };

  const updateAnimationDuration = (duration: number) => {
    setEditableConfig(prev => ({
      ...prev,
      elements: prev.elements.map(el => 
        el.type === 'text' && el === firstTextElement 
          ? { ...el, animation: el.animation ? { ...el.animation, duration } : undefined }
          : el
      )
    }));
  };

  return (
    <div className="remotion-video-preview space-y-4">
      <div className="rounded-lg overflow-hidden bg-black">
        <Player
          component={TemplateAdapter}
          inputProps={{ config: editableConfig }}
          durationInFrames={editableConfig.composition.durationInFrames}
          fps={editableConfig.composition.fps}
          compositionWidth={editableConfig.composition.width}
          compositionHeight={editableConfig.composition.height}
          controls
          style={{ 
            width: '100%', 
            maxWidth: '600px',
            height: '300px',
            minHeight: '300px'
          }}
          acknowledgeRemotionLicense
        />
      </div>
      
      {/* Simple Animation Controls */}
      <div className="space-y-3 border rounded-lg p-4">
        <div className="flex items-center justify-between">
          <h3 className="font-medium">Animation Controls</h3>
          <Button
            variant="ghost" 
            size="sm"
            onClick={() => setShowControls(!showControls)}
          >
            {showControls ? 'Hide' : 'Show'}
          </Button>
        </div>
        
        {showControls && firstTextElement && (
          <div className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Text Content</label>
              <Input
                value={firstTextElement.content}
                onChange={(e) => updateTextContent(e.target.value)}
                placeholder="Enter text..."
              />
            </div>
            
            <div className="space-y-2">
              <label className="text-sm font-medium">Animation Type</label>
              <Select
                value={firstTextElement.animation?.type || 'fadeIn'}
                onValueChange={updateAnimationType}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="fadeIn">Fade In</SelectItem>
                  <SelectItem value="fadeOut">Fade Out</SelectItem>
                  <SelectItem value="slideIn">Slide In</SelectItem>
                  <SelectItem value="slideOut">Slide Out</SelectItem>
                  <SelectItem value="typing">Typing</SelectItem>
                </SelectContent>
              </Select>
            </div>
            
            <div className="space-y-2">
              <label className="text-sm font-medium">
                Duration: {firstTextElement.animation?.duration || 30} frames
              </label>
              <Input
                type="range"
                min="5"
                max="150"
                value={firstTextElement.animation?.duration || 30}
                onChange={(e) => updateAnimationDuration(Number(e.target.value))}
                className="w-full"
              />
            </div>
          </div>
        )}
      </div>
    </div>
  );
};