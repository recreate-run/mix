// Types matching the Go backend structures
interface FileTypeInfo {
	extensions: string[];
	mime_types: Record<string, number>;
	size_limit: number;
}

export interface SupportedFileTypes {
	image: FileTypeInfo;
	video: FileTypeInfo;
	audio: FileTypeInfo;
}

type FileType = "image" | "video" | "audio" | "text";

// Static extension arrays - single source of truth
const IMAGE_EXTENSIONS = [
	"png",
	"jpg",
	"jpeg",
	"gif",
	"webp",
	"bmp",
	"tiff",
] as const;
const VIDEO_EXTENSIONS = [
	"mp4",
	"webm",
	"mov",
	"avi",
	"mkv",
	"wmv",
	"flv",
	"m4v",
] as const;
const AUDIO_EXTENSIONS = ["mp3", "wav", "flac", "aac", "m4a", "ogg"] as const;
const TEXT_EXTENSIONS = ["md", "txt"] as const;

// Type-safe extension checking helpers
type ImageExtension = (typeof IMAGE_EXTENSIONS)[number];
type VideoExtension = (typeof VIDEO_EXTENSIONS)[number];
type AudioExtension = (typeof AUDIO_EXTENSIONS)[number];
type TextExtension = (typeof TEXT_EXTENSIONS)[number];

function isImageExtension(ext: string): ext is ImageExtension {
	return IMAGE_EXTENSIONS.includes(ext as ImageExtension);
}

function isVideoExtension(ext: string): ext is VideoExtension {
	return VIDEO_EXTENSIONS.includes(ext as VideoExtension);
}

function isAudioExtension(ext: string): ext is AudioExtension {
	return AUDIO_EXTENSIONS.includes(ext as AudioExtension);
}

function isTextExtension(ext: string): ext is TextExtension {
	return TEXT_EXTENSIONS.includes(ext as TextExtension);
}

/**
 * Simple file type detection based on extension only (no backend dependency)
 * Use this when supportedTypes data isn't available yet
 */
export function getFileTypeFromExtension(fileName: string): FileType {
	const extension = fileName.split(".").pop()?.toLowerCase();
	if (!extension) return "text";

	if (isImageExtension(extension)) return "image";
	if (isVideoExtension(extension)) return "video";
	if (isAudioExtension(extension)) return "audio";
	if (isTextExtension(extension)) return "text";

	return "text"; // Default fallback
}

export function getFileType(
	fileName: string,
	supportedTypes?: SupportedFileTypes,
): FileType | null {
	const extension = `.${fileName.split(".").pop()?.toLowerCase()}`;
	if (!extension || extension === ".") return null;

	// Handle text files (frontend-only logic)
	const textExt = fileName.split(".").pop()?.toLowerCase();
	if (textExt && isTextExtension(textExt)) return "text";

	// If no supported types provided, use fallback detection
	if (!supportedTypes) {
		return getFileTypeFromExtension(fileName);
	}

	if (supportedTypes.image.extensions.includes(extension)) return "image";
	if (supportedTypes.video.extensions.includes(extension)) return "video";
	if (supportedTypes.audio.extensions.includes(extension)) return "audio";

	return null;
}
