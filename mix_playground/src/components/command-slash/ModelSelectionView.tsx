import { Settings } from 'lucide-react';
import {
  CommandEmpty,
  CommandGroup,
  CommandItem,
} from '@/components/ui/command';
import type { HierarchicalModelData } from '@/types';
import { BackButton } from './shared/BackButton';
import { StatusBadge } from './shared/StatusBadge';

interface ModelSelectionViewProps {
  hierarchicalModelData: HierarchicalModelData;
  selectedProvider: string | null;
  onBackToCommands: () => void;
  onBackToProviders: () => void;
  onProviderSelect: (providerId: string) => void;
  onModelSelect: (providerId: string, modelId: string) => void;
}

export function ModelSelectionView({
  hierarchicalModelData,
  selectedProvider,
  onBackToCommands,
  onBackToProviders,
  onProviderSelect,
  onModelSelect,
}: ModelSelectionViewProps) {
  // If we have a selected provider, show models for that provider
  if (selectedProvider) {
    const provider = hierarchicalModelData.providers.find(
      (p) => p.id === selectedProvider
    );

    if (!provider) {
      return <CommandEmpty>Provider not found</CommandEmpty>;
    }

    const handleModelSelect = (modelId: string) => {
      onModelSelect(selectedProvider, modelId);
    };

    if (!provider.models.length) {
      return <CommandEmpty>No models found</CommandEmpty>;
    }

    return (
      <CommandGroup
        heading={`${provider.displayName} Models (${provider.models.length})`}
      >
        <BackButton
          label="Back to Providers"
          onSelect={onBackToProviders}
          value="back-to-providers"
        />

        {provider.models.map((model) => (
          <CommandItem
            key={model.id}
            onSelect={() => handleModelSelect(model.id)}
            value={model.displayName}
          >
            <Settings className="size-4 text-muted-foreground" />
            <div className="flex-1">
              <div className="flex items-center gap-2 font-medium text-sm">
                {model.displayName}
                {model.isSelected && <StatusBadge status="current" />}
              </div>
            </div>
          </CommandItem>
        ))}
      </CommandGroup>
    );
  }

  // Show provider selection
  const handleProviderSelect = (providerId: string) => {
    const provider = hierarchicalModelData.providers.find(
      (p) => p.id === providerId
    );
    if (provider && !provider.authenticated) {
      // Don't allow selection of unauthenticated providers
      return;
    }
    onProviderSelect(providerId);
  };

  if (!hierarchicalModelData.providers.length) {
    return <CommandEmpty>No providers found</CommandEmpty>;
  }

  return (
    <CommandGroup
      heading={`Providers (${hierarchicalModelData.providers.length})`}
    >
      <BackButton
        label="Back to Commands"
        onSelect={onBackToCommands}
        value="back-to-commands"
      />

      {hierarchicalModelData.providers.map((provider) => {
        const isDisabled = !provider.authenticated;

        return (
          <CommandItem
            key={provider.id}
            onSelect={() => {
              if (!isDisabled) {
                handleProviderSelect(provider.id);
              }
            }}
            value={provider.displayName}
            className={isDisabled ? 'opacity-50 cursor-not-allowed' : ''}
          >
            <Settings className="size-4 text-muted-foreground" />
            <div className="flex-1">
              <div className="flex items-center gap-2 font-medium text-sm">
                {provider.displayName}
                <div className="flex items-center gap-2">
                  {provider.isPreferred && <StatusBadge status="preferred" />}
                  {!provider.authenticated && (
                    <StatusBadge status="not-authenticated" />
                  )}
                </div>
              </div>
              <div className="text-muted-foreground text-xs">
                {`${provider.models.length} models available${
                  provider.authenticated ? ' • Authenticated' : ''
                }`}
              </div>
            </div>
          </CommandItem>
        );
      })}
    </CommandGroup>
  );
}
