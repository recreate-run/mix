import { open } from '@tauri-apps/plugin-dialog';
import { readFile } from '@tauri-apps/plugin-fs';
import { Paperclip, Loader2 } from 'lucide-react';
import { Button } from '@/components/ui/button';
import { useFileUpload } from '@/hooks/useFileUpload';
import { useBoundStore } from '@/stores';
import { cn } from '@/lib/utils';
import { getFileTypeFromExtension, IMAGE_EXTENSIONS, VIDEO_EXTENSIONS, AUDIO_EXTENSIONS } from '@/utils/fileTypes';

interface FileUploadButtonProps {
  sessionId: string;
  className?: string;
  onUploadSuccess?: (fileName: string) => void;
  onUploadError?: (error: string) => void;
}

export function FileUploadButton({ 
  sessionId, 
  className,
  onUploadSuccess,
  onUploadError 
}: FileUploadButtonProps) {
  const fileUpload = useFileUpload();
  const addAttachment = useBoundStore((state) => state.addAttachment);

  const handleFileSelect = async () => {
    try {
      // Open native file picker dialog
      const selected = await open({
        multiple: true,
        filters: [
          {
            name: 'All Files',
            extensions: ['*']
          },
          {
            name: 'Images',
            extensions: [...IMAGE_EXTENSIONS]
          },
          {
            name: 'Videos',
            extensions: [...VIDEO_EXTENSIONS]
          },
          {
            name: 'Audio',
            extensions: [...AUDIO_EXTENSIONS]
          },
          {
            name: 'Documents',
            extensions: ['pdf', 'doc', 'docx', 'txt', 'md', 'rtf', 'csv', 'xls', 'xlsx']
          }
        ]
      });

      if (!selected) {
        return; // User cancelled
      }

      const filePaths = Array.isArray(selected) ? selected : [selected];

      // Process each selected file
      for (const filePath of filePaths) {
        try {
          // Read file data using Tauri FS plugin
          const fileData = await readFile(filePath);
          
          // Extract filename from path
          const fileName = filePath.split(/[/\\]/).pop() || 'unnamed-file';
          
          // Create File object for upload
          const file = new File([fileData], fileName, {
            type: getMimeType(fileName)
          });

          // Upload file using our hook
          const result = await fileUpload.mutateAsync({
            sessionId,
            file
          });

          // Add to attachment store for UI preview
          addAttachment({
            id: `file:${result.name}`,
            name: result.name,
            type: getFileTypeFromExtension(result.name),
            path: `/api/sessions/${sessionId}/files/${result.name}`,
            extension: result.name.split('.').pop(),
            isDirectory: false,
          });

          // Call success callback
          onUploadSuccess?.(result.name);

        } catch (error) {
          console.error(`Failed to upload file ${filePath}:`, error);
          const errorMessage = error instanceof Error ? error.message : 'Unknown error';
          onUploadError?.(`Failed to upload ${filePath.split(/[/\\]/).pop()}: ${errorMessage}`);
        }
      }

    } catch (error) {
      console.error('Failed to open file picker:', error);
      const errorMessage = error instanceof Error ? error.message : 'Unknown error';
      onUploadError?.(`Failed to open file picker: ${errorMessage}`);
    }
  };

  return (
    <Button
      type="button"
      variant="ghost"
      size="icon"
      className={cn(
        'h-8 w-8 text-muted-foreground hover:text-foreground',
        'rounded-lg transition-colors',
        className
      )}
      onClick={handleFileSelect}
      disabled={fileUpload.isPending}
      title="Upload files"
    >
      {fileUpload.isPending ? (
        <Loader2 className="h-4 w-4 animate-spin" />
      ) : (
        <Paperclip className="h-4 w-4" />
      )}
    </Button>
  );
}

// Helper function to determine MIME type from file extension
function getMimeType(fileName: string): string {
  const extension = fileName.split('.').pop()?.toLowerCase();
  
  const mimeTypes: Record<string, string> = {
    // Images
    png: 'image/png',
    jpg: 'image/jpeg',
    jpeg: 'image/jpeg',
    gif: 'image/gif',
    webp: 'image/webp',
    bmp: 'image/bmp',
    tiff: 'image/tiff',
    
    // Videos
    mp4: 'video/mp4',
    webm: 'video/webm',
    mov: 'video/quicktime',
    avi: 'video/x-msvideo',
    mkv: 'video/x-matroska',
    wmv: 'video/x-ms-wmv',
    flv: 'video/x-flv',
    m4v: 'video/x-m4v',
    
    // Audio
    mp3: 'audio/mpeg',
    wav: 'audio/wav',
    flac: 'audio/flac',
    aac: 'audio/aac',
    m4a: 'audio/mp4',
    ogg: 'audio/ogg',
    
    // Documents
    pdf: 'application/pdf',
    doc: 'application/msword',
    docx: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    txt: 'text/plain',
    md: 'text/markdown',
    rtf: 'application/rtf',
    csv: 'text/csv',
    xls: 'application/vnd.ms-excel',
    xlsx: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    
    // Code
    js: 'text/javascript',
    ts: 'text/typescript',
    jsx: 'text/jsx',
    tsx: 'text/tsx',
    json: 'application/json',
    xml: 'application/xml',
    html: 'text/html',
    css: 'text/css',
  };

  return mimeTypes[extension || ''] || 'application/octet-stream';
}