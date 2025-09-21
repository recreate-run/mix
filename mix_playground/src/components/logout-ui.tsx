import { useState } from "react";
import { Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { logoutProvider } from "@/handlers/logout-command-handler";
import { ProviderInfo } from "@/types/provider";

// Logout state from command handler
interface LogoutState {
  providers: ProviderInfo[];
}

interface LogoutUIProps {
  logoutState: LogoutState;
  onUpdate: (message: any) => void;
}

export function LogoutUI({ logoutState, onUpdate }: LogoutUIProps) {
  // Validate required logout state data
  if (!logoutState.providers || logoutState.providers.length === 0) {
    return (
      <Card>
        <CardContent className="p-4">
          <h3 className="font-medium mb-2">No authenticated providers found</h3>
          <p className="text-sm text-muted-foreground">Use <code>/login</code> to authenticate with a provider first.</p>
        </CardContent>
      </Card>
    );
  }

  const [isLoading, setIsLoading] = useState(false);
  const [loggingOutProvider, setLoggingOutProvider] = useState<string | null>(null);
  
  // Handle provider selection for logout
  const handleProviderSelect = async (provider: string) => {
    if (isLoading) return;
    
    setIsLoading(true);
    setLoggingOutProvider(provider);
    
    try {
      const result = await logoutProvider(provider);
      onUpdate(result);
    } finally {
      setIsLoading(false);
      setLoggingOutProvider(null);
    }
  };
  
  return (
    <Card>
      <CardContent className="p-4">
        <h3 className="font-medium mb-2">Select a provider to log out from:</h3>
        <div className="space-y-2">
          {logoutState.providers.map(provider => (
            <Button
              key={provider.id}
              onClick={() => handleProviderSelect(provider.id)}
              variant="outline"
              className={`w-full justify-start ${provider.isPreferred ? "border-2 border-primary" : ""}`}
              disabled={isLoading}
            >
              {provider.displayName}
              {provider.isPreferred && (
                <span className="ml-auto text-xs text-muted-foreground">(Preferred)</span>
              )}
              {provider.authMethod && (
                <span className="ml-2 text-xs text-muted-foreground">
                  via {provider.authMethod === "api_key" ? "API Key" : "OAuth"}
                </span>
              )}
              {loggingOutProvider === provider.id && isLoading && (
                <Loader2 className="h-4 w-4 ml-2 animate-spin" />
              )}
            </Button>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}