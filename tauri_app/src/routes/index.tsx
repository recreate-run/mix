import { createFileRoute, redirect } from '@tanstack/react-router'
import { getDefaultWorkingDir } from '@/utils/defaultWorkingDir';
import { rpcCall } from '@/lib/rpc';

export const Route = createFileRoute('/')({
  beforeLoad: async () => {
    const defaultWorkingDir = await getDefaultWorkingDir();
    const result = await rpcCall('sessions.create', {
      title: 'New Session',
      workingDirectory: defaultWorkingDir,
    });
    const sessionId = result?.id || result;
    
    throw redirect({
      to: '/$sessionId',
      params: { sessionId },
      replace: true,
    });
  },
})