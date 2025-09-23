import { useState } from 'react';
import { Eye, Loader2, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useToolsStatus, useStoreToolCredentials, useDeleteToolCredentials } from '@/hooks/useTools';

export function MultimodalAnalyzersCard() {
  const { data: toolsStatus, isLoading } = useToolsStatus();
  const storeCredentials = useStoreToolCredentials();
  const deleteCredentials = useDeleteToolCredentials();
  
  const [apiKeys, setApiKeys] = useState<Record<string, string>>({});

  const multimodalCategory = toolsStatus?.categories?.multimodal_analyzer;
  const geminiTool = multimodalCategory?.tools?.find(tool => tool.provider === 'gemini');

  const handleApiKeyChange = (provider: string, value: string) => {
    setApiKeys(prev => ({ ...prev, [provider]: value }));
  };

  const handleStoreApiKey = async (provider: string) => {
    const apiKey = apiKeys[provider]?.trim();
    if (!apiKey) return;

    await storeCredentials.mutateAsync({
      toolType: 'multimodal_analyzer',
      provider,
      apiKey,
    });

    // Clear the input after successful storage
    setApiKeys(prev => ({ ...prev, [provider]: '' }));
  };

  const handleDeleteCredentials = async (provider: string) => {
    await deleteCredentials.mutateAsync({
      toolType: 'multimodal_analyzer',
      provider,
    });
  };

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Eye className="h-5 w-5" />
            <CardTitle>Multimodal Analyzers</CardTitle>
          </div>
          <CardDescription>Configure image and document analysis capabilities</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="animate-pulse space-y-3">
            <div className="h-4 bg-muted rounded w-3/4"></div>
            <div className="h-10 bg-muted rounded"></div>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          <Eye className="h-5 w-5" />
          <CardTitle>Multimodal Analyzers</CardTitle>
        </div>
        <CardDescription>Configure image and document analysis capabilities</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {geminiTool ? (
          <div className="p-4 border rounded-lg">
            <div className="flex items-start justify-between">
              <div>
                <p className="font-medium">{geminiTool.displayName}</p>
                <p className="text-sm text-muted-foreground mt-1 pr-4 max-w-md">
                  {geminiTool.description}
                </p>
                <div className="flex items-center gap-2 mt-2">
                  {geminiTool.authenticated ? (
                    <Badge variant="outline" className="text-xs text-green-600 border-green-600">
                      ✓ Configured
                    </Badge>
                  ) : (
                    <Badge variant="outline" className="text-xs text-muted-foreground">
                      Requires setup
                    </Badge>
                  )}
                </div>
              </div>

              {geminiTool.authenticated && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handleDeleteCredentials('gemini')}
                  disabled={deleteCredentials.isPending}
                  className="flex items-center gap-2"
                >
                  {deleteCredentials.isPending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Trash2 className="h-4 w-4" />
                  )}
                  Remove
                </Button>
              )}
            </div>

            {!geminiTool.authenticated && geminiTool.apiKeyRequired && (
              <div className="mt-4 space-y-3">
                <div className="space-y-2">
                  <Label htmlFor="gemini-api-key" className="text-sm font-medium">
                    Gemini API Key <span className="text-xs text-muted-foreground">(AI...)</span>
                  </Label>
                  <div className="flex gap-2">
                    <Input
                      id="gemini-api-key"
                      type="password"
                      placeholder="Enter your Gemini API key..."
                      value={apiKeys.gemini || ''}
                      onChange={(e) => handleApiKeyChange('gemini', e.target.value)}
                      disabled={storeCredentials.isPending}
                      className="flex-1"
                    />
                    <Button
                      onClick={() => handleStoreApiKey('gemini')}
                      disabled={storeCredentials.isPending || !apiKeys.gemini?.trim()}
                      size="sm"
                      className="flex items-center gap-2"
                    >
                      {storeCredentials.isPending ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Eye className="h-4 w-4" />
                      )}
                      Configure
                    </Button>
                  </div>
                </div>
              </div>
            )}
          </div>
        ) : (
          <div className="text-center py-8">
            <p className="text-sm text-muted-foreground">
              No multimodal analyzers available.
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}