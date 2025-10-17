import {
	CommandEmpty,
	CommandGroup,
	CommandItem,
} from "@/components/ui/command";
import type { HelpData } from "@/types/command-slash";
import { BackButton } from "./shared/BackButton";

interface HelpMenuViewProps {
	helpData: HelpData;
	onBackToCommands: () => void;
	onClose: () => void;
	onExecuteCommand: (command: string) => void;
}

export function HelpMenuView({
	helpData,
	onBackToCommands,
	onClose,
	onExecuteCommand,
}: HelpMenuViewProps) {
	const handleHelpItemSelect = async (item: (typeof helpData.menuItems)[0]) => {
		if (item.action === "link" && item.url) {
			// Open external link
			try {
				window.open(item.url, "_blank", "noopener,noreferrer");
			} catch (error) {
				console.error("Failed to open link:", error);
			}
			onClose();
		} else if (item.action === "commands") {
			// Show available commands
			onExecuteCommand("help-commands");
		}
	};

	if (!helpData?.menuItems.length) {
		return <CommandEmpty>No help items available</CommandEmpty>;
	}

	return (
		<CommandGroup heading="Help & Documentation">
			<BackButton
				label="Back to Commands"
				onSelect={onBackToCommands}
				value="back-to-commands"
			/>

			{helpData.menuItems.map((item) => (
				<CommandItem
					key={item.id}
					onSelect={() => handleHelpItemSelect(item)}
					value={item.id}
				>
					<div className="flex-1">
						<div className="font-medium text-sm">{item.name}</div>
						<div className="text-muted-foreground text-xs">
							{item.description}
						</div>
					</div>
					{item.action === "link" && (
						<div className="text-muted-foreground text-xs">external</div>
					)}
				</CommandItem>
			))}
		</CommandGroup>
	);
}
