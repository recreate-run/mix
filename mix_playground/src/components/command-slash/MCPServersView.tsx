import { Plug } from 'lucide-react';
import { CommandEmpty, CommandGroup, CommandItem } from '@/components/ui/command';
import { useMCPList } from '@/hooks/useMCPList';
import { BackButton } from './shared/BackButton';
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
          <CommandItem
            key={server.name}
            onSelect={() => handleServerSelect(server.name)}
            value={server.name}
          >
            <Plug className="size-4 text-muted-foreground" />
            <div className="flex-1">
              <div className="flex items-center gap-2 font-medium text-sm">
                {server.name}
                <StatusBadge
                  status={server.connected ? 'connected' : 'disconnected'}
                  label={server.status}
                />
              </div>
              <div className="text-muted-foreground text-xs">
                {server.tools?.length || 0} tools available
              </div>
            </div>
          </CommandItem>
        ))}
    </CommandGroup>
  );
}