import { useState } from "react";
import type { MediaOutput } from "@/types/media";
import { isYouTubeUrl } from "@/utils/videoUrlDetection";
import { AIResponse } from "@/components/ui/kibo-ui/ai/response";
import { CsvViewer } from "./CsvViewer";
import { LazyVideoPlayer } from "./LazyVideoPlayer";
import { MediaDownloadButton } from "./media-download-button";
import { PlaylistSidebar } from "./playlist-sidebar";

// Main Media Player Component
const MainMediaPlayer = ({
	media,
	sessionId,
	getMediaSrc,
}: {
	media: MediaOutput;
	sessionId: string;
	getMediaSrc: (path: string, sessionId: string) => string;
}) => {
	return (
		<div className="mb-2 space-y-2">
			<div className="flex items-start justify-between gap-2">
				<div className="flex-1">
					<h3 className="font-semibold">{media.title}</h3>
				</div>
				<MediaDownloadButton
					getMediaSrc={getMediaSrc}
					media={media}
					sessionId={sessionId}
				/>
			</div>

			{media.type === "image" && (
				<img
					alt={media.title}
					className="aspect-auto max-h-120 object-contain"
					onError={(e) => {
						e.currentTarget.style.display = "none";
						const fallback = e.currentTarget.nextElementSibling as HTMLElement;
						if (fallback) fallback.style.display = "block";
					}}
					src={getMediaSrc(media.path!, sessionId)}
				/>
			)}

			{media.type === "video" &&
				(isYouTubeUrl(media.path!) ? (
					<div className="overflow-hidden rounded-md">
						<iframe
							allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture; web-share"
							allowFullScreen
							className="aspect-video w-full bg-black"
							frameBorder="0"
							onError={(e) => {
								e.currentTarget.style.display = "none";
								const fallback = e.currentTarget
									.nextElementSibling as HTMLElement;
								if (fallback) fallback.style.display = "block";
							}}
							referrerPolicy="strict-origin-when-cross-origin"
							src={(() => {
								const baseUrl = getMediaSrc(media.path!, sessionId);
								if (
									media.startTime !== undefined ||
									media.duration !== undefined
								) {
									try {
										const url = new URL(baseUrl);
										if (media.startTime !== undefined) {
											url.searchParams.set("start", media.startTime.toString());
										}
										if (
											media.duration !== undefined &&
											media.startTime !== undefined
										) {
											url.searchParams.set(
												"end",
												(media.startTime + media.duration).toString(),
											);
										}
										return url.toString();
									} catch {
										return baseUrl;
									}
								}
								return baseUrl;
							})()}
							title={media.title}
						/>
						<div
							className="flex h-48 items-center justify-center bg-stone-700 text-stone-400"
							style={{ display: "none" }}
						>
							Failed to load YouTube video: {media.path}
						</div>
					</div>
				) : (
					<LazyVideoPlayer media={media} sessionId={sessionId} />
				))}

			{media.type === "audio" && (
				<div className="rounded-md bg-stone-700/30 p-4">
					<audio
						className="w-full"
						controls
						onError={(e) => {
							e.currentTarget.style.display = "none";
							const fallback = e.currentTarget
								.nextElementSibling as HTMLElement;
							if (fallback) fallback.style.display = "block";
						}}
						preload="metadata"
						src={getMediaSrc(media.path!, sessionId)}
					>
						Your browser does not support the audio tag.
					</audio>
					<div
						className="mt-2 text-center text-stone-400"
						style={{ display: "none" }}
					>
						Failed to load audio: {media.path}
					</div>
				</div>
			)}

			{media.type === "markdown" && (
				<div className="rounded-md border border-border bg-muted/30 p-6">
					<AIResponse>{media.data}</AIResponse>
				</div>
			)}

			{media.type === "pdf" && (
				<div className="overflow-hidden rounded-md">
					<iframe
						className="aspect-[4/5] w-full bg-white"
						frameBorder="0"
						onError={(e) => {
							e.currentTarget.style.display = "none";
							const fallback = e.currentTarget
								.nextElementSibling as HTMLElement;
							if (fallback) fallback.style.display = "block";
						}}
						src={getMediaSrc(media.path!, sessionId)}
						title={media.title}
					/>
					<div
						className="flex h-48 items-center justify-center bg-stone-700 text-stone-400"
						style={{ display: "none" }}
					>
						Failed to load PDF document: {media.path}
					</div>
				</div>
			)}

			{media.type === "csv" && (
				<CsvViewer
					title={media.title}
					url={getMediaSrc(media.path!, sessionId)}
				/>
			)}
		</div>
	);
};

// Media Showcase Component
export const MediaShowcase = ({
	mediaOutputs,
	sessionId,
	getMediaSrc,
}: {
	mediaOutputs: MediaOutput[];
	sessionId: string;
	getMediaSrc: (path: string, sessionId: string) => string;
}) => {
	const [selectedIndex, setSelectedIndex] = useState(0);

	if (!mediaOutputs || mediaOutputs.length === 0) return null;

	// Single media file - show directly
	if (mediaOutputs.length === 1) {
		return (
			<MainMediaPlayer
				getMediaSrc={getMediaSrc}
				media={mediaOutputs[0]}
				sessionId={sessionId}
			/>
		);
	}

	// Multiple media files - show player + playlist
	return (
		<div className="space-y-4">
			<MainMediaPlayer
				getMediaSrc={getMediaSrc}
				media={mediaOutputs[selectedIndex]}
				sessionId={sessionId}
			/>
			<PlaylistSidebar
				mediaOutputs={mediaOutputs}
				onSelect={setSelectedIndex}
				selectedIndex={selectedIndex}
				sessionId={sessionId}
			/>
		</div>
	);
};
