import { CheckCircle } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { ProviderInfo } from '@/types/provider';

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
        <h3 className="font-medium mb-2">Authentication status:</h3>
        <p className="text-sm text-muted-foreground mb-4">
          Use <code>/login</code> to change authentication settings.
        </p>
        <div className="space-y-2">
          {statusState.providers.map((provider) => (
            <Button
              key={provider.id}
              variant="outline"
              className={`w-full justify-start ${
                provider.isPreferred ? 'border-2 border-primary' : ''
              }`}
              // All buttons are disabled in status view
              disabled={true}
            >
              {provider.displayName}
              {provider.authenticated && (
                <CheckCircle className="h-4 w-4 ml-2 text-green-600" />
              )}
              {provider.isPreferred && (
                <span className="ml-auto text-xs text-muted-foreground">
                  (Preferred)
                </span>
              )}
              {provider.authenticated && provider.authMethod && (
                <span className="ml-auto text-xs text-muted-foreground">
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
