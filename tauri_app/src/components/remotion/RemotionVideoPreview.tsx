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
import type { RemotionVideoConfig, VideoFormat } from '@/types/remotion';
import { TemplateAdapter } from './TemplateAdapter';
import { getDimensionsForFormat } from '../../../../packages/remotion_template/src/constants/videoDimensions';

interface RemotionVideoPreviewProps {
  config: RemotionVideoConfig;
  sessionId?: string;
}

export const RemotionVideoPreview: React.FC<RemotionVideoPreviewProps> = ({
  config,
}) => {
  // Helper function to safely revoke blob URLs
  const revokeBlobUrl = (url: string) => {
    if (url?.startsWith('blob:')) {
      URL.revokeObjectURL(url);
    }
  };

  const [editableConfig, setEditableConfig] =
    useState<RemotionVideoConfig>(config);

  // Calculate dimensions based on format (fallback to horizontal if format is missing)
  const format = editableConfig.composition.format || 'horizontal';
  const dimensions = getDimensionsForFormat(format);


  // Get the first text element for editing (keep it simple)
  const firstTextElement = editableConfig.elements.find(
    (el) => el.type === 'text'
  );

  // Find background elements
  const backgroundElements = editableConfig.elements.filter(
    (el) => el.type === 'image' || el.type === 'video'
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
                  strokeType === 'tiktokBlack' ? { width: 20, color: '#000000' } :
                    strokeType === 'tiktokWhite' ? { width: 20, color: '#ffffff' } :
                      strokeType === 'tiktokNeonPink' ? { width: 15, color: '#ff1493' } :
                        strokeType === 'tiktokNeonCyan' ? { width: 15, color: '#00ffff' } :
                          el.stroke, // fallback to current stroke
          }
          : el
      ),
    }));
  };

  // Helper function to get current stroke type
  const getCurrentStrokeType = () => {
    if (!firstTextElement?.stroke) return 'none';

    const { width, color } = firstTextElement.stroke;

    // Detect specific TikTok styles by width and color
    if (width === 20 && color === '#ffffff') return 'tiktokWhite';
    if (width === 15 && color === '#ff1493') return 'tiktokNeonPink';
    if (width === 15 && color === '#00ffff') return 'tiktokNeonCyan';
    if (width === 20 && color === '#000000') return 'tiktokBlack';
    if (width <= 4) return 'normal';

    // Default fallback
    return 'tiktokBlack';
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
              backgroundColor: enabled ? '#8B0000' : 'transparent',
            },
          }
          : el
      ),
    }));
  };

  const updateTextPosition = (layout: string) => {
    setEditableConfig((prev) => ({
      ...prev,
      elements: prev.elements.map((el) =>
        el.type === 'text' && el === firstTextElement
          ? { ...el, layout: layout as 'top-center' | 'bottom-center' }
          : el
      ),
    }));
  };

  // Helper function to get current position
  const getCurrentPosition = () => {
    return firstTextElement?.layout || 'top-center';
  };

  const updateFormat = (format: VideoFormat) => {
    setEditableConfig((prev) => ({
      ...prev,
      composition: {
        ...prev.composition,
        format,
      },
    }));
  };

  // Background management functions
  const addBackgroundElement = (type: 'image' | 'video', src: string) => {
    // Revoke existing background blob URL before adding new one
    if (backgroundElements.length > 0) {
      revokeBlobUrl(backgroundElements[0].content);
    }

    setEditableConfig((prev) => ({
      ...prev,
      elements: [
        {
          type,
          content: src,
          from: 0,
          durationInFrames: prev.composition.durationInFrames
        },
        ...prev.elements.filter(el => el.type !== 'image' && el.type !== 'video') // Remove existing backgrounds, keep text/shape
      ]
    }));
  };

  const removeBackground = () => {
    // Revoke blob URL before removing
    if (backgroundElements.length > 0) {
      revokeBlobUrl(backgroundElements[0].content);
    }

    setEditableConfig((prev) => ({
      ...prev,
      elements: prev.elements.filter((el) => el.type !== 'image' && el.type !== 'video')
    }));
  };


  const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    const fileURL = URL.createObjectURL(file);
    const type = file.type.startsWith('video/') ? 'video' : 'image';
    addBackgroundElement(type, fileURL);
  };

  return (
    <div className="remotion-video-preview my-4 mx-8 flex gap-8">
      <div className="overflow-hidden">
        <Player
          acknowledgeRemotionLicense
          component={TemplateAdapter}
          compositionHeight={dimensions.height}
          compositionWidth={dimensions.width}
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
        <TabsList className="w-full">
          {/* <TabsTrigger value="background">Background</TabsTrigger> */}
          <TabsTrigger value="format">Format</TabsTrigger>
          <TabsTrigger value="animation">Animation</TabsTrigger>
          <TabsTrigger value="layout">Layout</TabsTrigger>
        </TabsList>

        <TabsContent alue="background" className="space-y-4 mt-4">
          <div className="space-y-3">
            <h4 className="text-sm font-medium text-foreground">Background Media</h4>

            <input
              type="file"
              accept="image/*,video/*"
              onChange={handleFileUpload}
              className="hidden"
              id="background-upload"
            />
            <label
              htmlFor="background-upload"
              className="flex items-center justify-center w-full h-32 border-2 border-dashed border-gray-300 rounded-lg cursor-pointer hover:border-gray-400 transition-colors"
            >
              <div className="text-center">
                <div className="text-sm text-gray-600">
                  {backgroundElements.length > 0 ? 'Replace Background' : 'Upload Image or Video'}
                </div>
                <div className="text-xs text-gray-400 mt-1">
                  Click to browse files
                </div>
              </div>
            </label>

            {/* Show current background info */}
            {backgroundElements.length > 0 && (
              <div className="space-y-3">
                <div className="p-3 bg-gray-50 rounded-lg">
                  <div className="flex justify-between items-center">
                    <span className="text-sm font-medium">
                      {backgroundElements[0].type === 'video' ? '🎥' : '🖼️'} {backgroundElements[0].type.charAt(0).toUpperCase() + backgroundElements[0].type.slice(1)} Background
                    </span>
                    <button
                      onClick={removeBackground}
                      className="text-red-500 hover:text-red-700 text-sm"
                    >
                      Remove
                    </button>
                  </div>
                </div>

              </div>
            )}
          </div>
        </TabsContent>

        <TabsContent value="format" className="space-y-6 my-2">
          {/* Text Section */}
          <div className="space-y-3">
            <h4 className="text-sm font-medium text-foreground">Text Content</h4>
            <Input
              onChange={(e) => updateTextContent(e.target.value)}
              placeholder="Enter text..."
              value={firstTextElement.content}
            />
          </div>


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
                  <SelectItem value="tiktokBlack">TikTok Black</SelectItem>
                  <SelectItem value="tiktokWhite">TikTok White</SelectItem>
                  <SelectItem value="tiktokNeonPink">TikTok Neon Pink</SelectItem>
                  <SelectItem value="tiktokNeonCyan">TikTok Neon Cyan</SelectItem>
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

          <div className="flex items-center gap-4">
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

        <TabsContent value="layout" className="space-y-4 mt-4">
          {/* Video Format Section */}
          <div className="space-y-3">
            <h4 className="text-sm font-medium text-foreground">Video Format</h4>
            <Select
              onValueChange={(value) => updateFormat(value as VideoFormat)}
              value={format}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="horizontal">Horizontal (1920×1080)</SelectItem>
                <SelectItem value="vertical">Vertical (1080×1920)</SelectItem>
              </SelectContent>
            </Select>
            <div className="text-xs text-muted-foreground">
              {format === 'horizontal'
                ? 'Standard landscape format for YouTube, presentations, and general use'
                : 'Portrait format optimized for TikTok, Instagram Stories, and mobile content'
              }
            </div>
          </div>

          <Separator />

          <div className="space-y-3">
            <h4 className="text-sm font-medium text-foreground">Text Position</h4>

            <Select
              onValueChange={updateTextPosition}
              value={getCurrentPosition()}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="top-center">Top Center</SelectItem>
                <SelectItem value="bottom-center">Bottom Center</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </TabsContent>
      </Tabs>
    </div>
  );
};
