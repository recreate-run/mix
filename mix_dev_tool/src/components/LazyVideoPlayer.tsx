import { useEffect, useRef, useState } from "react";
import type { MediaOutput } from "@/types/media";
import { VideoPlayer } from "./video-player";

interface LazyVideoPlayerProps {
	media: MediaOutput;
	sessionId: string;
}

export const LazyVideoPlayer = ({ media, sessionId }: LazyVideoPlayerProps) => {
	const [isVisible, setIsVisible] = useState(false);
	const containerRef = useRef<HTMLDivElement>(null);

	useEffect(() => {
		// Reset visibility state when media changes
		setIsVisible(false);

		const observer = new IntersectionObserver(
			([entry]) => {
				if (entry.isIntersecting) {
					setIsVisible(true);
					observer.disconnect(); // Stop observing once loaded
				}
			},
			{ threshold: 0.1 }, // Trigger when 10% visible
		);

		if (containerRef.current) {
			observer.observe(containerRef.current);
		}

		return () => observer.disconnect();
	}, []);

	if (!media.path) {
		return null;
	}

	return (
		<div className="min-h-[200px]" ref={containerRef}>
			{isVisible ? (
				<VideoPlayer
					duration={media.duration}
					path={media.path}
					sessionId={sessionId}
					startTime={media.startTime}
					title=""
				/>
			) : (
				<div className="flex h-48 items-center justify-center rounded-md bg-stone-700/30">
					<div className="animate-pulse">
						<div className="mb-2 h-6 w-32 rounded bg-stone-600" />
						<div className="h-4 w-48 rounded bg-stone-600" />
					</div>
				</div>
			)}
		</div>
	);
};
