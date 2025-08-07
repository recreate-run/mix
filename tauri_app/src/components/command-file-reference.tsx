import { useState, useRef, useCallback, useEffect } from 'react';
import { FolderIcon, ImageIcon, VideoIcon, AudioLines, Play, Monitor, NotebookPen } from 'lucide-react';
import { type FileEntry } from '@/hooks/useFileSystem';
import { convertFileSrc } from '@tauri-apps/api/core';
import { readDir } from '@tauri-apps/plugin-fs';
import { type Attachment } from '@/stores/attachmentStore';
import { getFileType } from '@/utils/fileTypes';
import { filterAndSortEntries } from '@/stores/attachmentStore';
import { AppIcon } from './app-icon';
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command';

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
          src={previewUrl}
          alt={file.name}
          className="size-8 object-cover rounded-sm"
          onError={(e) => {
            e.currentTarget.style.display = 'none';
            const fallback = e.currentTarget.nextElementSibling as HTMLElement;
            if (fallback) fallback.style.display = 'block';
          }}
        />
        <ImageIcon 
          className="size-4 text-green-500 absolute top-0 left-0" 
          style={{ display: 'none' }}
        />
      </div>
    );
  }
  
  if (fileType === 'video') {
    return (
      <div className="relative size-4 flex-shrink-0">
        <video 
          src={previewUrl}
          className="size-4 object-cover rounded-sm"
          preload="metadata"
          onLoadedMetadata={(e) => {
            e.currentTarget.currentTime = 1;
          }}
          onError={(e) => {
            e.currentTarget.style.display = 'none';
            const fallback = e.currentTarget.nextElementSibling as HTMLElement;
            if (fallback) fallback.style.display = 'block';
          }}
        />
        <Play className="absolute -bottom-0.5 -right-0.5 w-2 h-2 text-white bg-black/50 rounded-full p-0.5" />
        <VideoIcon 
          className="size-4 text-green-500 absolute top-0 left-0" 
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
  onClose 
}: Props) {
  const [selectedValue, setSelectedValue] = useState<string>('');
  const [searchQuery, setSearchQuery] = useState<string>('');
  const [allFiles, setAllFiles] = useState<FileEntry[]>([]);
  const [isLoadingAllFiles, setIsLoadingAllFiles] = useState(false);
  const commandRef = useRef<HTMLDivElement>(null);
  
  // Recursive fetch function - loads all files upfront
  const recursiveFetch = useCallback(async (
    basePath: string,
    depth: number = 0
  ): Promise<FileEntry[]> => {
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
      const directoryEntries = fileEntries.filter(file => file.isDirectory);
      const recursivePromises = directoryEntries.map(async (dir) => {
        try {
          return await recursiveFetch(dir.path || '', depth + 1);
        } catch (error) {
          // Skip directories we can't access
          return [];
        }
      });

      const recursiveResults = await Promise.all(recursivePromises);
      recursiveResults.forEach(result => results.push(...result));

      return results;
    } catch (error) {
      console.error('Error in recursive fetch:', error);
      return [];
    }
  }, []);

  // Load all files recursively on component mount
  useEffect(() => {
    const loadAllFiles = async () => {
      const basePath = currentFolder || (files.length > 0 ? files[0].path?.split('/').slice(0, -1).join('/') : '');
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
    ? allFiles.filter(file => file.name.toLowerCase().includes(searchQuery.toLowerCase()))
    : files;

  // Filter apps based on search query
  const filteredApps = searchQuery.trim()
    ? apps.filter(app =>
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
      const file = filteredFiles.find(f => f.name === fileName) || files.find(f => f.name === fileName);
      if (file) {
        onSelect(file);
      }
    } else if (value.startsWith('app:')) {
      const appName = value.substring(4);
      const app = apps.find(a => a.name === appName);
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
        const selectedFile = filteredFiles.find(f => f.name === fileName);
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
    <div className="absolute bottom-full left-0 right-0 mb-2 bg-popover border border-border rounded-xl shadow-lg z-50 overflow-hidden">
      <Command 
        ref={commandRef}
        onKeyDown={handleKeyDown} 
        className="max-h-64"
        value={selectedValue}
        onValueChange={setSelectedValue}
      >
        
        <CommandInput 
          placeholder="Search files and folders..." 
          value={searchQuery}
          onValueChange={setSearchQuery}
          autoFocus
        />
        
        <CommandList>
          {isLoadingFolder || isLoadingAllFiles ? (
            <div className="text-xs text-muted-foreground px-3 py-2">
              {isLoadingAllFiles ? 'Loading all files...' : 'Loading folder contents...'}
            </div>
          ) : !filteredFiles.length && !filteredApps.length ? (
            <CommandEmpty>
              {searchQuery ? 'No files or apps match your search' : currentFolder ? 'No files found in folder' : 'No files or apps found'}
            </CommandEmpty>
          ) : (
            <>
              {/* Files & Folders Section */}
              {filteredFiles.length > 0 && (
                <CommandGroup heading={currentFolder ? "Files & Folders" : "Media & Folders"}>
                  {filteredFiles.map((file) => {
                    const fileType = getFileType(file.name);
                    const typeLabel = fileType ? fileType.charAt(0).toUpperCase() + fileType.slice(1) : 'File';
                    
                    return (
                      <CommandItem
                        key={file.path}
                        value={`file:${file.name}`}
                        onSelect={() => handleSelect(`file:${file.name}`)}
                      >
                        {file.isDirectory ? (
                          <FolderIcon className="size-4 text-blue-500" />
                        ) : (
                          <MediaThumbnail file={file} />
                        )}
                        <div className="flex-1">
                          <div className="font-medium text-sm">{file.name}</div>
                          {file.extension && (
                            <div className="text-xs text-muted-foreground">
                              {file.isDirectory ? 'Folder' : typeLabel} • .{file.extension}
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
                      value={`app:${app.name}`}
                      onSelect={() => handleSelect(`app:${app.name}`)}
                    >
                      <div className="flex-shrink-0 p-1 rounded-md bg-white dark:bg-gray-700 shadow-sm">
                        <AppIcon 
                          bundleId={app.bundleId || app.id.replace('app:', '')} 
                          name={app.name}
                          className="size-4"
                        />
                      </div>
                      <div className="flex-1">
                        <div className="font-medium text-sm">{app.name}</div>
                        <div className="text-xs text-muted-foreground">
                          Application • Running
                        </div>
                      </div>
                    </CommandItem>
                  ))}
                </CommandGroup>
              )}
            </>
          )}
        </CommandList>
        
        {/* Bottom Toolbar - Raycast Style */}
        <div className="h-6 px-3 py-1 bg-gray-50/80 dark:bg-gray-800/80 border-t border-gray-200/50 dark:border-gray-700/50 flex items-center justify-between text-xs">
          {/* Left side - Selection context */}
          <div className="flex items-center gap-2 text-gray-600 dark:text-gray-400">
            {currentFolder && (
              <span className="font-medium">
                {currentFolder}
              </span>
            )}
          </div>
          
          {/* Right side - Keyboard shortcuts */}
          <div className="flex items-center gap-1.5">
            <div className="flex items-center gap-0.5">
              <kbd className="px-1 py-0 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded text-gray-600 dark:text-gray-300 font-mono text-xs">
                ↵
              </kbd>
              <span className="text-gray-500 dark:text-gray-400">select</span>
            </div>
            
            {selectedValue.startsWith('file:') && (() => {
              const fileName = selectedValue.substring(5);
              const selectedFile = filteredFiles.find(f => f.name === fileName);
              return selectedFile?.isDirectory && onEnterFolder && (
                <div className="flex items-center gap-0.5">
                  <kbd className="px-1 py-0 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded text-gray-600 dark:text-gray-300 font-mono text-xs">
                    →
                  </kbd>
                  <span className="text-gray-500 dark:text-gray-400">open</span>
                </div>
              );
            })()}
            
            {currentFolder && onGoBack && (
              <div className="flex items-center gap-0.5">
                <kbd className="px-1 py-0 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded text-gray-600 dark:text-gray-300 font-mono text-xs">
                  ←
                </kbd>
                <span className="text-gray-500 dark:text-gray-400">back</span>
              </div>
            )}
            
            {onClose && (
              <div className="flex items-center gap-0.5">
                <kbd className="px-1 py-0 bg-white dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded text-gray-600 dark:text-gray-300 font-mono text-xs">
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