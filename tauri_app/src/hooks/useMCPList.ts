import { useQuery } from '@tanstack/react-query';
import { rpcCall } from '@/lib/rpc';

export interface ToolData {
  name: string;
  description: string;
}

export interface MCPServerData {
  name: string;
  connected: boolean;
  status: string;
  tools: ToolData[];
}

const loadMCPList = async (): Promise<MCPServerData[]> => {
  const result = await rpcCall<MCPServerData[]>('mcp.list', {});
  return result || [];
};

export const useMCPList = () => {
  return useQuery({
    queryKey: ['mcp', 'list'],
    queryFn: loadMCPList,
    refetchOnWindowFocus: false,
  });
};
