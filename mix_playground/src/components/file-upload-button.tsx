import { open } from "@tauri-apps/plugin-dialog";
import { readFile } from "@tauri-apps/plugin-fs";
import { Loader2, Paperclip } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useFileUpload } from "@/hooks/useFileUpload";
import { cn } from "@/lib/utils";
import { useBoundStore } from "@/stores";
import {
	AUDIO_EXTENSIONS,
	getFileTypeFromExtension,
	IMAGE_EXTENSIONS,
	VIDEO_EXTENSIONS,
} from "@/utils/fileTypes";
import { useState } from "react";

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
		try {
			// Open native file picker dialog
			const selected = await open({
				multiple: true,
				filters: [
					{
						name: "All Files",
						extensions: ["*"],
					},
					{
						name: "Images",
						extensions: [...IMAGE_EXTENSIONS],
					},
					{
						name: "Videos",
						extensions: [...VIDEO_EXTENSIONS],
					},
					{
						name: "Audio",
						extensions: [...AUDIO_EXTENSIONS],
					},
					{
						name: "Documents",
						extensions: [
							"pdf",
							"doc",
							"docx",
							"txt",
							"md",
							"rtf",
							"csv",
							"xls",
							"xlsx",
						],
					},
				],
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
					const fileName = filePath.split(/[/\\]/).pop() || "unnamed-file";

					// Create File object for upload
					const file = new File([fileData], fileName, {
						type: getMimeType(fileName),
					});

					// Process file
					await processFile(file);
				} catch (error) {
					console.error(`Failed to upload file ${filePath}:`, error);
					const errorMessage =
						error instanceof Error ? error.message : "Unknown error";
					onUploadError?.(
						`Failed to upload ${filePath.split(/[/\\]/).pop()}: ${errorMessage}`,
					);
				}
			}
		} catch (error) {
			console.error("Failed to open file picker:", error);
			const errorMessage =
				error instanceof Error ? error.message : "Unknown error";
			onUploadError?.(`Failed to open file picker: ${errorMessage}`);
		}
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

		const files = Array.from(e.dataTransfer.files);

		if (files.length === 0) {
			return;
		}

		// Process each dropped file
		for (const file of files) {
			await processFile(file);
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
			<div
				className={cn(
					"relative transition-all",
					isDraggingOver && "opacity-80",
					dropZoneClassName,
				)}
				onDragLeave={handleDragLeave}
				onDragOver={handleDragOver}
				onDrop={handleDrop}
			>
				{buttonElement}
				{isDraggingOver && (
					<div className="pointer-events-none absolute inset-0 flex items-center justify-center rounded-lg border-2 border-dashed border-primary bg-primary/10">
						<span className="text-xs text-primary">Drop files here</span>
					</div>
				)}
			</div>
		);
	}

	return buttonElement;
}

// Helper function to determine MIME type from file extension
function getMimeType(fileName: string): string {
	const extension = fileName.split(".").pop()?.toLowerCase();

	const mimeTypes: Record<string, string> = {
		// Images
		png: "image/png",
		jpg: "image/jpeg",
		jpeg: "image/jpeg",
		gif: "image/gif",
		webp: "image/webp",
		bmp: "image/bmp",
		tiff: "image/tiff",

		// Videos
		mp4: "video/mp4",
		webm: "video/webm",
		mov: "video/quicktime",
		avi: "video/x-msvideo",
		mkv: "video/x-matroska",
		wmv: "video/x-ms-wmv",
		flv: "video/x-flv",
		m4v: "video/x-m4v",

		// Audio
		mp3: "audio/mpeg",
		wav: "audio/wav",
		flac: "audio/flac",
		aac: "audio/aac",
		m4a: "audio/mp4",
		ogg: "audio/ogg",

		// Documents
		pdf: "application/pdf",
		doc: "application/msword",
		docx: "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		txt: "text/plain",
		md: "text/markdown",
		rtf: "application/rtf",
		csv: "text/csv",
		xls: "application/vnd.ms-excel",
		xlsx: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",

		// Code
		js: "text/javascript",
		ts: "text/typescript",
		jsx: "text/jsx",
		tsx: "text/tsx",
		json: "application/json",
		xml: "application/xml",
		html: "text/html",
		css: "text/css",
	};

	return mimeTypes[extension || ""] || "application/octet-stream";
}
