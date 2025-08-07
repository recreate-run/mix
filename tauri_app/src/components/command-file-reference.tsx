import { convertFileSrc } from '@tauri-apps/api/core';
import { readDir } from '@tauri-apps/plugin-fs';
import {
  AudioLines,
  FolderIcon,
  ImageIcon,
  Monitor,
  NotebookPen,
  Play,
  VideoIcon,
} from 'lucide-react';
import { useCallback, useEffect, useRef, useState } from 'react';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';
import type { FileEntry } from '@/hooks/useFileSystem';
import {
  type Attachment,
  filterAndSortEntries,
} from '@/stores/attachmentStore';
import { getFileType } from '@/utils/fileTypes';
import { AppIcon } from './app-icon';

const RECURSIVE_SEARCH_DEPTH = 3;

interface Props {
  files: FileEntry[];
  apps?: Attachment[];
  onSelect: (file: FileEntry) => void;
  onSelectApp?: (app: Attachment) => void;
  currentFolder?: string | null;
  isLoadingFolder?: boolean;
  onGoBack?: () => void;
  onEnterFolder?: (file: FileEntry) => void;
  onClose?: () => void;
}

// Media thumbnail component
const MediaThumbnail = ({ file }: { file: FileEntry }) => {
  const fileType = getFileType(file.name);

  if (!fileType) {
    return <ImageIcon className="size-4 text-green-500" />;
  }

  const previewUrl = convertFileSrc(file.path);

  if (fileType === 'image') {
    return (
      <div className="relative flex-shrink-0">
        <img
          alt={file.name}
          className="size-8 rounded-sm object-cover"
          onError={(e) => {
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
    return (
      <div className="relative size-4 flex-shrink-0">
        <video
          className="size-4 rounded-sm object-cover"
          onError={(e) => {
            e.currentTarget.style.display = 'none';
            const fallback = e.currentTarget.nextElementSibling as HTMLElement;
            if (fallback) fallback.style.display = 'block';
          }}
          onLoadedMetadata={(e) => {
            e.currentTarget.currentTime = 1;
          }}
          preload="metadata"
          src={previewUrl}
        />
        <Play className="-bottom-0.5 -right-0.5 absolute h-2 w-2 rounded-full bg-black/50 p-0.5 text-white" />
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
  files,
  apps = [],
  onSelect,
  onSelectApp,
  currentFolder,
  isLoadingFolder,
  onGoBack,
  onEnterFolder,
  onClose,
}: Props) {
  const [selectedValue, setSelectedValue] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [allFiles, setAllFiles] = useState<FileEntry[]>([]);
  const [isLoadingAllFiles, setIsLoadingAllFiles] = useState(false);
  const commandRef = useRef<HTMLDivElement>(null);

  // Recursive fetch function - loads all files upfront
  const recursiveFetch = useCallback(
    async (basePath: string, depth = 0): Promise<FileEntry[]> => {
      if (depth >= RECURSIVE_SEARCH_DEPTH) {
        return [];
      }

      try {
        const entries = await readDir(basePath);
        const fileEntries = filterAndSortEntries(entries, basePath);
        const results: FileEntry[] = [];

        // Add all files/folders from current directory
        results.push(...fileEntries);

        // Recursively fetch subdirectories
        const directoryEntries = fileEntries.filter((file) => file.isDirectory);
        const recursivePromises = directoryEntries.map(async (dir) => {
          try {
            return await recursiveFetch(dir.path || '', depth + 1);
          } catch (error) {
            // Skip directories we can't access
            return [];
          }
        });

        const recursiveResults = await Promise.all(recursivePromises);
        recursiveResults.forEach((result) => results.push(...result));

        return results;
      } catch (error) {
        console.error('Error in recursive fetch:', error);
        return [];
      }
    },
    []
  );

  // Load all files recursively on component mount
  useEffect(() => {
    const loadAllFiles = async () => {
      const basePath =
        currentFolder ||
        (files.length > 0
          ? files[0].path?.split('/').slice(0, -1).join('/')
          : '');
      if (!basePath) {
        return;
      }

      setIsLoadingAllFiles(true);
      try {
        const allFileResults = await recursiveFetch(basePath);
        setAllFiles(allFileResults);
      } catch (error) {
        console.error('Failed to load all files:', error);
      } finally {
        setIsLoadingAllFiles(false);
      }
    };

    loadAllFiles();
  }, [currentFolder, files, recursiveFetch]);

  // Filter files based on search query - client-side filtering of preloaded files
  const filteredFiles = searchQuery.trim()
    ? allFiles.filter((file) =>
        file.name.toLowerCase().includes(searchQuery.toLowerCase())
      )
    : files;

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
      // Look in both current files and all files for the selection
      const file =
        filteredFiles.find((f) => f.name === fileName) ||
        files.find((f) => f.name === fileName);
      if (file) {
        onSelect(file);
      }
    } else if (value.startsWith('app:')) {
      const appName = value.substring(4);
      const app = apps.find((a) => a.name === appName);
      if (app && onSelectApp) {
        onSelectApp(app);
      }
    }
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'ArrowLeft' && currentFolder && onGoBack) {
      e.preventDefault();
      onGoBack();
    } else if (e.key === 'ArrowRight') {
      if (selectedValue.startsWith('file:')) {
        const fileName = selectedValue.substring(5);
        const selectedFile = filteredFiles.find((f) => f.name === fileName);
        if (selectedFile?.isDirectory && onEnterFolder) {
          e.preventDefault();
          onEnterFolder(selectedFile);
        }
      }
    } else if (e.key === 'Escape' && onClose) {
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
        ref={commandRef}
        value={selectedValue}
      >
        <CommandInput
          autoFocus
          onValueChange={setSearchQuery}
          placeholder="Search files and folders..."
          value={searchQuery}
        />

        <CommandList>
          {isLoadingFolder || isLoadingAllFiles ? (
            <div className="px-3 py-2 text-muted-foreground text-xs">
              {isLoadingAllFiles
                ? 'Loading all files...'
                : 'Loading folder contents...'}
            </div>
          ) : filteredFiles.length || filteredApps.length ? (
            <>
              {/* Files & Folders Section */}
              {filteredFiles.length > 0 && (
                <CommandGroup
                  heading={
                    currentFolder ? 'Files & Folders' : 'Media & Folders'
                  }
                >
                  {filteredFiles.map((file) => {
                    const fileType = getFileType(file.name);
                    const typeLabel = fileType
                      ? fileType.charAt(0).toUpperCase() + fileType.slice(1)
                      : 'File';

                    return (
                      <CommandItem
                        key={file.path}
                        onSelect={() => handleSelect(`file:${file.name}`)}
                        value={`file:${file.name}`}
                      >
                        {file.isDirectory ? (
                          <FolderIcon className="size-4 text-blue-500" />
                        ) : (
                          <MediaThumbnail file={file} />
                        )}
                        <div className="flex-1">
                          <div className="font-medium text-sm">{file.name}</div>
                          {file.extension && (
                            <div className="text-muted-foreground text-xs">
                              {file.isDirectory ? 'Folder' : typeLabel} • .
                              {file.extension}
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
                : currentFolder
                  ? 'No files found in folder'
                  : 'No files or apps found'}
            </CommandEmpty>
          )}
        </CommandList>

        {/* Bottom Toolbar - Raycast Style */}
        <div className="flex h-6 items-center justify-between border-gray-200/50 border-t bg-gray-50/80 px-3 py-1 text-xs dark:border-gray-700/50 dark:bg-gray-800/80">
          {/* Left side - Selection context */}
          <div className="flex items-center gap-2 text-gray-600 dark:text-gray-400">
            {currentFolder && (
              <span className="font-medium">{currentFolder}</span>
            )}
          </div>

          {/* Right side - Keyboard shortcuts */}
          <div className="flex items-center gap-1.5">
            <div className="flex items-center gap-0.5">
              <kbd className="rounded border border-gray-300 bg-white px-1 py-0 font-mono text-gray-600 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-300">
                ↵
              </kbd>
              <span className="text-gray-500 dark:text-gray-400">select</span>
            </div>

            {selectedValue.startsWith('file:') &&
              (() => {
                const fileName = selectedValue.substring(5);
                const selectedFile = filteredFiles.find(
                  (f) => f.name === fileName
                );
                return (
                  selectedFile?.isDirectory &&
                  onEnterFolder && (
                    <div className="flex items-center gap-0.5">
                      <kbd className="rounded border border-gray-300 bg-white px-1 py-0 font-mono text-gray-600 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-300">
                        →
                      </kbd>
                      <span className="text-gray-500 dark:text-gray-400">
                        open
                      </span>
                    </div>
                  )
                );
              })()}

            {currentFolder && onGoBack && (
              <div className="flex items-center gap-0.5">
                <kbd className="rounded border border-gray-300 bg-white px-1 py-0 font-mono text-gray-600 text-xs dark:border-gray-600 dark:bg-gray-700 dark:text-gray-300">
                  ←
                </kbd>
                <span className="text-gray-500 dark:text-gray-400">back</span>
              </div>
            )}

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
