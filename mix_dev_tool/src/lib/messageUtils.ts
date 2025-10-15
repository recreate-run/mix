import type { BackendMessage } from "mix-typescript-sdk/models";
import { mix } from "@/lib/mix-sdk";
import type { Attachment } from "@/stores/attachmentSlice";
import type { ToolCall, ToolCallData } from "@/types/common";
import type { MediaOutput } from "@/types/media";
import type { TimelineEntry, UIMessage } from "@/types/message";
import {
	createFileAttachment,
	createFolderAttachment,
} from "@/utils/attachmentUtils";
import { PlatformFeatures } from "@/utils/platform";

interface ParsedContent {
	text: string;
	media: string[];
}

const extractContentData = (content: string): ParsedContent => {
	try {
		const parsed = JSON.parse(content);
		return {
			text: parsed.text || content,
			media: parsed.media || [],
		};
	} catch {
		// Extract media URLs from plain text
		const mediaUrlRegex =
			/https?:\/\/[^\s]+\/api\/sessions\/[^\s]+\/files\/[^\s]+/g;
		const mediaUrls = content.match(mediaUrlRegex) || [];

		// Remove media URLs from text to get clean text
		const cleanText = content;
		// mediaUrls.forEach(url => {
		//   cleanText = cleanText.replace(url, '').trim();
		// });

		return {
			text: cleanText,
			media: mediaUrls,
		};
	}
};

const convertMediaToAttachments = async (
	mediaPaths: string[],
): Promise<Attachment[]> => {
	const attachments: Attachment[] = [];

	for (const mediaPath of mediaPaths) {
		try {
			let attachment: Attachment | null = null;

			// Check if this is a server URL (from reloaded session) - can be full URL or relative path
			if (
				(mediaPath.startsWith("http") ||
					mediaPath.startsWith("/api/sessions/")) &&
				mediaPath.includes("/api/sessions/") &&
				mediaPath.includes("/files/")
			) {
				// Extract filename from server URL
				const urlParts = mediaPath.split("/");
				const filename = decodeURIComponent(urlParts[urlParts.length - 1]);

				// Create attachment with just the filename as path
				attachment = createFileAttachment(filename);
			} else {
				// Handle local file paths (during upload)
				if (PlatformFeatures.hasFileSystemAccess()) {
					// Desktop: Check if directory using stat
					try {
						const { stat } = await import("@tauri-apps/plugin-fs");
						const fileStat = await stat(mediaPath);
						if (fileStat.isDirectory) {
							attachment = await createFolderAttachment(mediaPath);
						} else {
							attachment = createFileAttachment(mediaPath);
						}
					} catch (_statError) {
						// If stat fails, try to create as file based on file extension
						attachment = createFileAttachment(mediaPath);
					}
				} else {
					// Browser: Treat as file (folders not supported)
					attachment = createFileAttachment(mediaPath);
				}
			}

			if (attachment) {
				attachments.push(attachment);
			}
		} catch (error) {
			console.warn(`Failed to create attachment for ${mediaPath}:`, error);
		}
	}

	return attachments;
};

const convertToolCallsToUI = (toolCalls: ToolCallData[]): ToolCall[] => {
	return toolCalls.map((tc) => {
		let parameters: Record<string, unknown> = {};
		try {
			parameters = JSON.parse(tc.input || "{}");
		} catch {
			// If input is not valid JSON, treat as empty parameters
			parameters = {};
		}

		return {
			id: tc.id,
			name: tc.name,
			description: tc.name, // Use name as description since we don't have a separate description
			status: tc.finished ? "completed" : "pending",
			parameters,
			result: tc.result || undefined,
			error: tc.isError ? tc.result : undefined,
		};
	});
};

const convertBackendMessageToUI = async (
	backendMessage: BackendMessage,
	subagentTimeline?: Map<string, TimelineEntry[]>, // Map of tool call ID -> timeline entries from that subagent
): Promise<UIMessage> => {
	const { text, media } = extractContentData(backendMessage.userInput);

	// Convert media paths to attachments
	const attachments = await convertMediaToAttachments(media);

	// Convert tool calls if present
	const toolCalls = backendMessage.toolCalls
		? convertToolCallsToUI(backendMessage.toolCalls)
		: undefined;

	// Extract media outputs from show_media tool calls
	const mediaOutputs = toolCalls?.find((tc) => tc.name === "show_media")
		?.parameters?.outputs as MediaOutput[] | undefined;

	// Build timeline from stored data
	const timeline: TimelineEntry[] = [];

	// Add reasoning block if available
	if (backendMessage.reasoning?.trim()) {
		timeline.push({
			type: "thinking",
			timestamp: Date.now(),
			content: backendMessage.reasoning,
			id: `stored-reasoning-${backendMessage.id}`,
		});
	}

	// Add tool calls to timeline, with their subagent events nested underneath
	if (toolCalls && toolCalls.length > 0) {
		for (const tc of toolCalls) {
			timeline.push({
				type: "tool",
				timestamp: Date.now(),
				content: tc,
				id: tc.id,
			});

			// Add subagent events for THIS specific tool call only
			if (subagentTimeline?.has(tc.id)) {
				const subagentEvents = subagentTimeline.get(tc.id);
				if (subagentEvents) {
					timeline.push(...subagentEvents);
				}
			}
		}
	}

	return {
		id: backendMessage.id,
		content: text,
		from: backendMessage.role === "user" ? "user" : "assistant",
		toolCalls: toolCalls && toolCalls.length > 0 ? toolCalls : undefined,
		attachments: attachments.length > 0 ? attachments : undefined,
		timeline: timeline.length > 0 ? timeline : undefined,
		mediaOutputs:
			mediaOutputs && mediaOutputs.length > 0 ? mediaOutputs : undefined,
		reasoning: backendMessage.reasoning,
		reasoningDuration: backendMessage.reasoningDuration,
	};
};

export const convertBackendMessagesToUI = async (
	backendMessages: BackendMessage[],
	parentSessionId?: string, // Optional: for loading subagent timeline
): Promise<UIMessage[]> => {
	// Load subagent timeline entries if parentSessionId provided
	let subagentTimeline: Map<string, TimelineEntry[]> | undefined;

	if (parentSessionId) {
		try {
			subagentTimeline = await loadSubagentTimeline(parentSessionId);
		} catch (error) {
			console.error("Failed to load subagent timeline:", error);
			// Continue without subagent timeline rather than failing entirely
		}
	}

	const uiMessages: UIMessage[] = [];

	for (const backendMessage of backendMessages) {
		const uiMessage = await convertBackendMessageToUI(
			backendMessage,
			subagentTimeline,
		);
		uiMessages.push(uiMessage);
	}

	return uiMessages;
};

// Load timeline entries from all subagent sessions for a given parent session
async function loadSubagentTimeline(
	parentSessionId: string,
): Promise<Map<string, TimelineEntry[]>> {
	const timelineMap = new Map<string, TimelineEntry[]>();

	// Load all sessions to find subagents of this parent
	// IMPORTANT: Include subagents in the response to enable nested timeline loading
	const sessions = await mix.sessions.list({ includeSubagents: true });
	const subagentSessions = sessions.filter(
		(s) =>
			s.parentSessionId === parentSessionId && s.sessionType === "subagent",
	);

	// Load messages for each subagent session
	for (const subagentSession of subagentSessions) {
		if (!subagentSession.parentToolCallId) {
			console.warn(
				`Subagent session ${subagentSession.id} missing parentToolCallId`,
			);
			continue;
		}

		try {
			const messages = await mix.messages.getSession({
				id: subagentSession.id,
			});

			// Convert subagent messages to timeline entries
			const entries: TimelineEntry[] = [];

			for (const msg of messages) {
				// Add thinking entries
				if (msg.reasoning?.trim()) {
					entries.push({
						type: "thinking",
						timestamp: Date.now(),
						content: msg.reasoning,
						id: `subagent-thinking-${msg.id}`,
						parentToolCallId: subagentSession.parentToolCallId,
					});
				}

				// Add tool call entries
				if (msg.toolCalls) {
					const toolCalls = convertToolCallsToUI(msg.toolCalls);
					for (const tc of toolCalls) {
						entries.push({
							type: "tool",
							timestamp: Date.now(),
							content: tc,
							id: `subagent-tool-${tc.id}`,
							parentToolCallId: subagentSession.parentToolCallId,
						});
					}
				}

				// Add content entries for assistant responses
				if (msg.role === "assistant" && msg.assistantResponse?.trim()) {
					entries.push({
						type: "content",
						timestamp: Date.now(),
						content: msg.assistantResponse,
						id: `subagent-content-${msg.id}`,
						parentToolCallId: subagentSession.parentToolCallId,
					});
				}
			}

			// Store entries mapped by parent tool call ID
			const existingEntries =
				timelineMap.get(subagentSession.parentToolCallId) || [];
			timelineMap.set(subagentSession.parentToolCallId, [
				...existingEntries,
				...entries,
			]);
		} catch (error) {
			console.error(
				`Failed to load messages for subagent ${subagentSession.id}:`,
				error,
			);
		}
	}

	return timelineMap;
}
