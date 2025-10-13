// Animation configuration with known fields + dynamic parameters
export interface AnimationConfig {
	url: string;
	aspectRatio?: string | number;
	aspect?: string | number;
	duration?: number;
	text?: string;
	overlayText?: string;
	displayText?: string;
	textColor?: string;
	style?: { color?: string; [key: string]: unknown };
	[key: string]: unknown; // Allow dynamic parameters from schema
}

export type MediaOutput = {
	path: string;
	type: "image" | "video" | "audio" | "gsap_animation" | "pdf" | "csv";
	title: string;
	description?: string;
	startTime?: number;
	duration?: number;
	config?: AnimationConfig;
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
