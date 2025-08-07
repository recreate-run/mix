import { AIResponse } from '@/components/ui/kibo-ui/ai/response';

interface McpTool {
  name: string;
  description: string;
}

interface McpServer {
  name: string;
  status: string;
  connected: boolean;
  toolCount: number;
  tools: McpTool[];
}

interface McpData {
  type: string;
  servers: McpServer[];
}

interface McpDisplayProps {
  data: McpData;
}

export function McpDisplay({ data }: McpDisplayProps) {
  // Generate markdown string
  let markdown = '# Available MCP Servers\n\n';

  data.servers.forEach((server) => {
    const statusIcon = server.connected ? '✓' : '✗';
    markdown += `• **${server.name}** ${statusIcon} ${server.status}\n`;

    if (server.connected && server.tools.length > 0) {
      const toolText = server.toolCount === 1 ? 'tool' : 'tools';
      markdown += `  ${server.toolCount} ${toolText} available:\n`;

      server.tools.forEach((tool) => {
        markdown += `    - ${tool.name}\n`;
      });
    } else if (!server.connected) {
      markdown += '  No tools available (connection failed)\n';
    }

    markdown += '\n';
  });

  return <AIResponse>{markdown}</AIResponse>;
}
