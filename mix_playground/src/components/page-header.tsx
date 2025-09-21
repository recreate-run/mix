import { SidebarTrigger } from '@/components/ui/sidebar';

interface PageHeaderProps {
  sessionId: string;
}

export function PageHeader({ sessionId }: PageHeaderProps) {
  return (
    <header
      className="sticky top-0 z-50 flex h-11 items-center justify-between border-sidebar-border border-b bg-sidebar px-8"
      data-tauri-drag-region="true"
    >
      <div className="flex items-center gap-4">
        <SidebarTrigger className="-ml-2" />

        <div className="rounded pl-4 font-mono text-stone-400 text-xs">
          Session: {sessionId?.slice(0, 8) || 'Loading...'}
        </div>
      </div>
    </header>
  );
}
