import { useQuery } from '@tanstack/react-query';
import { mix } from '@/lib/mix-sdk';
import { CACHE_KEYS } from '@/lib/cache-keys';

export interface LoginProviderInfo {
  id: string;
  displayName: string;
  authMethods: ('api_key' | 'oauth')[];
  authenticated: boolean;
  apiKeyFormat: string;
  isPreferred: boolean;
}

// Authentication method constants
const AUTH_METHODS: Record<string, ('api_key' | 'oauth')[]> = {
  anthropic: ['api_key', 'oauth'],
  openai: ['api_key'],
  openrouter: ['api_key'],
  gemini: ['api_key'],
};

const API_KEY_FORMATS: Record<string, string> = {
  anthropic: 'sk-ant-...',
  openai: 'sk-...',
  openrouter: 'sk-...',
  gemini: 'AI...',
};

async function fetchProviders(): Promise<LoginProviderInfo[]> {
  // Get authentication status and available providers
  const [authStatus, preferencesResponse] = await Promise.all([
    mix.authentication.getAuthStatus(),
    mix.preferences.get()
  ]);

  if (!preferencesResponse.availableProviders) {
    throw new Error('Failed to fetch available providers');
  }

  const preferredProvider = preferencesResponse.preferences?.preferredProvider;
  const providers: LoginProviderInfo[] = [];

  // Process all available providers
  Object.entries(preferencesResponse.availableProviders).forEach(([providerId, data]: [string, any]) => {
    const authProvider = authStatus.providers?.[providerId];
    const isAuthenticated = authProvider?.authenticated || false;

    // Extract clean display name
    const name = data.displayName || authProvider?.displayName || providerId;
    const cleanName = name.replace(' ⭐', '');
    const isPreferred = name.includes('⭐') || providerId === preferredProvider;

    providers.push({
      id: providerId,
      displayName: cleanName,
      authMethods: AUTH_METHODS[providerId] || ['api_key'],
      authenticated: isAuthenticated,
      apiKeyFormat: API_KEY_FORMATS[providerId] || 'API key',
      isPreferred
    });
  });

  // Sort providers - authenticated and preferred first, then unauthenticated
  providers.sort((a, b) => {
    if (a.authenticated !== b.authenticated) {
      return a.authenticated ? -1 : 1;
    }
    if (a.isPreferred !== b.isPreferred) {
      return a.isPreferred ? -1 : 1;
    }
    return a.displayName.localeCompare(b.displayName);
  });

  return providers;
}

export function useProviders() {
  return useQuery({
    queryKey: [...CACHE_KEYS.preferences, 'providers'],
    queryFn: fetchProviders,
    retry: 2,
  });
}