import { useCallback } from 'react';
import { handleStatusCommand, handleProviderSelection } from '@/handlers/status-command-handler';
import { startOAuthFlow, authenticateWithApiKey, handleOAuthCallback } from '@/handlers/login-command-handler';
import { handleUnifiedModelCommand, handleModelSelectionInHierarchy } from '@/handlers/unified-model-command-handler';
import type { CommandSlashProps, ViewState } from '@/types/command-slash';

interface UseCommandHandlersProps {
  onFeedbackMessage?: CommandSlashProps['onFeedbackMessage'];
  onAddMessage?: CommandSlashProps['onAddMessage'];
  onQueryClientInvalidate?: CommandSlashProps['onQueryClientInvalidate'];
  onClose: CommandSlashProps['onClose'];
  setStatusData: (data: any) => void;
  setHierarchicalModelData: (data: any) => void;
  setHelpData: (data: any) => void;
  goToView: (view: ViewState) => void;
}

export function useCommandHandlers({
  onFeedbackMessage,
  onAddMessage,
  onQueryClientInvalidate,
  onClose,
  setStatusData,
  setHierarchicalModelData,
  setHelpData,
  goToView,
}: UseCommandHandlersProps) {
  // Handle the status command
  const handleStatusCommandSpecial = useCallback(async () => {
    try {
      const statusResult = await handleStatusCommand();

      if (statusResult.statusData) {
        setStatusData({
          providers: statusResult.statusData.providers
        });
        goToView('status');
      } else {
        if (!statusResult.suppressChatMessage) {
          onAddMessage?.(statusResult);
        } else {
          if (statusResult.content.includes("Failed") || statusResult.content.includes("❌")) {
            onFeedbackMessage?.(`Error: ${statusResult.content.replace("❌", "").trim()}`);
          } else if (statusResult.content.includes("✅")) {
            onFeedbackMessage?.(statusResult.content.replace("✅", "").trim());
          }
        }
      }
    } catch (error) {
      console.error('Status command failed:', error);
      onFeedbackMessage?.(`Error: Failed to check authentication status`);
    }
  }, [onAddMessage, onFeedbackMessage, setStatusData, goToView]);


  // Handle the unified model command
  const handleUnifiedModelCommandSpecial = useCallback(async () => {
    try {
      const modelResult = await handleUnifiedModelCommand();

      if (modelResult.hierarchicalModel) {
        setHierarchicalModelData(modelResult.hierarchicalModel);
        goToView('hierarchical-model');
      } else {
        if (!modelResult.suppressChatMessage) {
          onAddMessage?.(modelResult);
        }
      }
    } catch (error) {
      console.error('Model command failed:', error);
      onFeedbackMessage?.(`Error: Failed to load model selection`);
    }
  }, [onAddMessage, onFeedbackMessage, setHierarchicalModelData, goToView]);

  // Handle the help command
  const handleHelpCommandSpecial = useCallback(async () => {
    try {
      const helpData = {
        menuItems: [
          {
            id: 'documentation',
            name: 'Documentation',
            description: 'View Mix documentation',
            action: 'link',
            url: 'https://docs.mix.com'
          },
          {
            id: 'commands',
            name: 'Available Commands',
            description: 'Show list of available slash commands',
            action: 'commands'
          },
          {
            id: 'support',
            name: 'Support',
            description: 'Get help and support',
            action: 'link',
            url: 'https://support.mix.com'
          }
        ]
      };

      setHelpData(helpData);
      goToView('help');
    } catch (error) {
      console.error('Help command failed:', error);
      onFeedbackMessage?.(`Error: Failed to load help menu`);
    }
  }, [setHelpData, goToView, onFeedbackMessage]);

  // Handle provider selection from status view
  const handleProviderSelectionSpecial = useCallback(async (providerId: string) => {
    try {
      const result = await handleProviderSelection(providerId);
      onAddMessage?.(result);
      onClose();
    } catch (error) {
      console.error('Provider selection failed:', error);
      onFeedbackMessage?.(`Error: Failed to select provider`);
    }
  }, [onAddMessage, onClose, onFeedbackMessage]);

  // Handle model selection from unified model view
  const handleModelSelectionSpecial = useCallback(async (providerId: string, modelId: string) => {
    try {
      const result = await handleModelSelectionInHierarchy(providerId, modelId);
      onAddMessage?.(result);
      onClose();
    } catch (error) {
      console.error('Model selection failed:', error);
      onFeedbackMessage?.(`Error: Failed to select model`);
    }
  }, [onAddMessage, onClose, onFeedbackMessage]);


  // Handle login provider selection
  const handleLoginProviderSelectionSpecial = useCallback(async (providerId: string, authMethod: 'api_key' | 'oauth') => {
    try {
      if (authMethod === "oauth") {
        const result = await startOAuthFlow(providerId);

        // OAuth state is handled by the individual auth functions

        onAddMessage?.(result);
      }
      // For API key method, the component will handle showing the input form
    } catch (error) {
      console.error('Login provider selection failed:', error);
      onFeedbackMessage?.(`Error: Failed to start authentication`);
    }
  }, [onAddMessage, onFeedbackMessage]);

  // Handle API key submission
  const handleApiKeySubmitSpecial = useCallback(async (providerId: string, apiKey: string) => {
    try {
      const result = await authenticateWithApiKey(providerId, apiKey);
      onQueryClientInvalidate?.(['providers']);
      onAddMessage?.(result);
      onClose();
    } catch (error) {
      console.error('API key submission failed:', error);
      onFeedbackMessage?.(`Error: Failed to authenticate with API key`);
    }
  }, [onQueryClientInvalidate, onAddMessage, onClose, onFeedbackMessage]);

  // Handle OAuth code submission
  const handleOAuthCodeSubmitSpecial = useCallback(async (providerId: string, code: string, state?: string) => {
    try {
      const result = await handleOAuthCallback(providerId, code, state || '');
      onQueryClientInvalidate?.(['providers']);
      onAddMessage?.(result);
      onClose();
    } catch (error) {
      console.error('OAuth code submission failed:', error);
      onFeedbackMessage?.(`Error: Failed to complete OAuth authentication`);
    }
  }, [onQueryClientInvalidate, onAddMessage, onClose, onFeedbackMessage]);

  return {
    handleStatusCommandSpecial,
    handleUnifiedModelCommandSpecial,
    handleHelpCommandSpecial,
    handleProviderSelectionSpecial,
    handleModelSelectionSpecial,
    handleLoginProviderSelectionSpecial,
    handleApiKeySubmitSpecial,
    handleOAuthCodeSubmitSpecial,
  };
}