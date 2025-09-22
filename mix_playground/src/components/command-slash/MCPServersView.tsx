import { Plug } from 'lucide-react';
import { CommandEmpty, CommandGroup } from '@/components/ui/command';
import { useMCPList } from '@/hooks/useMCPList';
import { BackButton } from './shared/BackButton';
import { CommandItemWrapper } from './shared/CommandItemWrapper';
import { StatusBadge } from './shared/StatusBadge';

interface MCPServersViewProps {
  onBackToCommands: () => void;
  onServerSelect: (serverName: string) => void;
}

export function MCPServersView({
  onBackToCommands,
  onServerSelect,
}: MCPServersViewProps) {
  const { data: mcpServers = [], isLoading: mcpLoading } = useMCPList();

  const handleServerSelect = (serverName: string) => {
    onServerSelect(serverName);
  };

  if (mcpLoading) {
    return <CommandEmpty>Loading MCP servers...</CommandEmpty>;
  }

  if (!mcpServers.length) {
    return <CommandEmpty>No MCP servers found</CommandEmpty>;
  }

  return (
    <CommandGroup heading={`MCP Servers (${mcpServers.length})`}>
      <BackButton
        label="Back to Commands"
        onSelect={onBackToCommands}
        value="back-to-commands"
      />

        {mcpServers.map((server) => (
        <CommandItemWrapper
          key={server.name}
          id={server.name}
          value={server.name}
          onSelect={handleServerSelect}
          icon={Plug}
          title={server.name}
          description={`${server.tools?.length || 0} tools available`}
          badge={
            <StatusBadge
              status={server.connected ? 'connected' : 'disconnected'}
              label={server.status}
            />
          }
        />
        ))}
    </CommandGroup>
  );
}