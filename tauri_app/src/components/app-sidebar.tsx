"use client"

import * as React from "react"
import {
    IconInnerShadowTop,
    IconClock,
    IconPlus,
} from "@tabler/icons-react"
import { useNavigate } from '@tanstack/react-router'

// import { NavDocuments } from "@/components/nav-documents"
// import { NavMain } from "@/components/nav-main"
// import { NavSecondary } from "@/components/nav-secondary"
import { NavUser } from "@/components/nav-user"
import {
    Sidebar,
    SidebarContent,
    SidebarFooter,
    SidebarHeader,
    SidebarMenu,
    SidebarMenuButton,
    SidebarMenuItem,
    SidebarGroup,
    SidebarGroupLabel,
    SidebarGroupContent,
} from "@/components/ui/sidebar"
import {
    useSelectSession,
    useSessionsList,
} from '@/hooks/useSessionsList'
import { SessionItem } from '@/components/session-item'

interface AppSidebarProps extends React.ComponentProps<typeof Sidebar> {
    sessionId?: string
}

export function AppSidebar({ sessionId, ...props }: AppSidebarProps) {
    const data = {
        user: {
            name: "shadcn",
            email: "m@example.com",
            avatar: "/avatars/shadcn.jpg",
        }
    }
    const navigate = useNavigate()
    const { data: sessions = [], isLoading: sessionsLoading } = useSessionsList()
    const selectSessionMutation = useSelectSession()

    // Sort sessions chronologically (most recent first)
    const sortedSessions = sessions.sort(
        (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
    )

    const handleSessionSelect = (sessionId: string) => {
        selectSessionMutation.mutate(sessionId, {
            onSuccess: () => {
                navigate({ 
                    to: '/$sessionId', 
                    params: { sessionId },
                    replace: true 
                })
            },
        })
    }

    const handleNewSession = () => {
        navigate({ to: '/', replace: true })
    }

    return (
        <Sidebar collapsible="offcanvas" {...props}>
            <SidebarHeader>
                <SidebarMenu>
                    <SidebarMenuItem>
                        <SidebarMenuButton
                            asChild
                            className="data-[slot=sidebar-menu-button]:!p-1.5"
                        >
                            <a href="#">
                                <IconInnerShadowTop className="!size-5" />
                                <span className="text-base font-semibold">Mix</span>
                            </a>
                        </SidebarMenuButton>
                    </SidebarMenuItem>
                </SidebarMenu>
            </SidebarHeader>
            <SidebarContent>
                <SidebarGroup>
                    <SidebarGroupLabel>Sessions</SidebarGroupLabel>
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
                                    const isActive = sessionId === session.id
                                    return (
                                        <SessionItem
                                            key={session.id}
                                            session={session}
                                            isActive={isActive}
                                            onClick={handleSessionSelect}
                                        />
                                    )
                                })
                            )}
                        </SidebarMenu>
                    </SidebarGroupContent>
                </SidebarGroup>
            </SidebarContent>
            <SidebarFooter>
                <NavUser user={data.user} />
            </SidebarFooter>
        </Sidebar>
    )
}
