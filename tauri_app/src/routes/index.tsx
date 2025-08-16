import { createFileRoute, redirect } from '@tanstack/react-router'
import { getDefaultWorkingDir } from '@/utils/defaultWorkingDir';
import { rpcCall } from '@/lib/rpc';

export const Route = createFileRoute('/')({
  beforeLoad: async () => {
    const defaultWorkingDir = await getDefaultWorkingDir();
    const result = await rpcCall<{ id: string }>('sessions.create', {
      title: 'New Session',
      workingDirectory: defaultWorkingDir,
    });
    const sessionId = typeof result === 'string' ? result : result?.id;
    
    throw redirect({
      to: '/$sessionId',
      params: { sessionId },
      replace: true,
    });
  },
})