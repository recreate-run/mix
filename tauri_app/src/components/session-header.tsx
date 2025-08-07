import { IconEdit } from '@tabler/icons-react';
import { useCreateSession } from '@/hooks/useSession';

interface SessionHeaderProps {
  onNewSession?: () => void;
}

export function SessionHeader({ onNewSession }: SessionHeaderProps) {
  const createSession = useCreateSession();

  const handleNewSession = async () => {
    try {
      await createSession.mutateAsync({ title: 'Chat Session' });
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
      >
        <IconEdit className="size-5" />
      </button>
    </div>
  );
}
