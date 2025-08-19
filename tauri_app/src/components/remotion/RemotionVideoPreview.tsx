import { Player } from '@remotion/player';
import type React from 'react';
import { useState } from 'react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import type { RemotionVideoConfig } from '@/types/remotion';
import { TemplateAdapter } from './TemplateAdapter';

interface RemotionVideoPreviewProps {
  config: RemotionVideoConfig;
  sessionId?: string;
}

export const RemotionVideoPreview: React.FC<RemotionVideoPreviewProps> = ({
  config,
}) => {
  const [editableConfig, setEditableConfig] =
    useState<RemotionVideoConfig>(config);

  // Get the first text element for editing (keep it simple)
  const firstTextElement = editableConfig.elements.find(
    (el) => el.type === 'text'
  );

  const updateTextContent = (content: string) => {
    setEditableConfig((prev) => ({
      ...prev,
      elements: prev.elements.map((el) =>
        el.type === 'text' && el === firstTextElement ? { ...el, content } : el
      ),
    }));
  };

  const updateAnimationType = (animationType: string) => {
    setEditableConfig((prev) => ({
      ...prev,
      elements: prev.elements.map((el) =>
        el.type === 'text' && el === firstTextElement
          ? {
              ...el,
              animation: el.animation
                ? { ...el.animation, type: animationType as any }
                : undefined,
            }
          : el
      ),
    }));
  };

  const updateAnimationDuration = (duration: number) => {
    setEditableConfig((prev) => ({
      ...prev,
      elements: prev.elements.map((el) =>
        el.type === 'text' && el === firstTextElement
          ? {
              ...el,
              animation: el.animation
                ? { ...el.animation, duration }
                : undefined,
            }
          : el
      ),
    }));
  };

  return (
    <div className="remotion-video-preview mb-4 flex gap-4">
      <div className="overflow-hidden rounded-lg">
        <Player
          acknowledgeRemotionLicense
          component={TemplateAdapter}
          compositionHeight={editableConfig.composition.height}
          compositionWidth={editableConfig.composition.width}
          controls
          durationInFrames={editableConfig.composition.durationInFrames}
          fps={editableConfig.composition.fps}
          inputProps={{ config: editableConfig }}
          style={{
            width: '100%',
            maxWidth: '600px',
            minHeight: '300px',
          }}
        />
      </div>

      {/* Simple Animation Controls */}
      <Card className="border-none">
        <CardHeader>
          <CardTitle className="text-base">Animation Controls</CardTitle>
        </CardHeader>

        {firstTextElement && (
          <CardContent className="space-y-4">
            <div className="space-y-2">
              <Label className="font-medium text-sm">Text Content</Label>
              <Input
                onChange={(e) => updateTextContent(e.target.value)}
                placeholder="Enter text..."
                value={firstTextElement.content}
              />
            </div>

            <div className="space-y-2">
              <Label className="font-medium text-sm">Animation Type</Label>
              <Select
                onValueChange={updateAnimationType}
                value={firstTextElement.animation?.type || 'fadeIn'}
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
              <Label className="font-medium text-sm">
                Duration: {firstTextElement.animation?.duration || 30} frames
              </Label>
              <Input
                className="w-full"
                max="150"
                min="5"
                onChange={(e) =>
                  updateAnimationDuration(Number(e.target.value))
                }
                type="range"
                value={firstTextElement.animation?.duration || 30}
              />
            </div>
          </CardContent>
        )}
      </Card>
    </div>
  );
};
