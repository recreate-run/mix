import { useQuery } from '@tanstack/react-query';
import type { SupportedFileTypes } from '@/utils/fileTypes';

async function fetchFileTypes(): Promise<SupportedFileTypes> {
  const response = await fetch(`${import.meta.env.VITE_BACKEND_URL}/api/file-types`);

  if (!response.ok) {
    throw new Error(`Failed to fetch file types: ${response.status}`);
  }

  return response.json();
}

export function useFileTypes() {
  return useQuery({
    queryKey: ['fileTypes'],
    queryFn: fetchFileTypes,
    retry: 3,
    staleTime: Infinity
  });
}