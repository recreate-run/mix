import {
  ArrowLeftIcon,
  ArrowRightIcon,
  FolderIcon,
  ImageIcon,
  KeyIcon,
} from 'lucide-react';
import type { FileEntry } from '@/hooks/useFileSystem';
import { isImageFile } from '@/lib/fileUtils';

interface Props {
  files: FileEntry[];
  selected: number;
  onSelect: (file: FileEntry) => void;
  currentFolder?: string | null;
  isLoadingFolder?: boolean;
  onGoBack?: () => void;
  onEnterFolder?: () => void;
  onClose?: () => void;
}

interface FileItemProps {
  file: FileEntry;
  isSelected: boolean;
  onSelect: (file: FileEntry) => void;
}

const FileItem = ({ file, isSelected, onSelect }: FileItemProps) => {
  const Icon = file.isDirectory ? FolderIcon : ImageIcon;
  const iconColor = file.isDirectory ? 'text-blue-500' : 'text-green-500';
  const isImage = file.extension && isImageFile(file.name);

  return (
    <div
      className={`flex cursor-pointer items-center gap-3 px-3 py-2 transition-colors ${
        isSelected ? 'rounded-md bg-muted/80' : 'hover:bg-muted/30'
      }`}
      onClick={() => onSelect(file)}
    >
      <Icon className={`size-4 ${iconColor}`} />
      <div className="flex-1">
        <div className="font-medium text-sm">{file.name}</div>
        {file.extension && !file.isDirectory && (
          <div className="text-muted-foreground text-xs">
            {isImage ? 'Image' : 'Video'} • .{file.extension}
          </div>
        )}
      </div>
    </div>
  );
};

const KeyShortcut = ({
  children,
  onClick,
  title,
}: {
  children: React.ReactNode;
  onClick?: () => void;
  title?: string;
}) => (
  <button
    className="flex items-center gap-1 rounded-md bg-muted/40 px-3 font-medium font-mono text-sm transition-colors hover:bg-muted/70 hover:text-foreground"
    onClick={onClick}
    title={title}
  >
    {children}
  </button>
);

const Header = ({
  currentFolder,
  filesCount,
  onGoBack,
  canNavigateForward,
  onEnterFolder,
  onClose,
}: {
  currentFolder?: string | null;
  filesCount: number;
  onGoBack?: () => void;
  canNavigateForward?: boolean;
  onEnterFolder?: () => void;
  onClose?: () => void;
}) => (
  <div className="mb-2 flex items-center justify-between border-b px-3 py-1 text-muted-foreground text-xs">
    <span className="font-medium">
      {currentFolder
        ? `${currentFolder} (${filesCount})`
        : `Folders & Media (${filesCount})`}
    </span>
    <div className="flex items-center gap-2">
      {onClose && (
        <KeyShortcut onClick={onClose} title="Close">
          ⌫
        </KeyShortcut>
      )}
      {currentFolder && onGoBack && (
        <KeyShortcut onClick={onGoBack} title="Go back">
          ←
        </KeyShortcut>
      )}
      {canNavigateForward && onEnterFolder && (
        <KeyShortcut onClick={onEnterFolder} title="Enter folder">
          →
        </KeyShortcut>
      )}
    </div>
  </div>
);

export function FileReferencePopup({
  files,
  selected,
  onSelect,
  currentFolder,
  isLoadingFolder,
  onGoBack,
  onEnterFolder,
  onClose,
}: Props) {
  const selectedFile = files[selected];
  const canNavigateForward = selectedFile?.isDirectory;

  const handleEnterFolder = () => {
    if (selectedFile?.isDirectory && onEnterFolder) {
      onEnterFolder();
    }
  };

  return (
    <div className="absolute right-0 bottom-full left-0 z-50 mb-2 max-h-64 overflow-hidden overflow-y-auto rounded-xl border border-border bg-popover p-2 shadow-lg">
      <Header
        canNavigateForward={canNavigateForward}
        currentFolder={currentFolder}
        filesCount={files.length}
        onClose={onClose}
        onEnterFolder={handleEnterFolder}
        onGoBack={onGoBack}
      />

      {isLoadingFolder ? (
        <div className="px-3 py-2 text-muted-foreground text-xs">
          Loading folder contents...
        </div>
      ) : files.length ? (
        files.map((file, index) => (
          <FileItem
            file={file}
            isSelected={index === selected}
            key={file.path}
            onSelect={onSelect}
          />
        ))
      ) : (
        <div className="px-3 py-2 text-muted-foreground text-xs">
          {currentFolder
            ? 'No files found in folder'
            : 'No folders or media files found'}
        </div>
      )}
    </div>
  );
}
