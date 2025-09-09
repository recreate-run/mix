import { useQuery } from '@tanstack/react-query';
import { mix } from '@/lib/mix-sdk';
import type { ListMcpServersData } from 'mix-typescript-sdk/models/operations';

export interface ToolData {
  name: string;
  description: string;
}

// Keep our correct interface that matches what backend actually returns
export interface MCPServerData {
  name: string;
  connected: boolean;
  status: string;
  tools: ToolData[];
}

// Transform SDK response to our expected interface
const transformMCPServerData = (sdkData: ListMcpServersData): MCPServerData => {
  if (!sdkData.name || typeof sdkData.name !== 'string') {
    throw new Error('Invalid MCP server data: name is required and must be a string');
  }

  if (!sdkData.status || typeof sdkData.status !== 'string') {
    throw new Error('Invalid MCP server data: status is required and must be a string');
  }

  return {
    name: sdkData.name,
    connected: sdkData.connected ?? false,
    status: sdkData.status,
    tools: sdkData.tools ?? [],
  };
};

const loadMCPList = async (): Promise<MCPServerData[]> => {
  // Let SDK validation errors propagate - don't mask them with type assertions
  const response = await mix.system.listMcpServers();

  if (response.error) {
    throw new Error(response.error.message || 'Failed to load MCP servers');
  }

  if (!response.data) {
    throw new Error('No MCP server data returned from server');
  }

  // Transform SDK data to our expected interface - fail fast if schema doesn't match
  return response.data.map(transformMCPServerData);
};

export const useMCPList = () => {
  return useQuery({
    queryKey: ['mcp', 'list'],
    queryFn: loadMCPList,
    refetchOnWindowFocus: false,
  });
};
