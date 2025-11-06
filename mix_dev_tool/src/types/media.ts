export type MediaOutput = {
	path: string;
	type: "image" | "video" | "audio" | "pdf" | "csv" | "markdown" | "code";
	title: string;
	description?: string;
	startTime?: number;
	duration?: number;
	config?: { language?: string };
	sourceVideo?: string;
};

export interface VideoPlayerProps {
	path: string;
	title: string;
	description?: string;
	startTime?: number;
	duration?: number;
	sessionId: string;
}
