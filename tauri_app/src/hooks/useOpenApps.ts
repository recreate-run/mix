import { useQuery, useQueryClient } from '@tanstack/react-query';
import { invoke } from '@tauri-apps/api/core';
import { listen } from '@tauri-apps/api/event';
import { useEffect } from 'react';

export interface AppMetadata {
  name: string;
  bundle_id: string;
  path: string;
}

export const fetchAppList = async (): Promise<AppMetadata[]> => {
  try {
    const apps = await invoke<AppMetadata[]>('get_app_list');
    return apps;
  } catch (error) {
    throw error;
  }
};

export function useAppList() {
  const queryClient = useQueryClient();
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['appList'],
    queryFn: fetchAppList,
    staleTime: 5 * 60 * 1000, // Cache for 5 minutes
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });

  // Listen for real-time app changes from NSWorkspace notifications
  useEffect(() => {
    let unlistenFn: (() => void) | null = null;

    const setupListener = async () => {
      unlistenFn = await listen('app-list-changed', () => {
        // Invalidate and refetch app list when apps launch/terminate
        queryClient.invalidateQueries({ queryKey: ['appList'] });
      });
    };

    setupListener();

    return () => {
      if (unlistenFn) {
        unlistenFn();
      }
    };
  }, [queryClient]);

  return {
    apps: data ?? [],
    isLoading,
    error: error?.message ?? null,
    refreshApps: refetch
  };
}

export function useAppIcon(bundleId: string | null) {
  const { data, isLoading, error } = useQuery({
    queryKey: ['appIcon', bundleId],
    queryFn: async () => {
      if (!bundleId) return null;
      return await invoke<string>('get_app_icon', { bundleId });
    },
    enabled: !!bundleId,
    staleTime: Infinity, // Icons don't change
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });

  return {
    iconBase64: data ?? null,
    isLoading: isLoading && !!bundleId,
    error: error?.message ?? null
  };
}

// Legacy export for backward compatibility
export function useOpenApps() {
  return useAppList();
}