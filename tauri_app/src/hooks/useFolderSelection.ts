import { useState, useCallback, useEffect } from 'react';
import { open } from '@tauri-apps/plugin-dialog';
import { safeTrackEvent } from '@/lib/posthog';

const FOLDER_STORAGE_KEY = 'file-reference-parent-folder';

export const useFolderSelection = () => {
  const [selectedFolder, setSelectedFolder] = useState<string | null>(null);

  // Load folder from localStorage on mount
  useEffect(() => {
    const stored = localStorage.getItem(FOLDER_STORAGE_KEY);
    if (stored) {
      setSelectedFolder(stored);
    }
  }, []);

  const selectFolder = useCallback(async (): Promise<string | null> => {
    try {
      const selected = await open({
        directory: true,
        multiple: false,
      });

      if (selected && typeof selected === 'string') {
        setSelectedFolder(selected);
        localStorage.setItem(FOLDER_STORAGE_KEY, selected);
        
        // Track folder selection event
        safeTrackEvent('folder_selected', {
          folder_path: selected,
          timestamp: new Date().toISOString()
        });
        
        return selected;
      }
      return null;
    } catch (error) {
      console.error('Failed to select folder:', error);
      return null;
    }
  }, []);

  const clearFolder = useCallback(() => {
    setSelectedFolder(null);
    localStorage.removeItem(FOLDER_STORAGE_KEY);
    
    // Track folder cleared event
    safeTrackEvent('folder_cleared', {
      timestamp: new Date().toISOString()
    });
  }, []);

  return {
    selectedFolder,
    selectFolder,
    clearFolder,
  };
};