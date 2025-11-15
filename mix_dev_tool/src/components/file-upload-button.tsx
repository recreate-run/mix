import { Loader2, Paperclip } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { useFileUpload } from "@/hooks/useFileUpload";
import { cn } from "@/lib/utils";
import { useBoundStore } from "@/stores";
import { getFileTypeFromExtension } from "@/utils/fileTypes";

interface FileUploadButtonProps {
	sessionId: string;
	className?: string;
	onUploadSuccess?: (fileName: string) => void;
	onUploadError?: (error: string) => void;
	enableDropZone?: boolean;
	dropZoneClassName?: string;
}

export function FileUploadButton({
	sessionId,
	className,
	onUploadSuccess,
	onUploadError,
	enableDropZone = false,
	dropZoneClassName,
}: FileUploadButtonProps) {
	const fileUpload = useFileUpload();
	const addAttachment = useBoundStore((state) => state.addAttachment);
	const [isDraggingOver, setIsDraggingOver] = useState(false);

	// Shared file processing logic
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
				extension: result.name.split(".").pop(),
				isDirectory: false,
			});

			// Call success callback
			onUploadSuccess?.(result.name);
		} catch (error) {
			console.error(`Failed to upload file ${file.name}:`, error);
			const errorMessage =
				error instanceof Error ? error.message : "Unknown error";
			onUploadError?.(`Failed to upload ${file.name}: ${errorMessage}`);
		}
	};

	const handleFileSelect = async () => {
		// Use HTML file input
		const input = document.createElement("input");
		input.type = "file";
		input.multiple = true;
		// Set accept attribute with common file types
		input.accept = [
			// Images
			"image/*",
			// Videos
			"video/*",
			// Audio
			"audio/*",
			// Documents
			".pdf",
			".doc",
			".docx",
			".txt",
			".md",
			".rtf",
			".csv",
			".xls",
			".xlsx",
			// Code files
			".js",
			".ts",
			".jsx",
			".tsx",
			".json",
			".xml",
			".html",
			".css",
		].join(",");

		input.onchange = async () => {
			const files = Array.from(input.files || []);
			for (const file of files) {
				await processFile(file);
			}
		};

		input.click();
	};

	// Drag and drop handlers
	const handleDragOver = (e: React.DragEvent) => {
		e.preventDefault();
		e.stopPropagation();
		setIsDraggingOver(true);
	};

	const handleDragLeave = (e: React.DragEvent) => {
		e.preventDefault();
		e.stopPropagation();
		setIsDraggingOver(false);
	};

	const handleDrop = async (e: React.DragEvent) => {
		e.preventDefault();
		e.stopPropagation();
		setIsDraggingOver(false);

		// Use DataTransferItemList for better folder support
		const items = Array.from(e.dataTransfer.items);

		if (items.length === 0) {
			return;
		}

		// Process each dropped item (files and folders)
		for (const item of items) {
			if (item.kind === "file") {
				const entry = item.webkitGetAsEntry();
				if (entry) {
					await processEntry(entry);
				}
			}
		}
	};

	// Recursively process file system entries (files and folders)
	const processEntry = async (entry: FileSystemEntry): Promise<void> => {
		if (entry.isFile) {
			const fileEntry = entry as FileSystemFileEntry;
			fileEntry.file(async (file) => {
				await processFile(file);
			});
		} else if (entry.isDirectory) {
			const dirEntry = entry as FileSystemDirectoryEntry;
			const reader = dirEntry.createReader();

			// Read all entries in the directory
			reader.readEntries(async (entries) => {
				for (const childEntry of entries) {
					await processEntry(childEntry);
				}
			});
		}
	};

	const buttonElement = (
		<Button
			className={cn(
				"h-8 w-8 text-muted-foreground hover:text-foreground",
				"rounded-lg transition-colors",
				className,
			)}
			disabled={fileUpload.isPending}
			onClick={handleFileSelect}
			size="icon"
			title="Upload files (click or drag & drop)"
			type="button"
			variant="ghost"
		>
			{fileUpload.isPending ? (
				<Loader2 className="h-4 w-4 animate-spin" />
			) : (
				<Paperclip className="h-4 w-4" />
			)}
		</Button>
	);

	// If drop zone is enabled, wrap the button with drag and drop handlers
	if (enableDropZone) {
		return (
			// biome-ignore lint/a11y/useSemanticElements: Section needs role for interactive drag/drop handlers
			<section
				className={cn(
					"relative transition-all",
					isDraggingOver && "opacity-80",
					dropZoneClassName,
				)}
				onDragLeave={handleDragLeave}
				onDragOver={handleDragOver}
				onDrop={handleDrop}
				role="region"
			>
				{buttonElement}
				{isDraggingOver && (
					<div className="pointer-events-none absolute inset-0 flex items-center justify-center rounded-lg border-2 border-dashed border-primary bg-primary/10">
						<span className="text-xs text-primary">Drop files here</span>
					</div>
				)}
			</section>
		);
	}

	return buttonElement;
}
