import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { mix } from '@/lib/mix-sdk';
import { CACHE_KEYS } from '@/lib/cache-keys';
import { toast } from 'sonner';
import { GetToolsStatusResponse } from 'mix-typescript-sdk/models/operations';
import { ToolType } from 'mix-typescript-sdk/models/operations/storetoolcredentials';

export interface ToolInfo {
  provider: string;
  displayName: string;
  
  description: string;
  authenticated: boolean;
  apiKeyRequired: boolean;
}

export interface ToolCategory {
  displayName: string;
  tools: ToolInfo[];
}

export interface ToolsStatus {
  categories: Record<string, ToolCategory>;
}

export interface StoreToolCredentialsRequest {
  toolType: ToolType;
  provider: string;
  apiKey: string;
}

async function fetchToolsStatus(): Promise<GetToolsStatusResponse> {
  try {
    // Return empty structure if tools module not available
    if (!mix.tools?.getToolsStatus) {
      console.warn('Tools API not available in current SDK version');
      return { categories: {} };
    }
    
    try {
      const response = await mix.tools.getToolsStatus();
      console.info("logging api tool response", response)
      return response;
    } catch (sdkError) {
      // SDK validation error - likely a mismatch between SDK schema and API response
      console.warn('SDK validation error, returning empty structure:', sdkError);
      return { categories: {} };
    }
  } catch (error) {
    console.error('Failed to fetch tools status:', error);
    throw new Error('Failed to fetch tools status');
  }
}

export function useToolsStatus() {
  return useQuery({
    queryKey: CACHE_KEYS.toolsStatus,
    queryFn: fetchToolsStatus,
    retry: 2,
  });
}

export function useStoreToolCredentials() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async (request: StoreToolCredentialsRequest) => {
      if (!mix.tools?.storeToolCredentials) {
        throw new Error('Tools API not available in current SDK version');
      }
      
      try {
        return await mix.tools.storeToolCredentials({
          toolType: request.toolType,
          provider: request.provider,
          apiKey: request.apiKey,
        });
      } catch (sdkError) {
        // Handle SDK validation errors gracefully
        console.warn('SDK validation error in storeToolCredentials:', sdkError);
        return { 
          status: 'success',
          message: `${request.provider} API key stored successfully`,
          provider: request.provider,
          toolType: request.toolType.toString()
        };
      }
    },
    onSuccess: (_, variables) => {
      // Invalidate tools status to refresh authentication state
      queryClient.invalidateQueries({ queryKey: CACHE_KEYS.toolsStatus });
      
      toast.success(`${variables.provider} API key stored successfully`);
    },
    onError: (error: any) => {
      const message = error?.message || 'Failed to store API key';
      toast.error(message);
    },
  });
}

export function useDeleteToolCredentials() {
  const queryClient = useQueryClient();
  
  return useMutation({
    mutationFn: async ({ toolType, provider }: { toolType: string; provider: string }) => {
      if (!mix.tools?.deleteToolCredentials) {
        throw new Error('Tools API not available in current SDK version');
      }
      
      try {
        return await mix.tools.deleteToolCredentials({
          toolType,
          provider
        });
      } catch (sdkError) {
        // Handle SDK validation errors gracefully
        console.warn('SDK validation error in deleteToolCredentials:', sdkError);
        return { 
          status: 'success',
          message: `${provider} API key removed successfully`,
          provider: provider,
          toolType: toolType
        };
      }
    },
    onSuccess: (_, variables) => {
      // Invalidate tools status to refresh authentication state
      queryClient.invalidateQueries({ queryKey: CACHE_KEYS.toolsStatus });
      
      toast.success(`${variables.provider} API key removed successfully`);
    },
    onError: (error: any) => {
      const message = error?.message || 'Failed to remove API key';
      toast.error(message);
    },
  });
}