import { IconClock, IconPlus, IconSettings } from "@tabler/icons-react";
import { useNavigate } from "@tanstack/react-router";
import type * as React from "react";
import { useState } from "react";
import { SessionItem } from "@/components/session-item";
import { SettingsDialog } from "@/components/settings-dialog";
import {
	Sidebar,
	SidebarContent,
	SidebarFooter,
	SidebarGroup,
	SidebarGroupContent,
	SidebarGroupLabel,
	SidebarMenu,
	SidebarMenuButton,
	SidebarMenuItem,
	SidebarTrigger,
} from "@/components/ui/sidebar";
import { useCreateSession } from "@/hooks/useSession";
import { useSessionsList } from "@/hooks/useSessionsList";

interface AppSidebarProps extends React.ComponentProps<typeof Sidebar> {
	sessionId?: string;
}

export function AppSidebar({ sessionId, ...props }: AppSidebarProps) {
	const navigate = useNavigate();
	const { data: sessions = [], isLoading: sessionsLoading } = useSessionsList();
	const createSession = useCreateSession();
	const [settingsOpen, setSettingsOpen] = useState(false);

	// Sort sessions chronologically (most recent first)
	const sortedSessions = sessions.sort(
		(a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime(),
	);

	const handleSessionSelect = (sessionId: string) => {
		// Navigate directly to the selected session (stateless design)
		navigate({
			to: "/$sessionId",
			params: { sessionId },
			// Remove replace: true to prevent full route replacement
		});
	};

	const handleNewSession = async () => {
		try {
			// Create a new session
			const newSession = await createSession.mutateAsync({
				title: "Chat Session",
			});
			navigate({
				to: "/$sessionId",
				params: { sessionId: newSession.id },
				// Remove replace: true for consistency
			});
		} catch (error) {
			console.error("Failed to create new session:", error);
		}
	};

	return (
		<>
			<SettingsDialog onOpenChange={setSettingsOpen} open={settingsOpen} />
			<Sidebar collapsible="offcanvas" {...props}>
				<SidebarContent>
					<SidebarGroup>
						<div className="flex items-center justify-between">
							<SidebarGroupLabel>Sessions</SidebarGroupLabel>
							<SidebarTrigger className="size-6" />
						</div>
						<SidebarGroupContent>
							<SidebarMenu>
								{/* New Session Button */}
								<SidebarMenuItem>
									<SidebarMenuButton onClick={handleNewSession}>
										<IconPlus className="size-4" />
										<span>New Session</span>
									</SidebarMenuButton>
								</SidebarMenuItem>

								{/* Sessions List */}
								{sessionsLoading ? (
									<SidebarMenuItem>
										<SidebarMenuButton disabled>
											<IconClock className="size-4" />
											<span>Loading sessions...</span>
										</SidebarMenuButton>
									</SidebarMenuItem>
								) : sortedSessions.length === 0 ? (
									<SidebarMenuItem>
										<SidebarMenuButton disabled>
											<IconClock className="size-4" />
											<span>No sessions</span>
										</SidebarMenuButton>
									</SidebarMenuItem>
								) : (
									sortedSessions.map((session) => {
										const isActive = sessionId === session.id;
										return (
											<SessionItem
												allSessions={sortedSessions}
												currentSessionId={sessionId}
												isActive={isActive}
												key={session.id}
												onClick={handleSessionSelect}
												session={session}
											/>
										);
									})
								)}
							</SidebarMenu>
						</SidebarGroupContent>
					</SidebarGroup>
				</SidebarContent>
				<SidebarFooter>
					<SidebarMenu>
						<SidebarMenuItem>
							<SidebarMenuButton onClick={() => setSettingsOpen(true)}>
								<IconSettings className="size-4" />
								<span>Settings</span>
							</SidebarMenuButton>
						</SidebarMenuItem>
					</SidebarMenu>
				</SidebarFooter>
			</Sidebar>
		</>
	);
}
