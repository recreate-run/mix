import { Download } from "lucide-react";
import { Button } from "@/components/ui/button";
import { useMediaDownload } from "@/hooks/use-media-download";
import type { MediaOutput } from "@/types/media";
import { isYouTubeUrl } from "@/utils/videoUrlDetection";

export const MediaDownloadButton = ({
	media,
	sessionId,
	getMediaSrc,
}: {
	media: MediaOutput;
	sessionId: string;
	getMediaSrc: (path: string, sessionId: string) => string;
}) => {
	const { isDownloading, downloadMedia } = useMediaDownload(
		media,
		sessionId,
		getMediaSrc,
	);

	return (
		<Button
			className="text-muted-foreground hover:text-foreground"
			disabled={isDownloading}
			onClick={downloadMedia}
			size="sm"
			title={
				media.type === "video" && isYouTubeUrl(media.path)
					? "Open in YouTube"
					: "Download media"
			}
			variant="ghost"
		>
			<Download className="size-4" />
		</Button>
	);
};
