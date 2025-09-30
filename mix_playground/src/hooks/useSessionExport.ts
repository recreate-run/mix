import { useMutation } from '@tanstack/react-query';
import { toast } from 'sonner';
import { save } from '@tauri-apps/plugin-dialog';
import { writeTextFile } from '@tauri-apps/plugin-fs';
import { downloadDir } from '@tauri-apps/api/path';
import { mix } from '@/lib/mix-sdk';

interface ExportSessionOptions {
  sessionId: string;
  sessionTitle?: string;
}

/**
 * Hook to export a session's complete transcript as a JSON file
 * Shows native save dialog with pre-filled filename
 */
export function useSessionExport() {
  return useMutation<unknown, Error, ExportSessionOptions>({
    mutationFn: async ({ sessionId }) => {
      // Use the SDK's export method - returns ExportSessionResponse with result property
      const response = await mix.sessions.exportSession({ id: sessionId });
      const data = response.result;

      // Create filename with session ID and timestamp
      const timestamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, -5); // Format: YYYY-MM-DDTHH-MM-SS
      const filename = `${sessionId}_${timestamp}.json`;

      // Get Downloads directory path
      const downloadsPath = await downloadDir();
      const defaultPath = `${downloadsPath}${filename}`;

      // Show native save dialog with pre-filled path
      const filePath = await save({
        defaultPath: defaultPath
      });

      if (!filePath) {
        throw new Error('Save cancelled');
      }

      // Write the file using Tauri's file system API
      await writeTextFile(filePath, JSON.stringify(data, null, 2));

      return { filePath, data };
    },
    onSuccess: () => {
      toast.success('Session exported to Downloads folder');
    },
    onError: (error) => {
      toast.error('Export failed', {
        description: error.message,
      });
    },
  });
}