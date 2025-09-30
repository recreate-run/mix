import { useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';
import { getBackendUrl } from '@/utils/backendUrl';

interface ExportSessionOptions {
  sessionId: string;
  sessionTitle?: string;
}

/**
 * Hook to export a session's complete transcript as a JSON file
 */
export function useSessionExport() {
  return useMutation({
    mutationFn: async ({ sessionId, sessionTitle }: ExportSessionOptions) => {
      const backendUrl = getBackendUrl();
      const url = `${backendUrl}/api/sessions/${sessionId}/export`;

      const response = await fetch(url, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      });

      if (!response.ok) {
        throw new Error(`Failed to export session: ${response.statusText}`);
      }

      // Get the JSON data
      const data = await response.json();

      // Create a blob and trigger download
      const blob = new Blob([JSON.stringify(data, null, 2)], {
        type: 'application/json',
      });
      const downloadUrl = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = downloadUrl;

      // Use session title for filename if available, otherwise use session ID
      const sanitizedTitle = sessionTitle
        ? sessionTitle.replace(/[^a-z0-9]/gi, '_').toLowerCase()
        : sessionId;
      a.download = `session_${sanitizedTitle}_transcript.json`;

      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(downloadUrl);

      return data;
    },
    onSuccess: () => {
      toast.success('Session exported successfully');
    },
    onError: (error: Error) => {
      toast.error('Export failed', {
        description: error.message,
      });
    },
  });
}