import {
  AudioLines,
  ImageIcon,
  NotebookPen,
  VideoIcon,
} from 'lucide-react';
import { useState } from 'react';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import type { useFileReference } from '@/hooks/useFileReference';
import { useBoundStore } from '@/stores';
import type { Attachment } from '@/stores/attachmentSlice';
import { getFileTypeFromExtension } from '@/utils/fileTypes';
import { generatePreviewUrl } from '@/utils/assetServer';
import { AppIcon } from './app-icon';

interface CommandFileReferenceProps {
  fileRef: ReturnType<typeof useFileReference>;
  apps?: Attachment[];
  text: string;
  onTextUpdate?: (newText: string) => void;
  onClose?: () => void;
  sessionId: string;
}

// Media thumbnail component
const MediaThumbnail = ({
  file,
  sessionId
}: {
  file: Attachment;
  sessionId: string;
}) => {
  // Safety checks
  if (!file || !file.name || !sessionId) {
    console.error('MediaThumbnail: Missing required props', { file, sessionId });
    return <ImageIcon className="size-4 text-red-500" />;
  }

  const fileType = getFileTypeFromExtension(file.name);

  const previewUrl = generatePreviewUrl({ path: file.name, type: fileType }, sessionId);
  
  // Log errors if generatePreviewUrl failed for media files
  if (!previewUrl && (fileType === 'image' || fileType === 'video')) {
    console.error('❌ Preview URL generation failed for media file:', {
      name: file.name,
      type: fileType,
      sessionId
    });
  }

  if (fileType === 'image') {
    if (!previewUrl) {
      return <ImageIcon className="size-4 text-green-500" />;
    }

    return (
      <div className="relative flex-shrink-0">
        <img
          alt={file.name}
          className="size-10 rounded-xs object-cover"
          onError={(e) => {
            console.error('Failed to load image thumbnail:', previewUrl);
            e.currentTarget.style.display = 'none';
            const fallback = e.currentTarget.nextElementSibling as HTMLElement;
            if (fallback) fallback.style.display = 'block';
          }}
          src={previewUrl}
        />
        <ImageIcon
          className="absolute top-0 left-0 size-4 text-green-500"
          style={{ display: 'none' }}
        />
      </div>
    );
  }

  if (fileType === 'video') {
    if (!previewUrl) {
      return <VideoIcon className="size-4 text-green-500" />;
    }

    return (
      <div className="relative flex-shrink-0">
        <img
          alt={`${file.name} thumbnail`}
          className="size-10 aspect-auto rounded-xs object-cover"
          onError={(e) => {
            console.error('Failed to load video thumbnail:', previewUrl);
            e.currentTarget.style.display = 'none';
            const fallback = e.currentTarget.nextElementSibling as HTMLElement;
            if (fallback) fallback.style.display = 'block';
          }}
          src={previewUrl}
        />
        <VideoIcon
          className="absolute top-0 left-0 size-4 text-green-500"
          style={{ display: 'none' }}
        />
      </div>
    );
  }

  if (fileType === 'audio') {
    return <AudioLines className="size-4 text-green-500" />;
  }

  if (fileType === 'text') {
    return <NotebookPen className="size-4 text-green-500" />;
  }

  return <ImageIcon className="size-4 text-green-500" />;
};

export function CommandFileReference({
  fileRef,
  apps = [],
  text,
  onTextUpdate,
  onClose,
  sessionId,
}: CommandFileReferenceProps) {
  const addAttachment = useBoundStore((state) => state.addAttachment);
  const addReference = useBoundStore((state) => state.addReference);

  const handleAppSelect = (app: Attachment) => {
    const words = text.split(' ');
    const displayReference = `@${app.name}`;
    words[words.length - 1] = `${displayReference} `;
    const newText = words.join(' ');

    addAttachment(app);
    addReference(displayReference, `app:${app.name}`);
    onTextUpdate?.(newText);
  };

  const [selectedValue, setSelectedValue] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState<string>('');

  // Filter files based on search query
  const filteredFiles = searchQuery.trim()
    ? fileRef.files.filter((file) =>
        file.name.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : fileRef.files;

  // Filter apps based on search query
  const filteredApps = searchQuery.trim()
    ? apps.filter((app) =>
        app.name.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : apps;

  const handleSelect = (value: string) => {
    // Clear search query and selected value to prevent state interference
    setSearchQuery('');
    setSelectedValue('');

    if (value.startsWith('file:')) {
      const fileName = value.substring(5);
      const file = filteredFiles.find((f) => f.name === fileName);
      if (file) {
        fileRef.selectFile(file);
      }
    } else if (value.startsWith('app:')) {
      const appName = value.substring(4);
      const app = apps.find((a) => a.name === appName);
      if (app) {
        handleAppSelect(app);
      }
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape' && onClose) {
      e.preventDefault();
      onClose();
    }
  };

  return (
    <div className="absolute right-0 bottom-full left-0 z-50 mb-2 overflow-hidden rounded-xl border border-border bg-popover shadow-lg">
      <Command
        className="max-h-64"
        onKeyDown={handleKeyDown}
        onValueChange={setSelectedValue}
        value={selectedValue}
      >
        <CommandInput
          autoFocus
          onValueChange={setSearchQuery}
          placeholder="Search session files..."
          value={searchQuery}
        />

        <CommandList>
          {fileRef.isLoadingFolder ? (
            <div className="px-3 py-2 text-muted-foreground text-xs">
              Loading session files...
            </div>
          ) : filteredFiles.length || filteredApps.length ? (
            <>
              {/* Session Files Section */}
              {filteredFiles.length > 0 && (
                <CommandGroup heading="Session Files">
                  {filteredFiles.map((file) => {
                    const fileType = getFileTypeFromExtension(file.name);
                    const typeLabel = fileType
                      ? fileType.charAt(0).toUpperCase() + fileType.slice(1)
                      : 'File';

                    return (
                      <CommandItem
                        key={file.id}
                        onSelect={() => handleSelect(`file:${file.name}`)}
                        value={`file:${file.name}`}
                      >
                        <MediaThumbnail
                          file={file}
                          sessionId={sessionId}
                        />
                        <div className="flex-1">
                          <div className="font-medium text-sm">{file.name}</div>
                          {file.extension && (
                            <div className="text-muted-foreground text-xs">
                              {typeLabel} • .{file.extension}
                            </div>
                          )}
                        </div>
                      </CommandItem>
                    );
                  })}
                </CommandGroup>
              )}

              {/* Applications Section */}
              {filteredApps.length > 0 && (
                <CommandGroup heading="Applications">
                  {filteredApps.map((app) => (
                    <CommandItem
                      key={app.id}
                      onSelect={() => handleSelect(`app:${app.name}`)}
                      value={`app:${app.name}`}
                    >
                      <div className="flex-shrink-0 rounded-md bg-white p-1 shadow-sm dark:bg-gray-700">
                        <AppIcon
                          bundleId={app.bundleId || app.id.replace('app:', '')}
                          className="size-4"
                          name={app.name}
                        />
                      </div>
                      <div className="flex-1">
                        <div className="font-medium text-sm">{app.name}</div>
                        <div className="text-muted-foreground text-xs">
                          Application • Running
                        </div>
                      </div>
                    </CommandItem>
                  ))}
                </CommandGroup>
              )}
            </>
          ) : (
            <CommandEmpty>
              {searchQuery
                ? 'No files or apps match your search'
                : 'No session files found'}
            </CommandEmpty>
          )}
        </CommandList>

        {/* Bottom Toolbar - Simplified */}
        <div className="flex h-6 items-center justify-between border-gray-200/50 border-t bg-gray-50/80 px-3 py-1 text-xs dark:border-gray-700/50 dark:bg-gray-800/80">
          {/* Left side - Context */}
          <div className="flex items-center gap-2 text-gray-600 dark:text-gray-400">
            <span className="font-medium">Session Files</span>
          </div>

          {/* Right side - Keyboard shortcuts */}
          <div className="flex items-center gap-1.5">
            <div className="flex items-center gap-0.5">
              <kbd className="rounded border border-gray-300 bg-white px-1 py-0 font-mono text-gray-600 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-300">
                ↵
              </kbd>
              <span className="text-gray-500 dark:text-gray-400">select</span>
            </div>

            {onClose && (
              <div className="flex items-center gap-0.5">
                <kbd className="rounded border border-gray-300 bg-white px-1 py-0 font-mono text-gray-600 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-300">
                  esc
                </kbd>
                <span className="text-gray-500 dark:text-gray-400">close</span>
              </div>
            )}
          </div>
        </div>
      </Command>
    </div>
  );
}