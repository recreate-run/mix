import { useReducer, useEffect, useRef } from 'react';
import { useFileSystem, type MediaItem } from './useFileSystem';
import { useAttachmentStore, getParentPath } from '@/stores/attachmentStore';
import { ALL_MEDIA_EXTENSIONS } from '@/utils/fileTypes';
import { safeTrackEvent } from '@/lib/posthog';

type State = {
  selected: number;
  folderContents: MediaItem[];
  currentFolder: string | null;
  isLoadingFolder: boolean;
};

type Action = 
  | { type: 'SET_SELECTED'; payload: number }
  | { type: 'RESET_SELECTION' }
  | { type: 'SET_FOLDER_CONTENTS'; payload: MediaItem[] }
  | { type: 'SET_CURRENT_FOLDER'; payload: string | null }
  | { type: 'SET_LOADING'; payload: boolean }
  | { type: 'RESET_STATE' }
  | { type: 'ENTER_FOLDER'; payload: { contents: MediaItem[]; folder: string } };

const initialState: State = {
  selected: 0,
  folderContents: [],
  currentFolder: null,
  isLoadingFolder: false,
};

const reducer = (state: State, action: Action): State => {
  switch (action.type) {
    case 'SET_SELECTED':
      return { ...state, selected: action.payload };
    case 'RESET_SELECTION':
      return { ...state, selected: 0 };
    case 'SET_FOLDER_CONTENTS':
      return { ...state, folderContents: action.payload };
    case 'SET_CURRENT_FOLDER':
      return { ...state, currentFolder: action.payload };
    case 'SET_LOADING':
      return { ...state, isLoadingFolder: action.payload };
    case 'RESET_STATE':
      return initialState;
    case 'ENTER_FOLDER':
      return {
        ...state,
        folderContents: action.payload.contents,
        currentFolder: action.payload.folder,
        selected: 0,
        isLoadingFolder: false,
      };
    default:
      return state;
  }
};

export const useFileReference = (text: string, setText: (text: string) => void, customBasePath?: string) => {
  const [state, dispatch] = useReducer(reducer, initialState);
  const { currentFiles, fetchFiles, fetchDirectoryContents } = useFileSystem(customBasePath);
  const loadingTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  
  const addAttachment = useAttachmentStore(state => state.addAttachment);
  const addReference = useAttachmentStore(state => state.addReference);
  
  const startDebouncedLoading = () => {
    // Clear any existing timeout
    if (loadingTimeoutRef.current) {
      clearTimeout(loadingTimeoutRef.current);
    }
    
    // Set loading after 150ms delay
    loadingTimeoutRef.current = setTimeout(() => {
      dispatch({ type: 'SET_LOADING', payload: true });
      loadingTimeoutRef.current = null;
    }, 150);
  };
  
  const clearLoadingTimeout = () => {
    if (loadingTimeoutRef.current) {
      clearTimeout(loadingTimeoutRef.current);
      loadingTimeoutRef.current = null;
    }
  };
  
  const baseFiles = state.currentFolder ? state.folderContents : currentFiles;
  const files = baseFiles.filter(f => 
    f.isDirectory || 
    (f.extension && ALL_MEDIA_EXTENSIONS.includes(f.extension as any))
  );
  
  const words = text.split(' ');
  const lastWord = words[words.length - 1];
  const show = lastWord.startsWith('@') && !lastWord.includes('/');
  
  useEffect(() => {
    if (show) {
      fetchFiles();
    }
  }, [show, fetchFiles]);
  
  useEffect(() => {
    dispatch({ type: 'RESET_SELECTION' });
  }, [files.length]);

  useEffect(() => {
    if (show && !state.currentFolder) {
      dispatch({ type: 'RESET_STATE' });
    }
  }, [show, state.currentFolder]);

  // Cleanup timeout on unmount or when popup is hidden
  useEffect(() => {
    return () => {
      clearLoadingTimeout();
    };
  }, []);

  useEffect(() => {
    if (!show) {
      clearLoadingTimeout();
      dispatch({ type: 'SET_LOADING', payload: false });
    }
  }, [show]);
  

  const handleSelection = async (selectedFile?: MediaItem) => {
    const file = selectedFile || files[state.selected];
    if (!file) return;
    
    const words = text.split(' ');
    // Only add "../" if file path contains subdirectories
    const hasSubdirectory = file.path.includes('/');
    const prefix = hasSubdirectory ? '../' : '';
    const displayReference = `@${prefix}${file.name}`;
    words[words.length - 1] = `${displayReference} `;
    const newText = words.join(' ');
    
    // Add file or folder to attachment store based on type
    if (file.isDirectory) {
      const { createFolderAttachment } = await import('@/stores/attachmentStore');
      const folderAttachment = await createFolderAttachment(file.path);
      addAttachment(folderAttachment);
      
      // Track folder attachment event
      safeTrackEvent('file_attachment_added', {
        file_type: 'folder',
        file_path: file.path,
        file_name: file.name,
        timestamp: new Date().toISOString()
      });
    } else {
      const { createFileAttachment } = await import('@/stores/attachmentStore');
      const fileAttachment = createFileAttachment(file.path);
      if (fileAttachment) {
        addAttachment(fileAttachment);
        
        // Track file attachment event
        safeTrackEvent('file_attachment_added', {
          file_type: 'file',
          file_path: file.path,
          file_name: file.name,
          file_extension: file.extension,
          timestamp: new Date().toISOString()
        });
      }
    }
    addReference(displayReference, file.path);
    setText(newText);
    
    // Track file referenced event
    safeTrackEvent('file_referenced', {
      file_path: file.path,
      file_name: file.name,
      is_directory: file.isDirectory,
      timestamp: new Date().toISOString()
    });
  };

  const handleEscape = () => {
    dispatch({ type: 'RESET_STATE' });
    const words = text.split(' ');
    words[words.length - 1] = '';
    const newText = words.join(' ').trim();
    setText(newText);
  };

  const closeDropdown = () => {
    dispatch({ type: 'RESET_STATE' });
  };

  const enterFolder = async (folder: MediaItem) => {
    startDebouncedLoading();
    try {
      const contents = await fetchDirectoryContents(folder.path);
      clearLoadingTimeout(); // Clear timeout since operation completed
      dispatch({ type: 'ENTER_FOLDER', payload: { contents, folder: folder.path } });
      
      // Track folder navigation event
      safeTrackEvent('file_navigation', {
        action: 'enter_folder',
        folder_path: folder.path,
        folder_name: folder.name,
        items_count: contents.length,
        timestamp: new Date().toISOString()
      });
    } catch (error) {
      console.error('Failed to load folder contents:', error);
      clearLoadingTimeout(); // Clear timeout on error
      dispatch({ type: 'SET_LOADING', payload: false });
    }
  };

  const goBack = async () => {
    if (!state.currentFolder) return;
    
    const parentPath = getParentPath(state.currentFolder);
    startDebouncedLoading();
    
    try {
      if (parentPath) {
        const contents = await fetchDirectoryContents(parentPath);
        clearLoadingTimeout(); // Clear timeout since operation completed
        dispatch({ type: 'ENTER_FOLDER', payload: { contents, folder: parentPath } });
        
        // Track folder navigation back event
        safeTrackEvent('file_navigation', {
          action: 'navigate_back',
          from_folder: state.currentFolder,
          to_folder: parentPath,
          items_count: contents.length,
          timestamp: new Date().toISOString()
        });
      } else {
        clearLoadingTimeout(); // Clear timeout for immediate operation
        dispatch({ type: 'SET_CURRENT_FOLDER', payload: null });
        dispatch({ type: 'SET_FOLDER_CONTENTS', payload: [] });
        dispatch({ type: 'RESET_SELECTION' });
        dispatch({ type: 'SET_LOADING', payload: false });
        
        // Track navigation to root event
        safeTrackEvent('file_navigation', {
          action: 'navigate_to_root',
          from_folder: state.currentFolder,
          timestamp: new Date().toISOString()
        });
      }
    } catch (error) {
      console.error('Failed to load parent folder contents:', error);
      clearLoadingTimeout(); // Clear timeout on error
      dispatch({ type: 'SET_LOADING', payload: false });
    }
  };
  
  
  const enterSelectedFolder = (file?: MediaItem) => {
    const selectedFile = file || files[state.selected];
    if (selectedFile?.isDirectory) {
      // Update selected state before entering
      const fileIndex = files.indexOf(selectedFile);
      if (fileIndex !== -1) {
        dispatch({ type: 'SET_SELECTED', payload: fileIndex });
      }
      enterFolder(selectedFile);
    }
  };


  return { 
    show, 
    files, 
    selected: state.selected, 
    selectFile: handleSelection, 
    currentFolder: state.currentFolder,
    isLoadingFolder: state.isLoadingFolder,
    goBack,
    enterSelectedFolder,
    close: handleEscape,
    closeDropdown
  };
};