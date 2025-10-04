import { IconLogin, IconLogout, IconServer } from '@tabler/icons-react';
import { useQueryClient } from '@tanstack/react-query';
import { Eye, Loader2, Search, Settings } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { toast } from 'sonner';
import { OAuthCodeDialog } from '@/components/oauth-code-dialog';
import { ProvidersLoadingSkeleton } from '@/components/provider-skeleton';
import { ToolCard } from '@/components/tool-card';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group';
import {
  authenticateWithApiKey,
  startOAuthFlow,
} from '@/handlers/login-command-handler';
import { logoutProvider } from '@/handlers/logout-command-handler';
import { useProviders } from '@/hooks/useProviders';
import { useToolsStatus } from '@/hooks/useTools';

interface SettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SettingsDialog({ open, onOpenChange }: SettingsDialogProps) {
  const queryClient = useQueryClient();

  // Fetch providers with TanStack Query
  const {
    data: allProviders = [],
    isLoading: loadingProviders,
    refetch,
  } = useProviders();

  // Fetch tools status
  const { data: toolsStatus, isLoading: loadingTools } = useToolsStatus();

  // Login/logout state
  const [loggingOutProvider, setLoggingOutProvider] = useState<string | null>(
    null
  );
  const [loginInProgress, setLoginInProgress] = useState<
    Record<string, boolean>
  >({});
  const [apiKeys, setApiKeys] = useState<Record<string, string>>({});
  const [selectedAuthMethod, setSelectedAuthMethod] = useState<
    Record<string, 'api_key' | 'oauth'>
  >({});

  // OAuth code dialog state
  const [oauthCodeDialog, setOauthCodeDialog] = useState<{
    open: boolean;
    provider: string;
    oauthState: string;
  }>({
    open: false,
    provider: '',
    oauthState: '',
  });

  // Initialize auth method selection for unauthenticated providers
  const initializeAuthMethods = useCallback(() => {
    const newSelectedMethods: Record<string, 'api_key' | 'oauth'> = {};
    allProviders.forEach((provider) => {
      if (!provider.authenticated && provider.authMethods.length > 0) {
        newSelectedMethods[provider.id] = provider.authMethods[0];
      }
    });
    setSelectedAuthMethod((prev) => ({ ...prev, ...newSelectedMethods }));
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
        toast.success(
          `Successfully logged out from ${allProviders.find((p) => p.id === providerId)?.displayName || providerId}`
        );
        // Invalidate and refetch provider data
        queryClient.invalidateQueries({
          queryKey: ['preferences', 'providers'],
        });
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
    const provider = allProviders.find((p) => p.id === providerId);
    const authMethod =
      selectedAuthMethod[providerId] || provider?.authMethods[0];
    if (!authMethod) return;

    setLoginInProgress((prev) => ({ ...prev, [providerId]: true }));

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
          setApiKeys((prev) => ({ ...prev, [providerId]: '' }));
          // Invalidate and refetch provider data
          queryClient.invalidateQueries({
            queryKey: ['preferences', 'providers'],
          });
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
            try {
              const { open: shellOpen } = await import(
                '@tauri-apps/plugin-shell'
              );
              await shellOpen(authUrl);
              toast.info(
                'OAuth browser opened. Please complete authentication in the browser.'
              );

              // Show OAuth code dialog after opening browser
              setOauthCodeDialog({
                open: true,
                provider: providerId,
                oauthState: result.loginData.oauthState,
              });
            } catch (shellError) {
              console.warn(
                'Tauri shell failed, falling back to window.open:',
                shellError
              );
              try {
                window.open(authUrl, '_blank', 'width=600,height=800');
                toast.info(
                  'OAuth window opened. Please complete authentication in the new window.'
                );

                // Show OAuth code dialog after opening browser
                setOauthCodeDialog({
                  open: true,
                  provider: providerId,
                  oauthState: result.loginData.oauthState,
                });
              } catch (windowError) {
                console.error(
                  'Both browser opening methods failed:',
                  windowError
                );
                toast.error(
                  'Failed to open OAuth browser. Please copy this URL manually: ' +
                    authUrl
                );
              }
            }
          }
        } else {
          toast.error(result.content || 'Failed to start OAuth flow');
        }
      }
    } catch (error) {
      console.error('Login failed:', error);
      toast.error('Login failed. Please try again.');
    } finally {
      setLoginInProgress((prev) => ({ ...prev, [providerId]: false }));
    }
  };

  const handleAuthMethodChange = (
    providerId: string,
    method: 'api_key' | 'oauth'
  ) => {
    setSelectedAuthMethod((prev) => ({ ...prev, [providerId]: method }));
    // Clear API key when switching away from API key method
    if (method !== 'api_key') {
      setApiKeys((prev) => ({ ...prev, [providerId]: '' }));
    }
  };

  const handleApiKeyChange = (providerId: string, value: string) => {
    setApiKeys((prev) => ({ ...prev, [providerId]: value }));
  };

  return (
    <>
      <Dialog onOpenChange={onOpenChange} open={open}>
        <DialogContent className="max-h-[80vh] max-w-2xl overflow-hidden">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2">
              <Settings className="h-5 w-5" />
              Settings
            </DialogTitle>
            <DialogDescription>
              Manage your providers, tools, and authentication settings.
            </DialogDescription>
          </DialogHeader>

          <div className="max-h-[60vh] min-h-[500px] space-y-6 overflow-y-auto">
            {/* Providers Section */}
            <Card>
              <CardHeader>
                <div className="flex items-center gap-2">
                  <IconServer className="h-5 w-5" />
                  <CardTitle>AI Providers</CardTitle>
                </div>
              </CardHeader>
              <CardContent>
                {loadingProviders ? (
                  <ProvidersLoadingSkeleton />
                ) : allProviders.length === 0 ? (
                  <div className="py-8 text-center">
                    <p className="text-muted-foreground text-sm">
                      No providers available.
                    </p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {allProviders.map((provider) => (
                      <div className="rounded-lg border p-4" key={provider.id}>
                        <div className="flex items-center justify-between">
                          <div>
                            <p className="font-medium">
                              {provider.displayName}
                            </p>
                            <div className="mt-1 flex items-center gap-2">
                              {provider.authenticated ? (
                                <>
                                  <Badge
                                    className="border-green-600 text-green-600 text-xs"
                                    variant="outline"
                                  >
                                    ✓ Authenticated
                                  </Badge>
                                  {provider.isPreferred && (
                                    <Badge
                                      className="text-xs"
                                      variant="default"
                                    >
                                      Preferred
                                    </Badge>
                                  )}
                                </>
                              ) : (
                                <Badge
                                  className="text-muted-foreground text-xs"
                                  variant="outline"
                                >
                                  Not authenticated
                                </Badge>
                              )}
                            </div>
                          </div>

                          {provider.authenticated ? (
                            <Button
                              className="flex items-center gap-2"
                              disabled={!!loggingOutProvider}
                              onClick={() => handleProviderLogout(provider.id)}
                              size="sm"
                              variant="outline"
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
                                <Label className="font-medium text-sm">
                                  Authentication Method
                                </Label>
                                <RadioGroup
                                  className="mt-2 flex gap-6"
                                  onValueChange={(value: string) =>
                                    handleAuthMethodChange(
                                      provider.id,
                                      value as 'api_key' | 'oauth'
                                    )
                                  }
                                  value={
                                    selectedAuthMethod[provider.id] ||
                                    provider.authMethods[0]
                                  }
                                >
                                  {provider.authMethods.includes('api_key') && (
                                    <div className="flex items-center space-x-2">
                                      <RadioGroupItem
                                        id={`${provider.id}-api-key`}
                                        value="api_key"
                                      />
                                      <Label
                                        className="text-sm"
                                        htmlFor={`${provider.id}-api-key`}
                                      >
                                        API Key
                                      </Label>
                                    </div>
                                  )}
                                  {provider.authMethods.includes('oauth') && (
                                    <div className="flex items-center space-x-2">
                                      <RadioGroupItem
                                        id={`${provider.id}-oauth`}
                                        value="oauth"
                                      />
                                      <Label
                                        className="text-sm"
                                        htmlFor={`${provider.id}-oauth`}
                                      >
                                        OAuth
                                      </Label>
                                    </div>
                                  )}
                                </RadioGroup>
                              </div>
                            )}

                            {/* API Key input or OAuth button */}
                            <div>
                              {(selectedAuthMethod[provider.id] ||
                                provider.authMethods[0]) === 'api_key' ? (
                                <div className="space-y-2">
                                  <Label
                                    className="font-medium text-sm"
                                    htmlFor={`${provider.id}-key`}
                                  >
                                    API Key{' '}
                                    {provider.apiKeyFormat && (
                                      <span className="text-muted-foreground text-xs">
                                        ({provider.apiKeyFormat})
                                      </span>
                                    )}
                                  </Label>
                                  <div className="flex gap-2">
                                    <Input
                                      className="flex-1"
                                      disabled={loginInProgress[provider.id]}
                                      id={`${provider.id}-key`}
                                      onChange={(e) =>
                                        handleApiKeyChange(
                                          provider.id,
                                          e.target.value
                                        )
                                      }
                                      placeholder={
                                        provider.apiKeyFormat ||
                                        'Enter API key...'
                                      }
                                      type="password"
                                      value={apiKeys[provider.id] || ''}
                                    />
                                    <Button
                                      className="flex items-center gap-2"
                                      disabled={
                                        loginInProgress[provider.id] ||
                                        !apiKeys[provider.id]?.trim()
                                      }
                                      onClick={() => handleLogin(provider.id)}
                                      size="sm"
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
                                    className="flex items-center gap-2"
                                    disabled={loginInProgress[provider.id]}
                                    onClick={() => handleLogin(provider.id)}
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

            {/* Tools & Agents Section */}
            <div className="space-y-4">
              <div className="flex items-center gap-2">
                <h3 className="font-semibold text-lg">Tools & Subagents</h3>
              </div>

              {!loadingTools &&
                toolsStatus?.categories &&
                Object.entries(toolsStatus.categories).map(
                  ([categoryId, category]) => {
                    // Map category IDs to icons
                    const categoryIcons: Record<string, React.ReactNode> = {
                      web_search: <Search className="h-5 w-5" />,
                      multimodal_analyzer: <Eye className="h-5 w-5" />,
                    };

                    return (
                      <ToolCard
                        categoryDisplayName={category.displayName}
                        icon={
                          categoryIcons[categoryId] || (
                            <Settings className="h-5 w-5" />
                          )
                        }
                        key={categoryId}
                        tools={category.tools}
                      />
                    );
                  }
                )}
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {/* OAuth Code Dialog */}
      <OAuthCodeDialog
        oauthState={oauthCodeDialog.oauthState}
        onOpenChange={(open) =>
          setOauthCodeDialog((prev) => ({ ...prev, open }))
        }
        onSuccess={() => {
          // Refresh providers data after successful authentication
          refetch();
          // Clear login in progress state
          setLoginInProgress((prev) => ({
            ...prev,
            [oauthCodeDialog.provider]: false,
          }));
        }}
        open={oauthCodeDialog.open}
        provider={oauthCodeDialog.provider}
      />
    </>
  );
}
