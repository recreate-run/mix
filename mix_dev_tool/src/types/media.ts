export type MediaOutput = {
	path?: string;
	data?: string; // Inline content for markdown/json/status types
	type:
		| "image"
		| "video"
		| "audio"
		| "pdf"
		| "csv"
		| "markdown"
		| "json"
		| "status";
	title: string;
	startTime?: number;
	duration?: number;
	config?: Record<string, unknown>;
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
