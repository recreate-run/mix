import type { Attachment } from "@/stores/attachmentSlice";
import type { ToolCall } from "./common";
import type { MediaOutput } from "./media";
import type {
	HierarchicalModelData,
	ModelInfo,
	ProviderInfo,
} from "./provider";

// Login-specific provider info that requires authMethods
export interface LoginProviderInfo {
	id: string;
	displayName: string;
	authMethods: ("api_key" | "oauth")[];
	authenticated: boolean;
	apiKeyFormat?: string;
	isPreferred?: boolean;
}

export type TimelineEntry =
	| {
			type: "thinking";
			timestamp: number;
			content: string;
			id: string;
	  }
	| {
			type: "tool";
			timestamp: number;
			content: ToolCall;
			id: string;
	  }
	| {
			type: "content";
			timestamp: number;
			content: string;
			id: string;
	  };

export interface UIMessage {
	id?: string;
	content: string;
	from: "user" | "assistant";
	frontend_only?: boolean;
	toolCalls?: ToolCall[];
	attachments?: Attachment[];
	timeline?: TimelineEntry[];
	mediaOutputs?: MediaOutput[];
	reasoning?: string;
	reasoningDuration?: number;
	loginData?: {
		providers: LoginProviderInfo[];
		hasExistingPreferences?: boolean;
		oauthState?: string;
	};
	status?: {
		providers: ProviderInfo[];
		hasAuthenticatedProvider: boolean;
	};
	statusData?: {
		providers: ProviderInfo[];
	};
	provider?: {
		providers: ProviderInfo[];
		currentProvider?: string;
	};
	model?: {
		models: ModelInfo[];
		currentModel?: string;
		provider: {
			id: string;
			displayName: string;
		};
	};
	hierarchicalModel?: HierarchicalModelData;
	shouldInvalidatePreferencesCache?: boolean; // Signal to the UI to invalidate the preferences cache
	suppressChatMessage?: boolean; // When true, message won't be shown in the chat interface
}
