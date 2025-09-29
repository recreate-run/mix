import { useQuery } from '@tanstack/react-query';
import { mix } from '@/lib/mix-sdk';
import type { ListMcpServersResponse } from 'mix-typescript-sdk/models/operations/listmcpservers';

// Use SDK types directly
type MCPServerData = ListMcpServersResponse;

// Transform SDK response - validate and ensure tools array is never null
function transformMCPServerData(sdkData: ListMcpServersResponse): MCPServerData {
  return {
    ...sdkData,
    tools: sdkData.tools ?? [], // Convert null/undefined to empty array
  };
}

async function loadMCPList(): Promise<MCPServerData[]> {
  // Let SDK validation errors propagate - don't mask them with type assertions
  const response = await mix.system.listMcpServers();

  // Transform SDK data to ensure tools is never null
  return response.map(transformMCPServerData);
}

export function useMCPList() {
  return useQuery({
    queryKey: ['mcp', 'list'],
    queryFn: loadMCPList,
    refetchOnWindowFocus: false,
  });
}
