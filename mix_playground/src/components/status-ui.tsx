import { CheckCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import type { ProviderInfo } from '@/types/provider';

// Status state
interface StatusState {
  providers: ProviderInfo[];
  hasAuthenticatedProvider: boolean;
}

interface StatusUIProps {
  statusState: StatusState;
}

export function StatusUI({ statusState }: StatusUIProps) {
  return (
    <Card>
      <CardContent className="p-4">
        <h3 className="mb-2 font-medium">Authentication status:</h3>
        <p className="mb-4 text-muted-foreground text-sm">
          Use <code>/login</code> to change authentication settings.
        </p>
        <div className="space-y-2">
          {statusState.providers.map((provider) => (
            <Button
              className={`w-full justify-start ${
                provider.isPreferred ? 'border-2 border-primary' : ''
              }`}
              disabled={true}
              key={provider.id}
              // All buttons are disabled in status view
              variant="outline"
            >
              {provider.displayName}
              {provider.authenticated && (
                <CheckCircle className="ml-2 h-4 w-4 text-green-600" />
              )}
              {provider.isPreferred && (
                <span className="ml-auto text-muted-foreground text-xs">
                  (Preferred)
                </span>
              )}
              {provider.authenticated && provider.authMethod && (
                <span className="ml-auto text-muted-foreground text-xs">
                  via {provider.authMethod === 'api_key' ? 'API Key' : 'OAuth'}
                </span>
              )}
            </Button>
          ))}
        </div>
      </CardContent>
    </Card>
  );
}
