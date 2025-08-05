import { useQuery } from '@tanstack/react-query';
import { invoke } from '@tauri-apps/api/core';

export interface OpenApp {
  name: string;
  icon_png_base64: string;
}

export const fetchVisibleApps = async (): Promise<OpenApp[]> => {
  try {
    const apps = await invoke<OpenApp[]>('list_apps_with_icons');
    return apps;
  } catch (error) {
    throw error;
  }
};

export function useOpenApps() {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['openApps'],
    queryFn: fetchVisibleApps,
    enabled: false, // Don't auto-fetch on mount - only fetch when explicitly requested
    staleTime: Infinity, // Never consider data stale - only refetch when explicitly requested
    refetchInterval: false, // Disable continuous polling - CRITICAL FIX for memory leak
    refetchOnWindowFocus: false, // Don't refetch on window focus
    refetchOnReconnect: false, // Don't refetch on network reconnect
  });

  return {
    apps: data ?? [],
    isLoading,
    error: error?.message ?? null,
    refreshApps: refetch
  };
}