import { IconEdit } from '@tabler/icons-react';
import { useCreateSession } from '@/hooks/useSession';
import type { Session } from '@/types/common';

interface SessionHeaderProps {
  onNewSession?: () => void;
  currentSession?: Session | null;
}

export function SessionHeader({
  onNewSession,
  currentSession,
}: SessionHeaderProps) {
  const createSession = useCreateSession();

  const handleNewSession = async () => {
    try {
      // NOTE: currentSession?.workingDirectory should always be defined here because:
      // 1. New users must select a project before accessing chat (enforced by routing)
      // 2. Backend now requires working directory for all session creation
      // 3. If currentSession is null/undefined, this will fail gracefully with backend validation
      await createSession.mutateAsync({
        title: 'Chat Session',
        workingDirectory: currentSession?.workingDirectory,
      });
      onNewSession?.();
    } catch (error) {
      console.error('Failed to create new session:', error);
    }
  };

  return (
    <div className="flex justify-end">
      <button
        className="flex items-center gap-2 rounded-lg font-medium text-sm text-stone-500 transition-colors hover:bg-stone-700/50 hover:text-stone-100 disabled:cursor-not-allowed disabled:opacity-50"
        disabled={createSession.isPending}
        onClick={handleNewSession}
        title="Start New Session"
        type="button"
      >
        <IconEdit className="size-5" />
      </button>
    </div>
  );
}
