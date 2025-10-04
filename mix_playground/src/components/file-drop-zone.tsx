import { useState } from 'react';
import { cn } from '@/lib/utils';
import { useFileUpload } from '@/hooks/useFileUpload';
import { useBoundStore } from '@/stores';
import { getFileTypeFromExtension } from '@/utils/fileTypes';

interface FileDropZoneProps {
  children: React.ReactNode;
  sessionId: string;
  onUploadSuccess?: (fileName: string) => void;
  onUploadError?: (error: string) => void;
  className?: string;
}

export function FileDropZone({
  children,
  sessionId,
  onUploadSuccess,
  onUploadError,
  className,
}: FileDropZoneProps) {
  const [isDraggingOver, setIsDraggingOver] = useState(false);
  const fileUpload = useFileUpload();
  const addAttachment = useBoundStore((state) => state.addAttachment);

  const processFile = async (file: File) => {
    try {
      // Upload file using our hook
      const result = await fileUpload.mutateAsync({
        sessionId,
        file,
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
      console.error(`Failed to upload file ${file.name}:`, error);
      const errorMessage =
        error instanceof Error ? error.message : 'Unknown error';
      onUploadError?.(`Failed to upload ${file.name}: ${errorMessage}`);
    }
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();

    // Only show drag state if files are being dragged
    if (e.dataTransfer.types.includes('Files')) {
      setIsDraggingOver(true);
    }
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();

    // Only clear drag state if we're leaving the drop zone
    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX;
    const y = e.clientY;

    if (
      x <= rect.left ||
      x >= rect.right ||
      y <= rect.top ||
      y >= rect.bottom
    ) {
      setIsDraggingOver(false);
    }
  };

  const handleDrop = async (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDraggingOver(false);

    const files = Array.from(e.dataTransfer.files);

    if (files.length === 0) {
      return;
    }

    // Process each dropped file
    for (const file of files) {
      await processFile(file);
    }
  };

  return (
    <div
      className={cn('relative', className)}
      onDragLeave={handleDragLeave}
      onDragOver={handleDragOver}
      onDrop={handleDrop}
    >
      {children}

      {/* Drag overlay */}
      {isDraggingOver && (
        <div className="pointer-events-none absolute inset-0 z-50 flex items-center justify-center rounded-lg border-2 border-dashed border-primary bg-primary/10 backdrop-blur-sm">
          <div className="flex flex-col items-center gap-2 text-primary">
            <svg
              className="h-8 w-8"
              fill="none"
              stroke="currentColor"
              strokeWidth={2}
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
            <span className="text-sm font-medium">Drop files to upload</span>
          </div>
        </div>
      )}
    </div>
  );
}
