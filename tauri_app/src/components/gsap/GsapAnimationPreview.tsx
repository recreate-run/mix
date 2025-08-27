import type React from 'react';
import { useState, useEffect, useMemo, useRef } from 'react';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Switch } from '@/components/ui/switch';
import { Button } from '@/components/ui/button';
import { Separator } from '@/components/ui/separator';
import { ScrollArea } from '@/components/ui/scroll-area';
import { Copy, Check, RefreshCw } from 'lucide-react';
import {
  AnimationSchema,
  fetchAnimationSchema
} from '@/utils/gsapApi';
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard';

interface GsapAnimationPreviewProps {
  config: any; // Accept any configuration format from the endpoint
}


export const GsapAnimationPreview: React.FC<GsapAnimationPreviewProps> = ({
  config: initialConfig
}) => {
  // Use configuration directly as received from the endpoint
  const [config, setConfig] = useState(initialConfig);
  const [schema, setSchema] = useState<AnimationSchema | null>(null);
  const [isLoadingSchema, setIsLoadingSchema] = useState<boolean>(false);
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const { isCopied, copyToClipboard } = useCopyToClipboard();

  // Extract base server URL and animation name from config.url
  const baseServerUrl = config.url ? new URL(config.url).origin : null;
  const animationName = config.url ? new URL(config.url).pathname.substring(1).split('?')[0].replace('gsap_animations/', '') : null;


  // Load animation schema
  useEffect(() => {
    console.log(`[GsapAnimationPreview] Config URL: ${config.url}`);
    console.log(`[GsapAnimationPreview] Extracted animation name: ${animationName}`);
    console.log(`[GsapAnimationPreview] Base server URL: ${baseServerUrl}`);

    if (!animationName || !baseServerUrl) {
      console.log(`[GsapAnimationPreview] Missing animationName or baseServerUrl, skipping schema fetch`);
      return;
    }

    setIsLoadingSchema(true);
    fetchAnimationSchema(animationName, baseServerUrl)
      .then((fetchedSchema) => {
        console.log(`[GsapAnimationPreview] Schema loaded:`, fetchedSchema);
        setSchema(fetchedSchema);
      })
      .finally(() => {
        setIsLoadingSchema(false);
      });
  }, [animationName, baseServerUrl, config.url]);




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

  // Update parameter value - support flexible config structure
  const updateParameter = (key: string, value: any) => {
    setConfig(prev => ({
      ...prev,
      [key]: value
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
            <select
              value={value ?? param.default}
              onChange={(e) => updateParameter(paramKey, e.target.value)}
              className="w-full p-2 border rounded"
            >
              {param.options.map((option: string) => (
                <option key={option} value={option}>{option}</option>
              ))}
            </select>
          );
        } else {
          // Text input
          return (
            <Input
              value={value ?? param.default}
              onChange={(e) => updateParameter(paramKey, e.target.value)}
              placeholder={param.description}
            />
          );
        }

      case 'number':
        return (
          <div className="space-y-2">
            <Input
              type="number"
              min={param.min}
              max={param.max}
              value={value ?? param.default}
              onChange={(e) => updateParameter(paramKey, Number(e.target.value))}
            />
            {(param.min !== undefined && param.max !== undefined) && (
              <Input
                type="range"
                min={param.min}
                max={param.max}
                value={value ?? param.default}
                onChange={(e) => updateParameter(paramKey, Number(e.target.value))}
                className="w-full"
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
              type="color"
              value={value ?? param.default}
              onChange={(e) => updateParameter(paramKey, e.target.value)}
              className="w-16"
            />
            <Input
              type="text"
              value={value ?? param.default}
              onChange={(e) => updateParameter(paramKey, e.target.value)}
              className="flex-1"
            />
          </div>
        );

      default:
        return (
          <Input
            value={String(value ?? param.default)}
            onChange={(e) => updateParameter(paramKey, e.target.value)}
          />
        );
    }
  };


  return (
    <div className="gsap-animation-preview my-4 flex gap-8">
      {/* Animation Preview */}
      <div className="flex-1 max-w-2xl">
        <div className="relative w-[360px] h-[640px]">
          <iframe
            ref={iframeRef}
            src={iframeUrl}
            className="w-full h-full bg-transparent rounded-lg border"
            style={{ transform: 'scale(1)', transformOrigin: 'top left' }}
            title="GSAP Animation Preview"
            sandbox="allow-scripts allow-same-origin"
          />
        </div>
      </div>

      {/* Controls Panel */}
      <div className="bg-card w-80 p-4 rounded-xl">
        <div className="flex items-center justify-between mb-4">
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
              {(config.textColor || config.style?.color) && (
                <div className="space-y-2">
                  <Label>Text Color</Label>
                  <div className="flex items-center gap-2">
                    <Input
                      type="color"
                      value={config.textColor || config.style?.color || '#ffffff'}
                      onChange={(e) => {
                        if (config.textColor !== undefined) {
                          updateParameter('textColor', e.target.value);
                        } else {
                          updateParameter('style', { ...config.style, color: e.target.value });
                        }
                      }}
                      className="w-16"
                    />
                    <Input
                      type="text"
                      value={config.textColor || config.style?.color || '#ffffff'}
                      onChange={(e) => {
                        if (config.textColor !== undefined) {
                          updateParameter('textColor', e.target.value);
                        } else {
                          updateParameter('style', { ...config.style, color: e.target.value });
                        }
                      }}
                      className="flex-1"
                    />
                  </div>
                </div>
              )}

              {config.duration !== undefined && (
                <div className="space-y-2">
                  <Label>Duration (ms): {config.duration}</Label>
                  <Input
                    type="range"
                    min="1000"
                    max="10000"
                    value={config.duration}
                    onChange={(e) => updateParameter('duration', Number(e.target.value))}
                    className="w-full"
                  />
                </div>
              )}

              {/* Add separator if both common parameters and schema parameters exist */}
              {((config.text || config.overlayText || config.displayText) ||
                (config.textColor || config.style?.color) ||
                config.duration !== undefined) &&
                schema?.parameters?.length && (
                  <Separator />
                )}

              {/* Schema Parameters */}
              {schema?.parameters?.length ? (
                schema.parameters.map((param) => (
                  <div key={param.name} className="space-y-2">
                    <Label className="text-sm font-medium">
                      {param.name}
                      {param.description && (
                        <span className="text-xs text-muted-foreground ml-2">
                          {param.description}
                        </span>
                      )}
                    </Label>
                    {renderParameterControl(param, config[param.name])}
                  </div>
                ))
              ) : !((config.text || config.overlayText || config.displayText) ||
                (config.textColor || config.style?.color) ||
                config.duration !== undefined) && (
                <div className="text-center py-8 text-muted-foreground">
                  <p>No parameters available</p>
                  <p className="text-xs mt-2">Animation will use default settings</p>
                </div>
              )}

              {/* URL Preview Section */}
              {iframeUrl && (
                <>
                  <Separator className="my-4" />
                  <div className="space-y-2">
                    <Label className="text-sm font-medium">Current URL</Label>
                    <div className="flex items-center gap-2">
                      <Input
                        value={iframeUrl}
                        readOnly
                        className="flex-1 font-mono"
                        style={{ fontSize: '12px' }}
                      />
                      <Button
                        onClick={() => copyToClipboard(iframeUrl)}
                        variant="outline"
                        size="sm"
                        className="shrink-0"
                      >
                        {isCopied ? (
                          <Check className="w-3 h-3" />
                        ) : (
                          <Copy className="w-3 h-3" />
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