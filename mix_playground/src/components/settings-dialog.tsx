import { useState, useEffect } from "react";
import {
  IconServer,
  IconLogout,
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
import { logoutProvider } from "@/handlers/logout-command-handler";
import { mix } from "@/lib/mix-sdk";
import { ProviderInfo } from "@/types/provider";
import { toast } from "sonner";

interface SettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SettingsDialog({ open, onOpenChange }: SettingsDialogProps) {
  // Provider logout state
  const [providers, setProviders] = useState<ProviderInfo[]>([]);
  const [loadingProviders, setLoadingProviders] = useState(false);
  const [loggingOutProvider, setLoggingOutProvider] = useState<string | null>(null);

  // Load authenticated providers when providers tab might be accessed
  useEffect(() => {
    if (open) {
      loadProviders();
    }
  }, [open]);

  const loadProviders = async () => {
    setLoadingProviders(true);
    try {
      // Get authentication status directly from Mix SDK
      const authStatus = await mix.authentication.getAuthStatus();
      const formattedProviders: ProviderInfo[] = [];

      if (authStatus.providers) {
        Object.entries(authStatus.providers).forEach(([id, provider]) => {
          if (provider.authenticated) {
            // Extract the base name without the star symbol
            const name = provider.displayName || id;
            const cleanName = name.replace(" ⭐", "");
            const isPreferred = name.includes("⭐");

            formattedProviders.push({
              id,
              displayName: cleanName,
              authenticated: true,
              authMethod: provider.authMethod === "none" ? undefined : provider.authMethod,
              isPreferred,
              authMethods: ["api_key", "oauth"]
            });
          }
        });
      }

      // Sort providers - preferred first, then alphabetically
      formattedProviders.sort((a, b) => {
        if (a.isPreferred !== b.isPreferred) {
          return a.isPreferred ? -1 : 1;
        }
        return a.displayName.localeCompare(b.displayName);
      });

      setProviders(formattedProviders);
    } catch (error) {
      console.error('Failed to load providers:', error);
      setProviders([]);
    } finally {
      setLoadingProviders(false);
    }
  };

  const handleProviderLogout = async (providerId: string) => {
    if (loggingOutProvider) return;

    setLoggingOutProvider(providerId);

    try {
      const result = await logoutProvider(providerId);

      if (result.suppressChatMessage) {
        toast.success(`Successfully logged out from ${providers.find(p => p.id === providerId)?.displayName || providerId}`);
        await loadProviders(); // Reload providers list
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

        <div className=" overflow-y-auto max-h-[50vh] ">
          <Card className="border-0">
            <CardContent className="p-0 border-0">
              {loadingProviders ? (
                <div className="flex items-center justify-center">
                  <Loader2 className="h-6 w-6 animate-spin" />
                  <span className="ml-2 text-sm text-muted-foreground">Loading providers...</span>
                </div>
              ) : providers.length === 0 ? (
                <div className="text-center py-8">
                  <p className="text-sm text-muted-foreground">
                    No authenticated providers found.
                  </p>
                  <p className="text-xs text-muted-foreground mt-1">
                    Use <code className="bg-muted px-1 rounded">/login</code> to authenticate with a provider first.
                  </p>
                </div>
              ) : (
                <div className="space-y-3">
                  {providers.map((provider) => (
                    <div
                      key={provider.id}
                      className="flex items-center justify-between p-3 border rounded-lg"
                    >
                      <div className="flex items-center gap-3">
                        <div>
                          <p className="font-medium">{provider.displayName}</p>
                          <div className="flex items-center gap-2 mt-1">
                            <Badge variant="secondary" className="text-xs">
                              {provider.authMethod === "api_key" ? "API Key" : "OAuth"}
                            </Badge>
                            {provider.isPreferred && (
                              <Badge variant="default" className="text-xs">
                                Preferred
                              </Badge>
                            )}
                          </div>
                        </div>
                      </div>
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