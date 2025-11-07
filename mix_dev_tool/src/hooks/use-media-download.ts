import { useCallback, useState } from "react";
import { toast } from "sonner";
import type { MediaOutput } from "@/types/media";
import { isYouTubeUrl } from "@/utils/videoUrlDetection";

// Helper to get file extension from media type and path
const getFileExtension = (media: MediaOutput): string => {
	// Try to extract extension from path first
	const pathMatch = media.path?.match(/\.([^./?#]+)(?:[?#]|$)/);
	if (pathMatch) {
		return `.${pathMatch[1]}`;
	}

	// Fallback to media type
	const extensionMap: Record<string, string> = {
		image: ".jpg",
		video: ".mp4",
		audio: ".mp3",
		pdf: ".pdf",
		csv: ".csv",
	};

	return extensionMap[media.type] || "";
};

export const useMediaDownload = (
	media: MediaOutput,
	sessionId: string,
	getMediaSrc: (path: string, sessionId: string) => string,
) => {
	const [isDownloading, setIsDownloading] = useState(false);

	const downloadMedia = useCallback(async () => {
		// For YouTube videos, open in new tab instead of downloading
		if (media.type === "video" && media.path && isYouTubeUrl(media.path)) {
			window.open(media.path, "_blank");
			return;
		}

		if (!media.path) {
			toast.error("Download failed", {
				description: "No media path available",
			});
			return;
		}

		setIsDownloading(true);

		try {
			// For all media types, fetch as blob and trigger download
			const mediaUrl = getMediaSrc(media.path, sessionId);
			const response = await fetch(mediaUrl);

			if (!response.ok) {
				throw new Error(`Failed to fetch media: ${response.statusText}`);
			}

			const blob = await response.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement("a");
			a.href = url;

			// Create filename with proper extension
			const extension = getFileExtension(media);
			const filename = media.title
				? media.title.includes(".")
					? media.title
					: `${media.title}${extension}`
				: `media${extension}`;

			a.download = filename;
			document.body.appendChild(a);
			a.click();
			document.body.removeChild(a);
			URL.revokeObjectURL(url);

			toast.success("Download complete", {
				description: filename,
			});
		} catch (error) {
			console.error("Download failed:", error);
			toast.error("Download failed", {
				description: "Opening in new tab instead",
			});
			// Fallback: try opening in new tab
			if (media.path) {
				const mediaUrl = getMediaSrc(media.path, sessionId);
				window.open(mediaUrl, "_blank");
			}
		} finally {
			setIsDownloading(false);
		}
	}, [media, sessionId, getMediaSrc]);

	return {
		isDownloading,
		downloadMedia,
	};
};
