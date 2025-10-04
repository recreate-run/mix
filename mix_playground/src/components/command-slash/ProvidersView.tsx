import { CheckCircle, Settings } from 'lucide-react';
import {
  CommandEmpty,
  CommandGroup,
  CommandItem,
} from '@/components/ui/command';
import type { Provider, LoginProvider } from '@/types/command-slash';
import { BackButton } from './shared/BackButton';
import { StatusBadge } from './shared/StatusBadge';

type ProviderViewType = 'login' | 'logout' | 'status';

interface ProvidersViewProps {
  type: ProviderViewType;
  providers: Provider[] | LoginProvider[];
  onBackToCommands: () => void;
  onProviderSelect: (providerId: string) => void;
}

export function ProvidersView({
  type,
  providers,
  onBackToCommands,
  onProviderSelect,
}: ProvidersViewProps) {
  const getTitle = () => {
    switch (type) {
      case 'login':
        return `Providers (${providers.length})`;
      case 'logout':
        return `Providers (${providers.length})`;
      case 'status':
        return `Providers (${providers.length})`;
      default:
        return `Providers (${providers.length})`;
    }
  };

  const getDescription = (provider: Provider | LoginProvider) => {
    switch (type) {
      case 'login':
        const loginProvider = provider as LoginProvider;
        return provider.authenticated
          ? 'Authenticated'
          : `Supports: ${loginProvider.authMethods
              .map((m) => (m === 'api_key' ? 'API Key' : 'OAuth'))
              .join(', ')}`;
      case 'logout':
        return provider.authenticated ? 'Authenticated' : '';
      case 'status':
        return provider.authenticated
          ? 'Authenticated'
          : 'Not authenticated - select to authenticate';
      default:
        return '';
    }
  };

  const handleProviderSelect = (providerId: string) => {
    onProviderSelect(providerId);
  };

  if (!providers.length) {
    const emptyMessage =
      type === 'logout'
        ? 'No authenticated providers found'
        : 'No providers found';
    return <CommandEmpty>{emptyMessage}</CommandEmpty>;
  }

  return (
    <CommandGroup heading={getTitle()}>
      <BackButton
        label="Back to Commands"
        onSelect={onBackToCommands}
        value="back-to-commands"
      />

      {providers.map((provider) => {
        const isDisabled = type === 'status' && !provider.authenticated;

        return (
          <CommandItem
            key={provider.id}
            onSelect={() => handleProviderSelect(provider.id)}
            value={provider.displayName}
            className={isDisabled ? 'opacity-50 cursor-not-allowed' : ''}
          >
            <Settings className="size-4 text-muted-foreground" />
            <div className="flex-1">
              <div className="flex items-center gap-2 font-medium text-sm">
                {provider.displayName}
                <div className="flex items-center gap-2">
                  {provider.authenticated && type !== 'logout' && (
                    <CheckCircle className="h-4 w-4 text-green-600" />
                  )}
                  {provider.isPreferred && <StatusBadge status="preferred" />}
                </div>
              </div>
              {getDescription(provider) && (
                <div className="text-muted-foreground text-xs">
                  {getDescription(provider)}
                </div>
              )}
            </div>
          </CommandItem>
        );
      })}
    </CommandGroup>
  );
}
