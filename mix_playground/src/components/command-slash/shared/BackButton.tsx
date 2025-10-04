import { ArrowLeft } from "lucide-react";
import { CommandItem } from "@/components/ui/command";

interface BackButtonProps {
	label: string;
	onSelect: () => void;
	value?: string;
}

export function BackButton({
	label,
	onSelect,
	value = "back",
}: BackButtonProps) {
	return (
		<CommandItem onSelect={onSelect} value={value}>
			<ArrowLeft className="size-4 text-muted-foreground" />
			<div className="flex-1">
				<div className="font-medium text-sm">{label}</div>
			</div>
		</CommandItem>
	);
}
