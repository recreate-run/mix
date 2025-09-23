import { useState } from 'react';
import { Search, Loader2, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useToolsStatus, useStoreCredentials, useDeleteCredentials } from '@/hooks/useTools';

export function WebSearchToolsCard() {
  const { data: toolsStatus, isLoading } = useToolsStatus();
  const storeCredentials = useStoreCredentials();
  const deleteCredentials = useDeleteCredentials();
  
  const [apiKeys, setApiKeys] = useState<Record<string, string>>({});

  const webSearchCategory = toolsStatus?.categories?.web_search;
  const braveSearchTool = webSearchCategory?.tools?.find(tool => tool.provider === 'brave');

  const handleApiKeyChange = (provider: string, value: string) => {
    setApiKeys(prev => ({ ...prev, [provider]: value }));
  };

  const handleStoreApiKey = async (provider: string) => {
    const apiKey = apiKeys[provider]?.trim();
    if (!apiKey) return;

    await storeCredentials.mutateAsync({
      provider,
      api_key: apiKey,
    });

    // Clear the input after successful storage
    setApiKeys(prev => ({ ...prev, [provider]: '' }));
  };

  const handleDeleteCredentials = async (provider: string) => {
    await deleteCredentials.mutateAsync({
      provider,
    });
  };

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <div className="flex items-center gap-2">
            <Search className="h-5 w-5" />
            <CardTitle>Web Search Tools</CardTitle>
          </div>
          <CardDescription>Configure web search capabilities</CardDescription>
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
          <Search className="h-5 w-5" />
          <CardTitle>Web Search Tools</CardTitle>
        </div>
        <CardDescription>Configure web search capabilities</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {braveSearchTool ? (
          <div className="p-4 border rounded-lg">
            <div className="flex items-start justify-between">
              <div>
                <p className="font-medium">{braveSearchTool.displayName}</p>
                <p className="text-sm text-muted-foreground mt-1 pr-4 max-w-md">
                  {braveSearchTool.description}
                </p>
                <div className="flex items-center gap-2 mt-2">
                  {braveSearchTool.authenticated ? (
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

              {braveSearchTool.authenticated && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => handleDeleteCredentials('brave')}
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

            {!braveSearchTool.authenticated && braveSearchTool.apiKeyRequired && (
              <div className="mt-4 space-y-3">
                <div className="space-y-2">
                  <Label htmlFor="brave-api-key" className="text-sm font-medium">
                    Brave Search API Key
                  </Label>
                  <div className="flex gap-2">
                    <Input
                      id="brave-api-key"
                      type="password"
                      placeholder="Enter your Brave Search API key..."
                      value={apiKeys.brave || ''}
                      onChange={(e) => handleApiKeyChange('brave', e.target.value)}
                      disabled={storeCredentials.isPending}
                      className="flex-1"
                    />
                    <Button
                      onClick={() => handleStoreApiKey('brave')}
                      disabled={storeCredentials.isPending || !apiKeys.brave?.trim()}
                      size="sm"
                      className="flex items-center gap-2"
                    >
                      {storeCredentials.isPending ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Search className="h-4 w-4" />
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
              No web search tools available.
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}