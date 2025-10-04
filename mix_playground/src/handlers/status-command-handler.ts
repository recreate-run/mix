import { mix } from '@/lib/mix-sdk';
import { UIMessage } from '@/types/message';
import { ProviderInfo } from '@/types/provider';
import { handleLoginCommand } from './login-command-handler';

/**
 * Format provider information for the status UI
 */
function formatStatusProviders(providers: Record<string, any>): {
  formattedProviders: ProviderInfo[];
  hasAuthenticatedProvider: boolean;
} {
  const formattedProviders: ProviderInfo[] = [];

  let hasAuthenticatedProvider = false;

  Object.entries(providers).forEach(([id, provider]) => {
    // Extract the base name without the star symbol
    const name = provider.displayName || id;
    const cleanName = name.replace(' ⭐', '');
    const isPreferred = name.includes('⭐');

    // Format provider for UI
    formattedProviders.push({
      id,
      displayName: cleanName,
      authenticated: provider.authenticated,
      authMethod: provider.authMethod,
      isPreferred,
      authMethods: ['api_key', 'oauth'], // Standard auth methods for all providers
    });

    if (provider.authenticated) {
      hasAuthenticatedProvider = true;
    }
  });

  // Sort providers - authenticated and preferred first, then alphabetically
  formattedProviders.sort((a, b) => {
    // Preferred provider first
    if (a.isPreferred !== b.isPreferred) {
      return a.isPreferred ? -1 : 1;
    }

    // Then authenticated providers
    if (a.authenticated !== b.authenticated) {
      return a.authenticated ? -1 : 1;
    }

    // Then alphabetically
    return a.displayName.localeCompare(b.displayName);
  });

  return {
    formattedProviders,
    hasAuthenticatedProvider,
  };
}

/**
 * Handles the status command by using the SDK to check authentication status
 * and returns a message with status UI
 */
export async function handleStatusCommand(): Promise<UIMessage> {
  try {
    // Get authentication status using the SDK
    const authStatus = await mix.authentication.getAuthStatus();

    // Format providers for UI
    const { formattedProviders, hasAuthenticatedProvider } =
      formatStatusProviders(authStatus.providers || {});

    if (hasAuthenticatedProvider) {
      // Find the first authenticated provider for a friendly message
      const authenticatedProvider = formattedProviders.find(
        (provider) => provider.authenticated
      );

      // Format a friendly message
      let content = `✅ **Authenticated with ${authenticatedProvider?.displayName || 'provider'}**`;
      if (authenticatedProvider?.authMethod) {
        content += ` (via ${authenticatedProvider.authMethod === 'api_key' ? 'API Key' : 'OAuth'})`;
      }

      // Include information about number of available providers
      const unauthenticatedCount = formattedProviders.filter(
        (p) => !p.authenticated
      ).length;
      if (unauthenticatedCount > 0) {
        content += `\n\n${unauthenticatedCount} additional provider${unauthenticatedCount > 1 ? 's' : ''} available.`;
      }

      // Return message with status UI for command menu
      return {
        content,
        from: 'assistant',
        frontend_only: true,
        status: {
          providers: formattedProviders,
          hasAuthenticatedProvider,
        },
        statusData: {
          providers: formattedProviders,
        },
      };
    } else {
      // No authenticated providers
      return {
        content:
          '❌ **Not authenticated with any provider**\n\nSelect a provider to authenticate:',
        from: 'assistant',
        frontend_only: true,
        status: {
          providers: formattedProviders,
          hasAuthenticatedProvider,
        },
        statusData: {
          providers: formattedProviders,
        },
      };
    }
  } catch (error) {
    return {
      content: `Failed to check authentication status: ${error instanceof Error ? error.message : 'Unknown error'}`,
      from: 'assistant',
      frontend_only: true,
      suppressChatMessage: true,
    };
  }
}

/**
 * Handles the selection of a provider from the status UI
 * by initiating the login flow for that provider
 */
export async function handleProviderSelection(
  provider: string
): Promise<UIMessage> {
  try {
    // Use the existing login command handler to initiate login for the selected provider
    return await handleLoginCommand();
  } catch (error) {
    return {
      content: `Failed to initiate login for ${provider}: ${error instanceof Error ? error.message : 'Unknown error'}`,
      from: 'assistant',
      frontend_only: true,
      suppressChatMessage: true,
    };
  }
}
