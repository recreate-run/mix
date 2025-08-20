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
import { Switch } from '@/components/ui/switch';
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs';
import { Separator } from '@/components/ui/separator';
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

  const updateStroke = (strokeWidth: number, strokeColor: string) => {
    setEditableConfig((prev) => ({
      ...prev,
      elements: prev.elements.map((el) =>
        el.type === 'text' && el === firstTextElement
          ? {
            ...el,
            stroke: { width: strokeWidth, color: strokeColor },
          }
          : el
      ),
    }));
  };


  const updateStrokeType = (strokeType: string) => {
    setEditableConfig((prev) => ({
      ...prev,
      elements: prev.elements.map((el) =>
        el.type === 'text' && el === firstTextElement
          ? {
            ...el,
            stroke:
              strokeType === 'none' ? undefined :
                strokeType === 'normal' ? { width: 2, color: '#000000' } :
                  strokeType === 'tiktok' ? { width: 20, color: '#000000' } :
                    el.stroke, // fallback to current stroke
          }
          : el
      ),
    }));
  };

  // Helper function to get current stroke type
  const getCurrentStrokeType = () => {
    if (!firstTextElement?.stroke) return 'none';
    if (firstTextElement.stroke.width <= 4) return 'normal';
    return 'tiktok';
  };

  const updateBackgroundColor = (color: string) => {
    setEditableConfig((prev) => ({
      ...prev,
      elements: prev.elements.map((el) =>
        el.type === 'text' && el === firstTextElement
          ? {
            ...el,
            style: {
              ...el.style,
              backgroundColor: color,
            },
          }
          : el
      ),
    }));
  };


  const toggleBackground = (enabled: boolean) => {
    setEditableConfig((prev) => ({
      ...prev,
      elements: prev.elements.map((el) =>
        el.type === 'text' && el === firstTextElement
          ? {
            ...el,
            style: {
              ...el.style,
              backgroundColor: enabled ? '#000000' : 'transparent',
            },
          }
          : el
      ),
    }));
  };

  return (
    <div className="remotion-video-preview my-4 mx-8 flex gap-8">
      <div className="overflow-hidden">
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
            minHeight: '400px',
          }}
          className='rounded-lg'
        />
      </div>

      {/* Organized Settings Panel */}
      <Tabs defaultValue="format" className="bg-card w-80 p-4 rounded-xl">
        <TabsList className="grid w-full grid-cols-2">
          <TabsTrigger value="format">Format</TabsTrigger>
          <TabsTrigger value="animation">Animation</TabsTrigger>
        </TabsList>

        <TabsContent value="format" className="space-y-6 my-2">
          {/* Text Section */}

          <Input
            onChange={(e) => updateTextContent(e.target.value)}
            placeholder="Enter text..."
            value={firstTextElement.content}
          />


          <Separator />

          {/* Stroke Section */}
          <div className="space-y-3">
            <h4 className="text-sm font-medium text-foreground">Stroke</h4>

            <div className="flex items-center gap-2">
              <Label>Type:</Label>
              <Select
                onValueChange={updateStrokeType}
                value={getCurrentStrokeType()}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="none">None</SelectItem>
                  <SelectItem value="normal">Normal</SelectItem>
                  <SelectItem value="tiktok">TikTok Style</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {/* Show width/color controls when stroke is active */}
            {firstTextElement.stroke && (
              <div className="grid grid-cols-2">
                <div className="flex items-center gap-2">
                  <Label>Width:</Label>
                  <Input
                    className="w-16"
                    max="30"
                    min="1"
                    onChange={(e) =>
                      updateStroke(Number(e.target.value), firstTextElement.stroke?.color || '#000000')
                    }
                    type="number"
                    value={firstTextElement.stroke.width}
                  />
                </div>

                <div className="flex items-center gap-2">
                  <Label>Color:</Label>
                  <Input
                    className="w-16"
                    onChange={(e) =>
                      updateStroke(firstTextElement.stroke?.width || 20, e.target.value)
                    }
                    type="color"
                    value={firstTextElement.stroke.color}
                  />
                </div>
              </div>
            )}
          </div>

          <Separator />

          {/* Background Section */}

          <div className="space-y-3">

            <h4 className="text-sm font-medium text-foreground">Background</h4>
            <div className="grid grid-cols-2">

              <div className="flex items-center gap-2">
                <h4 className="text-sm font-medium text-foreground">Enable</h4>
                <Switch
                  checked={!!firstTextElement.style?.backgroundColor &&
                    firstTextElement.style.backgroundColor !== 'transparent'}
                  onCheckedChange={toggleBackground}
                />
              </div>

              {firstTextElement.style?.backgroundColor &&
                firstTextElement.style.backgroundColor !== 'transparent' && (
                  <div className="flex items-center gap-2">
                    <Label>Color:</Label>
                    <Input
                      type="color"
                      value={firstTextElement.style.backgroundColor}
                      onChange={(e) => updateBackgroundColor(e.target.value)}
                      className="w-16"
                    />
                  </div>
                )}
            </div>

          </div>


        </TabsContent>

        <TabsContent value="animation" className="space-y-4 mt-4">
          <div className="flex gap-4">
            <Label>Animation Type</Label>
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
                <SelectItem value="tiktokEntrance">TikTok Entrance</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label >
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
        </TabsContent>
      </Tabs>
    </div>
  );
};
