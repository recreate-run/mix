import { useState } from 'react';
import { Loader2, Trash2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Badge } from '@/components/ui/badge';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { useStoreCredentials, useDeleteCredentials } from '@/hooks/useTools';

interface ToolInfo {
  provider: string;
  displayName: string;
  authenticated: boolean;
  apiKeyRequired: boolean;
}

interface ToolCardProps {
  categoryDisplayName: string;
  icon: React.ReactNode;
  tools: ToolInfo[];
}

export function ToolCard({ categoryDisplayName, icon, tools }: ToolCardProps) {
  const storeCredentials = useStoreCredentials();
  const deleteCredentials = useDeleteCredentials();

  const [apiKeys, setApiKeys] = useState<Record<string, string>>({});

  const handleApiKeyChange = (provider: string, value: string) => {
    setApiKeys((prev) => ({ ...prev, [provider]: value }));
  };

  const handleStoreApiKey = async (provider: string) => {
    const apiKey = apiKeys[provider]?.trim();
    if (!apiKey) return;

    await storeCredentials.mutateAsync({
      provider,
      api_key: apiKey,
    });

    // Clear the input after successful storage
    setApiKeys((prev) => ({ ...prev, [provider]: '' }));
  };

  const handleDeleteCredentials = async (provider: string) => {
    await deleteCredentials.mutateAsync({
      provider,
    });
  };

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center gap-2">
          {icon}
          <CardTitle>{categoryDisplayName}</CardTitle>
        </div>
      </CardHeader>
      <CardContent className="space-y-4">
        {tools.length > 0 ? (
          tools.map((tool) => (
            <div key={tool.provider} className="rounded-lg border p-4">
              <div className="flex items-start justify-between">
                <div>
                  <p className="font-medium">{tool.displayName}</p>
                  <div className="mt-2 flex items-center gap-2">
                    {tool.authenticated ? (
                      <Badge
                        variant="outline"
                        className="border-green-600 text-green-600 text-xs"
                      >
                        ✓ Configured
                      </Badge>
                    ) : (
                      <Badge
                        variant="outline"
                        className="text-muted-foreground text-xs"
                      >
                        Requires setup
                      </Badge>
                    )}
                  </div>
                </div>

                {tool.authenticated && (
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => handleDeleteCredentials(tool.provider)}
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

              {!tool.authenticated && tool.apiKeyRequired && (
                <div className="mt-4 space-y-3">
                  <div className="space-y-2">
                    <Label
                      htmlFor={`${tool.provider}-api-key`}
                      className="font-medium text-sm"
                    >
                      API Key
                    </Label>
                    <div className="flex gap-2">
                      <Input
                        id={`${tool.provider}-api-key`}
                        type="password"
                        placeholder={`Enter your ${tool.displayName} API key...`}
                        value={apiKeys[tool.provider] || ''}
                        onChange={(e) =>
                          handleApiKeyChange(tool.provider, e.target.value)
                        }
                        disabled={storeCredentials.isPending}
                        className="flex-1"
                      />
                      <Button
                        onClick={() => handleStoreApiKey(tool.provider)}
                        disabled={
                          storeCredentials.isPending ||
                          !apiKeys[tool.provider]?.trim()
                        }
                        size="sm"
                        className="flex items-center gap-2"
                      >
                        {storeCredentials.isPending ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          'Configure'
                        )}
                      </Button>
                    </div>
                  </div>
                </div>
              )}
            </div>
          ))
        ) : (
          <div className="py-8 text-center">
            <p className="text-muted-foreground text-sm">
              No tools available in this category.
            </p>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
