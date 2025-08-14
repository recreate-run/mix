import React, { useState } from 'react';
import { Player } from '@remotion/player';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
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
    <div className="flex gap-4 remotion-video-preview mb-4">
      <div className="rounded-lg overflow-hidden">
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
            minHeight: '300px'
          }}
          acknowledgeRemotionLicense
        />
      </div>
      
      {/* Simple Animation Controls */}
      <Card className='border-none'>
        <CardHeader>
          <CardTitle className="text-base">Animation Controls</CardTitle>
        </CardHeader>
        
        {firstTextElement && (
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label className="text-sm font-medium">Text Content</Label>
              <Input
                value={firstTextElement.content}
                onChange={(e) => updateTextContent(e.target.value)}
                placeholder="Enter text..."
              />
            </div>
            
            <div className="space-y-2">
              <Label className="text-sm font-medium">Animation Type</Label>
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
              <Label className="text-sm font-medium">
                Duration: {firstTextElement.animation?.duration || 30} frames
              </Label>
              <Input
                type="range"
                min="5"
                max="150"
                value={firstTextElement.animation?.duration || 30}
                onChange={(e) => updateAnimationDuration(Number(e.target.value))}
                className="w-full"
              />
            </div>
          </CardContent>
        )}
      </Card>
    </div>
  );
};