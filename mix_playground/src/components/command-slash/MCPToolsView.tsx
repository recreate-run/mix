import { Settings } from "lucide-react";
import {
	CommandEmpty,
	CommandGroup,
	CommandItem,
} from "@/components/ui/command";
import { useMCPList } from "@/hooks/useMCPList";
import { BackButton } from "./shared/BackButton";
import { StatusBadge } from "./shared/StatusBadge";

interface MCPToolsViewProps {
	selectedMCPServer: string;
	onBackToServers: () => void;
}

export function MCPToolsView({
	selectedMCPServer,
	onBackToServers,
}: MCPToolsViewProps) {
	const { data: mcpServers = [] } = useMCPList();

	const selectedServerData = mcpServers.find(
		(s) => s.name === selectedMCPServer,
	);
	const selectedServerTools = selectedServerData?.tools || [];

	if (!selectedServerTools.length) {
		return <CommandEmpty>No tools found</CommandEmpty>;
	}

	return (
		<CommandGroup
			heading={`${selectedMCPServer} Tools (${selectedServerTools.length})`}
		>
			<BackButton
				label="Back to MCP Servers"
				onSelect={onBackToServers}
				value="back-to-mcp"
			/>

			{selectedServerTools.map((tool) => (
				<CommandItem
					className="cursor-default"
					key={tool.name}
					value={tool.name}
				>
					<Settings className="size-4 text-muted-foreground" />
					<div className="flex-1">
						<div className="font-medium text-sm">{tool.name}</div>
						<div className="text-muted-foreground text-xs">
							{tool.description}
						</div>
					</div>
					<StatusBadge
						status={
							selectedServerData?.connected ? "connected" : "disconnected"
						}
					/>
				</CommandItem>
			))}
		</CommandGroup>
	);
}
