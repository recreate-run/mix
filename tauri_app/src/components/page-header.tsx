import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { IconGitBranch } from "@tabler/icons-react";
import { FolderIcon } from 'lucide-react';

interface PageHeaderProps {
    sessionId: string;
    selectedFolder?: string;
    defaultWorkingDir: string;
    onFolderSelect: () => void;
}



export function PageHeader({ sessionId, selectedFolder, defaultWorkingDir, onFolderSelect }: PageHeaderProps) {
    return (
        <header className="sticky top-0 z-50 flex justify-between items-center px-8 h-11 bg-sidebar border-b border-sidebar-border" data-tauri-drag-region="true">
            <div className="flex items-center gap-4">
                <SidebarTrigger className="-ml-2" />

                <div className="rounded pl-4 font-mono text-stone-400 text-xs">
                    Session: {sessionId?.slice(0, 8) || 'Loading...'}
                </div>


            </div>

            <div className="flex items-center gap-3">

                <Button
                    onClick={onFolderSelect}
                    className="font-extralight  rounded-l hover:bg-muted/50 transition-colors"
                    variant="outline" size="sm">
                    <FolderIcon className="size-3 stroke-1" /> Open
                </Button>


            </div>
        </header>
    )
}
