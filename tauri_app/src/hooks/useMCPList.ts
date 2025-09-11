import { useQuery } from '@tanstack/react-query';
import { mix } from '@/lib/mix-sdk';
import type { ListMcpServersResponse, Tool } from 'mix-typescript-sdk/models/operations/listmcpservers';

// Use SDK types directly
export type ToolData = Tool;
export type MCPServerData = ListMcpServersResponse;

// Transform SDK response - validate and ensure tools array is never null
const transformMCPServerData = (sdkData: ListMcpServersResponse): MCPServerData => {
  return {
    ...sdkData,
    tools: sdkData.tools ?? [], // Convert null/undefined to empty array
  };
};

const loadMCPList = async (): Promise<MCPServerData[]> => {
  // Let SDK validation errors propagate - don't mask them with type assertions
  const response = await mix.system.listMcpServers();

  // Transform SDK data to ensure tools is never null
  return response.map(transformMCPServerData);
};

export const useMCPList = () => {
  return useQuery({
    queryKey: ['mcp', 'list'],
    queryFn: loadMCPList,
    refetchOnWindowFocus: false,
  });
};
