import { IconTrash } from "@tabler/icons-react";
import { SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar";
import {
	TITLE_TRUNCATE_LENGTH,
	type SessionData,
	useDeleteSession,
} from "@/hooks/useSessionsList";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { ask } from "@tauri-apps/plugin-dialog";

interface SessionItemProps {
	session: SessionData;
	isActive: boolean;
	onClick: (sessionId: string) => void;
}

export function SessionItem({ session, isActive, onClick }: SessionItemProps) {
	const [isHovered, setIsHovered] = useState(false);
	const deleteSessionMutation = useDeleteSession();

	const handleDelete = async (e: React.MouseEvent) => {
		e.stopPropagation();
		if (deleteSessionMutation.isPending) return;
		if (
			await ask(
				`Are you sure you want to delete the session "${getDisplayTitle(session)}"? This action cannot be undone.`,
			)
		) {
			deleteSessionMutation.mutate(session.id);
		}
	};

	// Helper function to get display title (copied from command-slash.tsx)
	const getDisplayTitle = (session: SessionData) => {
		if (!session.firstUserMessage || session.firstUserMessage.trim() === "") {
			return session.title;
		}

		let displayText = session.firstUserMessage;
		try {
			const parsed = JSON.parse(session.firstUserMessage);
			const textPart = parsed.find((part: any) => part.type === "text");
			if (textPart?.data?.text) {
				displayText = textPart.data.text;
			}
		} catch {
			displayText = session.firstUserMessage;
		}

		const truncated =
			displayText.length > TITLE_TRUNCATE_LENGTH
				? `${displayText.substring(0, TITLE_TRUNCATE_LENGTH)}...`
				: displayText;

		return truncated;
	};

	const formatDate = (date: Date) => {
		const now = new Date();
		const diffDays = Math.floor(
			(now.getTime() - date.getTime()) / (1000 * 60 * 60 * 24),
		);

		if (diffDays === 0) return "Today";
		if (diffDays === 1) return "Yesterday";
		if (diffDays < 7) return `${diffDays}d ago`;
		return date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
	};

	const createdDate = new Date(session.createdAt);

	return (
		<SidebarMenuItem>
			<div
				className="relative group"
				onMouseEnter={() => setIsHovered(true)}
				onMouseLeave={() => setIsHovered(false)}
			>
				<SidebarMenuButton
					onClick={() => onClick(session.id)}
					isActive={isActive}
					className="flex flex-col items-start gap-1 h-auto py-2 pr-8"
				>
					<div className="flex items-center gap-2 w-full">
						<span className="text-sm font-medium truncate flex-1">
							{getDisplayTitle(session)}
						</span>
					</div>
					<div className="flex items-center gap-2 text-muted-foreground text-xs">
						<span>{formatDate(createdDate)}</span>
						<span>•</span>
						<span>{session.messageCount} messages</span>
					</div>
				</SidebarMenuButton>
				{isHovered && !isActive && (
					<Button
						variant="ghost"
						size="sm"
						onClick={handleDelete}
						disabled={deleteSessionMutation.isPending}
						className="absolute right-1 top-1/2 -translate-y-1/2 opacity-70 hover:opacity-100"
					>
						<IconTrash className="size-4" />
					</Button>
				)}
			</div>
		</SidebarMenuItem>
	);
}
