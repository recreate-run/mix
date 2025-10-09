import { IconFolder, IconTrash } from "@tabler/icons-react";
import { useNavigate } from "@tanstack/react-router";
import { useState } from "react";
import {
	AlertDialog,
	AlertDialogAction,
	AlertDialogCancel,
	AlertDialogContent,
	AlertDialogDescription,
	AlertDialogFooter,
	AlertDialogHeader,
	AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar";
import { useDeleteSession } from "@/hooks/useSession";
import { useSystemInfo } from "@/hooks/useSystemInfo";
import type { SessionData } from "@/types/common";
import { PlatformFeatures } from "@/utils/platform";
import { getDisplayTitle } from "@/utils/sessionUtils";

interface SessionItemProps {
	session: SessionData;
	isActive: boolean;
	onClick: (sessionId: string) => void;
	currentSessionId?: string;
	allSessions: SessionData[];
}

export function SessionItem({
	session,
	isActive,
	onClick,
	currentSessionId,
	allSessions,
}: SessionItemProps) {
	const navigate = useNavigate();
	const { data: systemInfo } = useSystemInfo();
	const [showDeleteDialog, setShowDeleteDialog] = useState(false);

	// Simple delete hook with navigation callback - no circular dependencies
	const deleteSessionMutation = useDeleteSession({
		allSessions,
		currentSessionId,
		onNavigate: (nextSessionId) => {
			if (nextSessionId) {
				navigate({
					to: "/$sessionId",
					params: { sessionId: nextSessionId },
					replace: true,
				});
			} else {
				navigate({ to: "/", replace: true });
			}
		},
	});

	const handleDelete = (e: React.MouseEvent) => {
		e.stopPropagation();
		if (deleteSessionMutation.isPending || session.isDeleting) return;
		setShowDeleteDialog(true);
	};

	const confirmDelete = async () => {
		try {
			await deleteSessionMutation.mutateAsync(session.id);
			setShowDeleteDialog(false);
		} catch (error) {
			console.error("Failed to delete session:", error);
		}
	};

	const handleOpenFolder = async (e: React.MouseEvent) => {
		e.stopPropagation();

		if (!PlatformFeatures.hasShellAccess()) {
			console.error("Open folder feature is only available in desktop app");
			return;
		}

		if (!systemInfo?.storageBasePath) {
			console.error("Storage base path not available");
			return;
		}

		try {
			// Desktop: Open the session's storage directory in system file manager
			const { open } = await import("@tauri-apps/plugin-shell");
			const storagePath = `${systemInfo.storageBasePath}/${session.id}`;
			console.log("Opening storage path:", storagePath);
			await open(storagePath);
		} catch (error) {
			console.error("Failed to open storage folder:", error);
		}
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
		<>
			<SidebarMenuItem
				className={`group/session-item overflow-hidden ${
					session.isDeleting
						? "pointer-events-none cursor-not-allowed opacity-50"
						: ""
				}`}
			>
				<div
					className={`flex translate-x-0 transition-transform duration-200 ease-out will-change-transform ${
						PlatformFeatures.hasShellAccess()
							? "group-hover/session-item:translate-x-[-80px]"
							: "group-hover/session-item:translate-x-[-40px]"
					}`}
				>
					<SidebarMenuButton
						className="flex h-auto min-h-[60px] w-full flex-shrink-0 flex-col items-start gap-1 py-2 pr-2 hover:bg-transparent"
						isActive={isActive}
						onClick={() => !session.isDeleting && onClick(session.id)}
					>
						<div className="flex w-full items-center gap-2">
							<span className="flex-1 truncate font-medium text-sm">
								{getDisplayTitle(session)}
							</span>
						</div>
						<div className="flex items-center gap-2 text-muted-foreground text-xs">
							<span>{formatDate(createdDate)}</span>
						</div>
					</SidebarMenuButton>
					{PlatformFeatures.hasShellAccess() && (
						<Button
							className="flex min-h-[60px] flex-shrink-0 cursor-pointer items-center justify-center bg-blue-500 hover:bg-blue-600"
							onClick={handleOpenFolder}
							size="icon"
							title="Open storage folder"
						>
							<IconFolder />
						</Button>
					)}
					<Button
						className="flex min-h-[60px] flex-shrink-0 cursor-pointer items-center justify-center bg-red-500 hover:bg-red-500"
						disabled={deleteSessionMutation.isPending || session.isDeleting}
						onClick={handleDelete}
						size="icon"
						title="Delete session"
					>
						<IconTrash />
					</Button>
				</div>
			</SidebarMenuItem>

			{/* Delete Confirmation Dialog */}
			<AlertDialog onOpenChange={setShowDeleteDialog} open={showDeleteDialog}>
				<AlertDialogContent>
					<AlertDialogHeader>
						<AlertDialogTitle>Delete Session</AlertDialogTitle>
						<AlertDialogDescription>
							Are you sure you want to delete the session "
							{getDisplayTitle(session)}"? This action cannot be undone.
						</AlertDialogDescription>
					</AlertDialogHeader>
					<AlertDialogFooter>
						<AlertDialogCancel>Cancel</AlertDialogCancel>
						<AlertDialogAction
							className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
							onClick={confirmDelete}
						>
							Delete
						</AlertDialogAction>
					</AlertDialogFooter>
				</AlertDialogContent>
			</AlertDialog>
		</>
	);
}
