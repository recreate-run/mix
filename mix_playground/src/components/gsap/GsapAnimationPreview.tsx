import { Check, Copy, RefreshCw } from 'lucide-react';
import type React from 'react';
import { useEffect, useMemo, useRef, useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { ScrollArea } from '@/components/ui/scroll-area';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Separator } from '@/components/ui/separator';
import { Switch } from '@/components/ui/switch';
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard';
import { useAnimationSchema } from '@/hooks/useAnimationSchema';

interface GsapAnimationPreviewProps {
  config: any; // Accept any configuration format from the endpoint
}

export const GsapAnimationPreview: React.FC<GsapAnimationPreviewProps> = ({
  config: initialConfig,
}) => {
  // Use configuration directly as received from the endpoint
  const [config, setConfig] = useState(initialConfig);
  const [isVisible, setIsVisible] = useState<boolean>(true);
  const [containerNode, setContainerNode] = useState<HTMLDivElement | null>(
    null
  );
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const { isCopied, copyToClipboard } = useCopyToClipboard();

  const animationName = config.url
    ? (() => {
        try {
          const url = new URL(config.url);
          const path = url.pathname.substring(1).split('?')[0];

          // Handle format: 'animations/name/preview'
          if (path.startsWith('animations/')) {
            const parts = path.split('/');
            return parts[1]; // Get the animation name (second part)
          }

          console.error(
            `[GsapAnimationPreview] Unsupported URL format: ${path}`
          );
          return null;
        } catch (error) {
          console.error('[GsapAnimationPreview] Error parsing URL:', error);
          return null;
        }
      })()
    : null;

  // Load animation schema using TanStack Query
  const {
    data: schema,
    isLoading: isLoadingSchema,
    error: schemaError,
  } = useAnimationSchema({
    animationName,
    enabled: !!animationName,
  });

  // Log schema errors for debugging
  if (schemaError) {
    console.error('[GsapAnimationPreview] Schema fetch error:', schemaError);
  }

  // Intersection Observer to track iframe visibility
  useEffect(() => {
    if (!containerNode) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        setIsVisible(entry.isIntersecting);
      },
      {
        threshold: 0.1, // Trigger when 10% of the iframe container is visible
        rootMargin: '50px', // Start loading slightly before it comes into view
      }
    );

    observer.observe(containerNode);

    return () => {
      observer.disconnect();
    };
  }, [containerNode]);

  // Determine container dimensions based on aspect ratio
  const containerDimensions = useMemo(() => {
    const rawAspectRatio = config.aspectRatio || config.aspect || '9/16';

    // Parse aspect ratio - handle both decimal (0.5625) and string ("9/16") formats
    let aspectRatio: number;
    if (typeof rawAspectRatio === 'string' && rawAspectRatio.includes('/')) {
      const [width, height] = rawAspectRatio.split('/').map(Number);
      aspectRatio = width && height ? width / height : 9 / 16;
    } else {
      aspectRatio =
        typeof rawAspectRatio === 'number' ? rawAspectRatio : 9 / 16;
    }

    const isLandscape = aspectRatio > 1;

    return isLandscape
      ? { width: 640, height: 360 } // 16:9 landscape
      : { width: 360, height: 640 }; // 9:16 portrait
  }, [config.aspectRatio, config.aspect]);

  // Use iframe URL with config parameters as URL search params
  const iframeUrl = useMemo(() => {
    if (!config.url) return '';

    const url = new URL(config.url);

    // Add all config parameters as URL search params (excluding 'url' itself)
    Object.entries(config).forEach(([key, value]) => {
      if (key !== 'url' && value !== undefined && value !== null) {
        // Handle nested objects (like style) by JSON stringifying
        if (typeof value === 'object') {
          url.searchParams.set(key, JSON.stringify(value));
        } else {
          url.searchParams.set(key, String(value));
        }
      }
    });

    return url.toString();
  }, [config]);

  // Conditional iframe URL - only load when visible to save resources
  const effectiveIframeUrl = isVisible ? iframeUrl : undefined;

  // Update parameter value - support flexible config structure
  const updateParameter = (key: string, value: any) => {
    setConfig((prev: any) => ({
      ...prev,
      [key]: value,
    }));
  };

  // Render parameter control based on schema
  const renderParameterControl = (param: any, value: any) => {
    const paramKey = param.name;

    switch (param.type) {
      case 'string':
        if (param.options) {
          // Dropdown for string options
          return (
            <Select
              onValueChange={(selectedValue) =>
                updateParameter(paramKey, selectedValue)
              }
              value={value ?? param.default}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder="Select option..." />
              </SelectTrigger>
              <SelectContent>
                {param.options.map((option: string) => (
                  <SelectItem key={option} value={option}>
                    {option}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          );
        }
        // Text input
        return (
          <Input
            onChange={(e) => updateParameter(paramKey, e.target.value)}
            placeholder={param.description}
            value={value ?? param.default}
          />
        );

      case 'number':
        return (
          <div className="space-y-2">
            <Input
              max={param.max}
              min={param.min}
              onChange={(e) =>
                updateParameter(paramKey, Number(e.target.value))
              }
              type="number"
              value={value ?? param.default}
            />
            {param.min !== undefined && param.max !== undefined && (
              <Input
                className="w-full"
                max={param.max}
                min={param.min}
                onChange={(e) =>
                  updateParameter(paramKey, Number(e.target.value))
                }
                type="range"
                value={value ?? param.default}
              />
            )}
          </div>
        );

      case 'boolean':
        return (
          <Switch
            checked={value ?? param.default}
            onCheckedChange={(checked) => updateParameter(paramKey, checked)}
          />
        );

      case 'color':
        return (
          <div className="flex items-center gap-2">
            <Input
              className="w-16"
              onChange={(e) => updateParameter(paramKey, e.target.value)}
              type="color"
              value={value ?? param.default}
            />
            <Input
              className="flex-1"
              onChange={(e) => updateParameter(paramKey, e.target.value)}
              type="text"
              value={value ?? param.default}
            />
          </div>
        );

      default:
        return (
          <Input
            onChange={(e) => updateParameter(paramKey, e.target.value)}
            value={String(value ?? param.default)}
          />
        );
    }
  };

  return (
    <div className="gsap-animation-preview my-4 flex gap-8">
      {/* Animation Preview */}
      <div className="flex-1">
        <div
          className="relative rounded-lg"
          ref={setContainerNode}
          style={{
            width: `${containerDimensions.width}px`,
            height: `${containerDimensions.height}px`,
          }}
        >
          <iframe
            className="h-full w-full rounded-lg border bg-transparent"
            ref={iframeRef}
            sandbox="allow-scripts allow-same-origin"
            src={effectiveIframeUrl}
            style={{ transform: 'scale(1)', transformOrigin: 'top left' }}
            title="GSAP Animation Preview"
          />
        </div>
      </div>

      {/* Controls Panel */}
      <div className="w-80 rounded-xl bg-card p-4">
        <div className="mb-4 flex items-center justify-between">
          <h4 className="font-medium text-md">Animation Controls</h4>
        </div>

        {isLoadingSchema ? (
          <div className="flex items-center justify-center py-8">
            <RefreshCw className="size-6 animate-spin" />
            <span className="ml-2">Loading schema...</span>
          </div>
        ) : (
          <ScrollArea className="h-[600px] pr-4">
            <div className="space-y-4">
              {config.duration !== undefined && (
                <div className="space-y-2">
                  <Label>Duration (ms): {config.duration}</Label>
                  <Input
                    className="w-full"
                    max="10000"
                    min="1000"
                    onChange={(e) =>
                      updateParameter('duration', Number(e.target.value))
                    }
                    type="range"
                    value={config.duration}
                  />
                </div>
              )}

              {/* Add separator if both common parameters and schema parameters exist */}
              {(config.text ||
                config.overlayText ||
                config.displayText ||
                config.textColor ||
                config.style?.color ||
                config.duration !== undefined) &&
                schema?.parameters?.length && <Separator />}

              {/* Schema Parameters */}
              {schema?.parameters?.length
                ? schema.parameters.map((param) => (
                    <div className="space-y-2" key={param.name}>
                      <Label className="font-medium text-sm">
                        {param.name}
                        {param.description && (
                          <span className="ml-2 text-muted-foreground text-xs">
                            {param.description}
                          </span>
                        )}
                      </Label>
                      {renderParameterControl(param, config[param.name])}
                    </div>
                  ))
                : !(
                    config.text ||
                    config.overlayText ||
                    config.displayText ||
                    config.textColor ||
                    config.style?.color ||
                    config.duration !== undefined
                  ) && (
                    <div className="py-8 text-center text-muted-foreground">
                      <p>No parameters available</p>
                      <p className="mt-2 text-xs">
                        Animation will use default settings
                      </p>
                    </div>
                  )}

              {/* URL Preview Section */}
              {iframeUrl && (
                <>
                  <Separator className="my-4" />
                  <div className="space-y-2">
                    <Label className="font-medium text-sm">Current URL</Label>
                    <div className="flex items-center gap-2">
                      <Input
                        className="flex-1 font-mono"
                        readOnly
                        style={{ fontSize: '12px' }}
                        value={iframeUrl}
                      />
                      <Button
                        className="shrink-0"
                        onClick={() => copyToClipboard(iframeUrl)}
                        size="sm"
                        variant="outline"
                      >
                        {isCopied ? (
                          <Check className="h-3 w-3" />
                        ) : (
                          <Copy className="h-3 w-3" />
                        )}
                      </Button>
                    </div>
                  </div>
                </>
              )}
            </div>
          </ScrollArea>
        )}
      </div>
    </div>
  );
};
