import { useState, useEffect, useCallback } from "react";
import {
  IconServer,
  IconLogout,
  IconLogin,
} from '@tabler/icons-react';
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { logoutProvider } from "@/handlers/logout-command-handler";
import { authenticateWithApiKey, startOAuthFlow } from "@/handlers/login-command-handler";
import { useProviders } from "@/hooks/useProviders";
import { ProvidersLoadingSkeleton } from "@/components/provider-skeleton";
import { toast } from "sonner";
import { useQueryClient } from "@tanstack/react-query";

interface SettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SettingsDialog({ open, onOpenChange }: SettingsDialogProps) {
  const queryClient = useQueryClient();

  // Fetch providers with TanStack Query
  const { data: allProviders = [], isLoading: loadingProviders, refetch } = useProviders();

  // Login/logout state
  const [loggingOutProvider, setLoggingOutProvider] = useState<string | null>(null);
  const [loginInProgress, setLoginInProgress] = useState<Record<string, boolean>>({});
  const [apiKeys, setApiKeys] = useState<Record<string, string>>({});
  const [selectedAuthMethod, setSelectedAuthMethod] = useState<Record<string, 'api_key' | 'oauth'>>({});

  // Initialize auth method selection for unauthenticated providers
  const initializeAuthMethods = useCallback(() => {
    const newSelectedMethods: Record<string, 'api_key' | 'oauth'> = {};
    allProviders.forEach(provider => {
      if (!provider.authenticated && provider.authMethods.length > 0) {
        newSelectedMethods[provider.id] = provider.authMethods[0];
      }
    });
    setSelectedAuthMethod(prev => ({ ...prev, ...newSelectedMethods }));
  }, [allProviders]);

  // Initialize auth methods when providers change
  useEffect(() => {
    if (allProviders.length > 0) {
      initializeAuthMethods();
    }
  }, [allProviders, initializeAuthMethods]);

  const handleProviderLogout = async (providerId: string) => {
    if (loggingOutProvider) return;

    setLoggingOutProvider(providerId);

    try {
      const result = await logoutProvider(providerId);

      if (result.suppressChatMessage) {
        toast.success(`Successfully logged out from ${allProviders.find(p => p.id === providerId)?.displayName || providerId}`);
        // Invalidate and refetch provider data
        queryClient.invalidateQueries({ queryKey: ['preferences', 'providers'] });
        await refetch();
      } else {
        toast.error(result.content || 'Logout failed');
      }
    } catch (error) {
      console.error('Logout failed:', error);
      toast.error('Logout failed. Please try again.');
    } finally {
      setLoggingOutProvider(null);
    }
  };

  const handleLogin = async (providerId: string) => {
    const provider = allProviders.find(p => p.id === providerId);
    const authMethod = selectedAuthMethod[providerId] || provider?.authMethods[0];
    if (!authMethod) return;

    setLoginInProgress(prev => ({ ...prev, [providerId]: true }));

    try {
      if (authMethod === 'api_key') {
        const apiKey = apiKeys[providerId];
        if (!apiKey || apiKey.trim() === '') {
          toast.error('Please enter an API key');
          return;
        }

        const result = await authenticateWithApiKey(providerId, apiKey.trim());
        if (result.suppressChatMessage) {
          // Clear the API key after successful authentication
          setApiKeys(prev => ({ ...prev, [providerId]: '' }));
          // Invalidate and refetch provider data
          queryClient.invalidateQueries({ queryKey: ['preferences', 'providers'] });
          await refetch();
        } else {
          toast.error(result.content || 'Authentication failed');
        }
      } else if (authMethod === 'oauth') {
        const result = await startOAuthFlow(providerId);
        if (result.loginData?.oauthState) {
          // OAuth flow started successfully - open the auth URL
          const authUrl = result.content.match(/https?:\/\/[^\s]+/)?.[0];
          if (authUrl) {
            window.open(authUrl, '_blank', 'width=600,height=800');
            toast.info('OAuth window opened. Please complete authentication in the new window.');
          }
        } else {
          toast.error(result.content || 'Failed to start OAuth flow');
        }
      }
    } catch (error) {
      console.error('Login failed:', error);
      toast.error('Login failed. Please try again.');
    } finally {
      setLoginInProgress(prev => ({ ...prev, [providerId]: false }));
    }
  };

  const handleAuthMethodChange = (providerId: string, method: 'api_key' | 'oauth') => {
    setSelectedAuthMethod(prev => ({ ...prev, [providerId]: method }));
    // Clear API key when switching away from API key method
    if (method !== 'api_key') {
      setApiKeys(prev => ({ ...prev, [providerId]: '' }));
    }
  };

  const handleApiKeyChange = (providerId: string, value: string) => {
    setApiKeys(prev => ({ ...prev, [providerId]: value }));
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[80vh] overflow-hidden">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <IconServer className="h-5 w-5" />
            Providers
          </DialogTitle>
          <DialogDescription>
            Manage your authenticated providers and logout if needed.
          </DialogDescription>
        </DialogHeader>

        <div className="overflow-y-auto max-h-[50vh] min-h-[400px]">
          <Card className="border-0">
            <CardContent className="p-0 border-0">
              {loadingProviders ? (
                <ProvidersLoadingSkeleton />
              ) : allProviders.length === 0 ? (
                <div className="text-center py-8">
                  <p className="text-sm text-muted-foreground">
                    No providers available.
                  </p>
                </div>
              ) : (
                <div className="space-y-3">
                  {allProviders.map((provider) => (
                    <div
                      key={provider.id}
                      className="p-4 border rounded-lg"
                    >
                      <div className="flex items-center justify-between">
                        <div>
                          <p className="font-medium">{provider.displayName}</p>
                          <div className="flex items-center gap-2 mt-1">
                            {provider.authenticated ? (
                              <>
                                <Badge variant="outline" className="text-xs text-green-600 border-green-600">
                                  ✓ Authenticated
                                </Badge>
                                {provider.isPreferred && (
                                  <Badge variant="default" className="text-xs">
                                    Preferred
                                  </Badge>
                                )}
                              </>
                            ) : (
                              <Badge variant="outline" className="text-xs text-muted-foreground">
                                Not authenticated
                              </Badge>
                            )}
                          </div>
                        </div>

                        {provider.authenticated ? (
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleProviderLogout(provider.id)}
                            disabled={!!loggingOutProvider}
                            className="flex items-center gap-2"
                          >
                            {loggingOutProvider === provider.id ? (
                              <Loader2 className="h-4 w-4 animate-spin" />
                            ) : (
                              <IconLogout className="h-4 w-4" />
                            )}
                            Logout
                          </Button>
                        ) : null}
                      </div>

                      {!provider.authenticated && (
                        <div className="mt-4 space-y-3">
                          {/* Authentication method selection */}
                          {provider.authMethods.length > 1 && (
                            <div>
                              <Label className="text-sm font-medium">Authentication Method</Label>
                              <RadioGroup
                                value={selectedAuthMethod[provider.id] || provider.authMethods[0]}
                                onValueChange={(value: string) => handleAuthMethodChange(provider.id, value as 'api_key' | 'oauth')}
                                className="flex gap-6 mt-2"
                              >
                                {provider.authMethods.includes('api_key') && (
                                  <div className="flex items-center space-x-2">
                                    <RadioGroupItem value="api_key" id={`${provider.id}-api-key`} />
                                    <Label htmlFor={`${provider.id}-api-key`} className="text-sm">API Key</Label>
                                  </div>
                                )}
                                {provider.authMethods.includes('oauth') && (
                                  <div className="flex items-center space-x-2">
                                    <RadioGroupItem value="oauth" id={`${provider.id}-oauth`} />
                                    <Label htmlFor={`${provider.id}-oauth`} className="text-sm">OAuth</Label>
                                  </div>
                                )}
                              </RadioGroup>
                            </div>
                          )}

                          {/* API Key input or OAuth button */}
                          <div>
                            {(selectedAuthMethod[provider.id] || provider.authMethods[0]) === 'api_key' ? (
                              <div className="space-y-2">
                                <Label htmlFor={`${provider.id}-key`} className="text-sm font-medium">
                                  API Key {provider.apiKeyFormat && (
                                    <span className="text-xs text-muted-foreground">({provider.apiKeyFormat})</span>
                                  )}
                                </Label>
                                <div className="flex gap-2">
                                  <Input
                                    id={`${provider.id}-key`}
                                    type="password"
                                    placeholder={provider.apiKeyFormat || "Enter API key..."}
                                    value={apiKeys[provider.id] || ''}
                                    onChange={(e) => handleApiKeyChange(provider.id, e.target.value)}
                                    disabled={loginInProgress[provider.id]}
                                    className="flex-1"
                                  />
                                  <Button
                                    onClick={() => handleLogin(provider.id)}
                                    disabled={loginInProgress[provider.id] || !apiKeys[provider.id]?.trim()}
                                    size="sm"
                                    className="flex items-center gap-2"
                                  >
                                    {loginInProgress[provider.id] ? (
                                      <Loader2 className="h-4 w-4 animate-spin" />
                                    ) : (
                                      <IconLogin className="h-4 w-4" />
                                    )}
                                    Login
                                  </Button>
                                </div>
                              </div>
                            ) : (
                              <div>
                                <Button
                                  onClick={() => handleLogin(provider.id)}
                                  disabled={loginInProgress[provider.id]}
                                  className="flex items-center gap-2"
                                  size="sm"
                                >
                                  {loginInProgress[provider.id] ? (
                                    <Loader2 className="h-4 w-4 animate-spin" />
                                  ) : (
                                    <IconLogin className="h-4 w-4" />
                                  )}
                                  Connect with OAuth
                                </Button>
                              </div>
                            )}
                          </div>
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </DialogContent>
    </Dialog>
  );
}