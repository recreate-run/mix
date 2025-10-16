import type {
	Callback,
	SessionType,
	SubagentType,
} from "mix-typescript-sdk/models/sessiondata.js";
import type { AIToolStatus } from "@/components/ui/kibo-ui/ai/tool";

export type ToolCall = {
	id: string;
	name: string;
	description: string;
	status: AIToolStatus;
	parameters: Record<string, unknown>;
	result?: string;
	error?: string;
};

export interface Session {
	id: string;
	title: string;
}

// Canonical SessionData interface - matches backend contract exactly
export interface SessionData {
	id: string;
	title: string;
	sessionType: SessionType;
	subagentType?: SubagentType;
	parentSessionId?: string;
	parentToolCallId?: string;
	userMessageCount: number;
	assistantMessageCount: number;
	toolCallCount: number;
	promptTokens: number;
	completionTokens: number;
	cost: number;
	createdAt: Date; // Parsed from backend RFC3339 date string
	firstUserMessage?: string;
	callbacks?: Callback[];
	// Client-side only properties for UI state
	isDeleting?: boolean;
}

// Utility functions for message counts
export const getTotalMessages = (session: SessionData): number => {
	return (
		session.userMessageCount +
		session.assistantMessageCount +
		session.toolCallCount
	);
};

export const getExchangeCount = (session: SessionData): number => {
	return session.userMessageCount + session.assistantMessageCount;
};

// Smart message count formatting - centralized logic for consistent display
export const formatMessageCounts = (session: SessionData): string => {
	const { toolCallCount } = session;

	if (toolCallCount === 0) {
		// No tools used - show simple message count
		const total = getExchangeCount(session);
		return `${total} messages`;
	}

	// Tools were used - show exchanges and tools separately
	const exchanges = getExchangeCount(session);
	return `${exchanges} exchanges, ${toolCallCount} tools`;
};

export interface ToolCallData {
	id: string;
	name: string;
	input: string;
	type: string;
	finished: boolean;
	result?: string;
	isError?: boolean;
}
