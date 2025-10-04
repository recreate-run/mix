import { toast } from 'sonner';
import { mix } from '@/lib/mix-sdk';
import type { UIMessage } from '@/types/message';

/**
 * Logs out from the specified provider by deleting credentials
 */
export async function logoutProvider(provider: string): Promise<UIMessage> {
  try {
    // Delete credentials using the SDK
    await mix.authentication.deleteCredentials({ provider });

    // Get updated auth status to confirm logout
    const authStatus = await mix.authentication.getAuthStatus();
    const isStillAuthenticated =
      authStatus.providers?.[provider]?.authenticated;

    if (isStillAuthenticated) {
      return {
        content: `❌ Failed to log out from ${provider}. Please try again.`,
        from: 'assistant',
        frontend_only: true,
        suppressChatMessage: true,
      };
    }

    // Show success toast notification
    toast.success('Logged out successfully', {
      description: `You have been logged out from ${provider}`,
      duration: 3000,
    });

    return {
      content: `✅ Successfully logged out from ${provider}`,
      from: 'assistant',
      frontend_only: true,
      suppressChatMessage: true, // Hide this success message from the chat UI
    };
  } catch (error) {
    return {
      content: `❌ Failed to log out: ${error instanceof Error ? error.message : 'Unknown error'}`,
      from: 'assistant',
      frontend_only: true,
      suppressChatMessage: true,
    };
  }
}
