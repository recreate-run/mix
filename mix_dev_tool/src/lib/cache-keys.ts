export const CACHE_KEYS = {
	sessions: ["sessions"] as const,
	session: (id: string) => ["sessions", "session", id] as const,
	sessionMessages: (id: string) => ["sessions", "messages", id] as const,
	sessionFiles: (id: string) => ["sessions", "files", id] as const,
	messageHistory: ["messageHistory"] as const,
	preferences: ["preferences"] as const,
	tools: ["tools"] as const,
	toolsStatus: ["tools", "status"] as const,
	animationSchema: (animationName: string) =>
		["gsap", "animation", "schema", animationName] as const,
	animationList: ["gsap", "animations"] as const,
	csvData: (url: string) => ["csv", "data", url] as const,
} as const;
