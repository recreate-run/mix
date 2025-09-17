import { useQuery } from '@tanstack/react-query';
import { mix } from '@/lib/mix-sdk';
import type { GetPreferencesResponse } from 'mix-typescript-sdk/models/operations/getpreferences';
import { CACHE_KEYS } from '@/lib/cache-keys';

async function getPreferences(): Promise<GetPreferencesResponse> {
  const response = await mix.preferences.get();
  return response;
}

export function usePreferences() {
  return useQuery({
    queryKey: CACHE_KEYS.preferences,
    queryFn: getPreferences,
    staleTime: 30 * 1000, // 30 seconds - preferences don't change often
    gcTime: 5 * 60 * 1000, // 5 minutes - keep in cache
    refetchOnWindowFocus: false,
    retry: 2,
  });
}

// Helper function to format the current model display
export function formatCurrentModel(preferences: GetPreferencesResponse | undefined): string {
  if (!preferences?.preferences) {
    return 'No model selected';
  }

  const { preferredProvider, mainAgentModel } = preferences.preferences;
  
  if (!preferredProvider || !mainAgentModel) {
    return 'No model selected';
  }

  // Get the display name for the provider
  const providerDisplayName = preferences.availableProviders[preferredProvider]?.displayName || preferredProvider;
  
  // Shorten long model names for better display
  const shortModelName = mainAgentModel.length > 25 
    ? `${mainAgentModel.substring(0, 22)}...`
    : mainAgentModel;

  return `${providerDisplayName}: ${shortModelName}`;
}