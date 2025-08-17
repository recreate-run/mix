import { useNavigate } from "@tanstack/react-router";
import {
	type FormEventHandler,
	useEffect,
	useMemo,
	useRef,
	useState,
} from "react";
import {
	AIInput,
	AIInputSubmit,
	AIInputTextarea,
	AIInputToolbar,
	AIInputTools,
} from "@/components/ui/kibo-ui/ai/input";
import type { AIToolStatus } from "@/components/ui/kibo-ui/ai/tool";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { useFileReference } from "@/hooks/useFileReference";
import { useForkSession } from "@/hooks/useForkSession";
import { useMessageHistoryNavigation } from "@/hooks/useMessageHistoryNavigation";
import { useAppList } from "@/hooks/useOpenApps";
import { usePersistentSSE } from "@/hooks/usePersistentSSE";
import { useActiveSession, useCreateSession } from "@/hooks/useSession";
import { useSessionMessages } from "@/hooks/useSessionMessages";
import {
	type Attachment,
	expandFileReferences,
	reconstructAttachmentsFromHistory,
} from "@/stores/attachmentSlice";
import { useBoundStore } from "@/stores";
import {
	handleSlashCommandNavigation,
	shouldShowSlashCommands,
	slashCommands,
} from "@/utils/slash-commands";
import type { ToolCall } from "@/types/common";
import type { MediaOutput } from "@/types/media";
import type { UIMessage, MessageData } from "@/types/message";
import { AttachmentPreview } from "./attachment-preview";
import { CommandFileReference } from "./command-file-reference";
import { CommandSlash } from "./command-slash";
import { ConversationDisplay } from "./conversation-display";
import { PermissionDialog } from "./permission-dialog";

// Helper function to check if a message contains media_showcase tool call
const hasMediaShowcaseTool = (toolCalls: any[]) => {
	return toolCalls?.some((tc) => tc.name === "media_showcase");
};

// Helper function to extract media outputs from media_showcase tool call
const getMediaShowcaseOutputs = (toolCalls: any[]): MediaOutput[] => {
	const mediaShowcaseTool = toolCalls?.find(
		(tc) => tc.name === "media_showcase",
	);
	if (!mediaShowcaseTool?.parameters?.outputs) return [];

	try {
		return mediaShowcaseTool.parameters.outputs as MediaOutput[];
	} catch {
		return [];
	}
};

interface ChatAppProps {
	sessionId: string;
}

export function ChatApp({ sessionId }: ChatAppProps) {
	// Core conversation state
	const [text, setText] = useState<string>("");
	const [messages, setMessages] = useState<UIMessage[]>([]);

	// UI Interaction Mode 1: Slash Commands (dropdown when typing "/help", "/clear" etc.)
	const [showSlashCommands, setShowSlashCommands] = useState(false);
	const [selectedCommandIndex, setSelectedCommandIndex] = useState(0);

	// UI Interaction Mode 2: Command Palette (full modal triggered by "/" alone)
	const [showCommands, setShowCommands] = useState(false);

	// Input management and focus handling
	const [inputElement, setInputElement] = useState<HTMLTextAreaElement | null>(
		null,
	);

	// Mode toggles and session management
	const [isPlanMode, setIsPlanMode] = useState(false);
	const [pendingForkText, setPendingForkText] = useState<{
		text: string;
		attachments: Attachment[];
		referenceMap: Map<string, string>;
	} | null>(null);

	// Component lifecycle refs
	const interruptedMessageAddedRef = useRef(false);
	const previousSessionIdRef = useRef<string>("");

	// UI Mode 4: File Reference (managed in useFileReference hook)
	// UI Mode 5: Normal Input (default when all others are false)

	// All attachment store hooks at top to avoid temporal dead zone
	const attachments = useBoundStore((state) => state.attachments);
	const referenceMap = useBoundStore((state) => state.referenceMap);
	const clearAttachments = useBoundStore((state) => state.clearAttachments);
	const syncWithText = useBoundStore((state) => state.syncWithText);

	const { data: session, isLoading: sessionLoading } =
		useActiveSession(sessionId);
	const sessionMessages = useSessionMessages(session?.id || null);
	const sseStream = usePersistentSSE(session?.id || "");
	const { apps: openApps } = useAppList();
	const forkSession = useForkSession();
	const createSession = useCreateSession();
	const navigate = useNavigate();

	// Clear UI state when session changes (new working directory selected)
	useEffect(() => {
		if (session?.id && session.id !== previousSessionIdRef.current) {
			// Only clear if we're switching from one session to another (not initial load)
			if (previousSessionIdRef.current !== "") {
				setText("");
				clearAttachments();
				interruptedMessageAddedRef.current = false;
			}
			previousSessionIdRef.current = session.id;
		}
	}, [session?.id]);

	// Load messages when session messages data changes
	useEffect(() => {
		if (sessionMessages.data && session?.id) {
			setMessages(sessionMessages.data);
		} else {
			setMessages([]);
		}
	}, [sessionMessages.data, session?.id]);

	// Set fork text after session switching completes
	useEffect(() => {
		if (pendingForkText && session?.id) {
			setText(pendingForkText.text);
			useBoundStore
				.getState()
				.setHistoryState(
					pendingForkText.attachments,
					pendingForkText.referenceMap,
				);
			setPendingForkText(null);
		}
	}, [pendingForkText, session?.id]);

	// Transform open apps to Attachment format and filter allowed apps
	const allowedApps = [
		"Notes",
		"Obsidian",
		"Blender",
		"Pixelmator Pro",
		"Final Cut Pro",
	];
	const availableApps = useMemo(() => {
		return openApps
			.filter((app) =>
				allowedApps.some((allowed) =>
					app.name.toLowerCase().includes(allowed.toLowerCase()),
				),
			)
			.map((app) => ({
				id: `app:${app.bundle_id}`,
				name: app.name,
				type: "app" as const,
				icon: "placeholder", // Icons loaded on-demand for performance
				isOpen: true,
				bundleId: app.bundle_id,
			}));
	}, [openApps]);

	const fileRef = useFileReference(text, setText, session?.workingDirectory);

	// Initialize new hooks
	const historyNavigation = useMessageHistoryNavigation({
		text,
		setText,
		batchSize: 50,
	});

	// Simple auto-scroll to last user message
	const userMessageRefs = useRef<(HTMLDivElement | null)[]>([]);

	useEffect(() => {
		const lastUserMessageIndex = messages.findLastIndex(
			(m) => m.from === "user",
		);
		if (
			lastUserMessageIndex !== -1 &&
			userMessageRefs.current[lastUserMessageIndex]
		) {
			setTimeout(() => {
				userMessageRefs.current[lastUserMessageIndex]?.scrollIntoView({
					behavior: "smooth",
					block: "start",
				});
			}, 100);
		}
	}, [messages, sseStream.processing]);

	const setUserMessageRef = (index: number) => (el: HTMLDivElement | null) => {
		userMessageRefs.current[index] = el;
	};

	const handleTextChange = (value: string) => {
		setText(value);

		// Reset cancelled state when user starts typing after cancellation
		if (sseStream.cancelled && value.length > 0) {
			sseStream.resetCancelledState();
		}

		// Sync media store with text changes (bidirectional sync)
		syncWithText(value);

		// Check if user just typed a slash to open Command-K menu
		if (
			value.endsWith("/") &&
			value.length > 0 &&
			value[value.length - 1] === "/"
		) {
			// Remove the slash and open Command-K menu
			setText(value.slice(0, -1));
			setShowCommands(true);
			setShowSlashCommands(false);

			return;
		}

		// Show slash commands dropdown when conditions are met
		const shouldShowDropdown =
			shouldShowSlashCommands(value) && !showCommands && !fileRef.show;

		if (shouldShowDropdown !== showSlashCommands) {
			setShowSlashCommands(shouldShowDropdown);
			if (shouldShowDropdown) {
				setSelectedCommandIndex(0);
			}
		}

		// Close command palette if no slash commands
		if (!shouldShowSlashCommands(value)) {
			setShowCommands(false);
		}
	};

	// Unified command handler
	const handleCommand = (
		action: "select" | "execute" | "close",
		data?: any,
	) => {
		switch (action) {
			case "select": {
				setShowSlashCommands(false);
				setSelectedCommandIndex(0);
				setText(text.slice(0, -1));
				setShowCommands(true);
				break;
			}
			case "execute": {
				const command = data as string;
				setShowSlashCommands(false);
				setShowCommands(false);

				if (command === "clear") {
					// Create a new session instead of just clearing UI
					handleNewSession();
					return;
				}

				submitMessage(`/${command}`);
				break;
			}
			case "close": {
				setShowSlashCommands(false);
				setShowCommands(false);

				break;
			}
		}
	};

	const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
		// Handle Shift+Tab for plan mode toggle
		if (e.key === "Tab" && e.shiftKey) {
			e.preventDefault();
			setIsPlanMode((prev) => !prev);
			return;
		}

		// Handle Enter for form submission (without shift for new line)
		if (e.key === "Enter" && !e.shiftKey) {
			e.preventDefault();
			const form = e.currentTarget.form;
			if (form) {
				form.requestSubmit();
			}
			return;
		}

		// Handle slash command navigation
		const slashHandled = handleSlashCommandNavigation(
			e,
			showSlashCommands,
			selectedCommandIndex,
			setSelectedCommandIndex,
			(command) => handleCommand("select", command),
			() => setShowSlashCommands(false),
		);
		if (slashHandled) return;

		// Handle Escape key to stop processing or close popups
		if (e.key === "Escape") {
			e.preventDefault();

			// First priority: Stop message processing if active
			if (sseStream.processing) {
				handleCancelClick();
				return;
			}

			// Second priority: Close popups
			if (fileRef.show) {
				fileRef.close();
				return;
			}
			if (showCommands) {
				handleCommand("close");
				return;
			}
		}

		// Handle history navigation when not in UI modes
		const isInUIMode = showSlashCommands || fileRef.show || showCommands;
		const historyHandled = historyNavigation.handleHistoryNavigation(
			e,
			isInUIMode,
		);
		if (historyHandled) {
			return;
		}
	};

	// Handle completion of streaming
	useEffect(() => {
		if (
			sseStream.completed &&
			(sseStream.finalContent || sseStream.toolCalls.length > 0) &&
			!sseStream.processing
		) {
			// Convert SSE tool calls to our Message format
			const convertedToolCalls: ToolCall[] = sseStream.toolCalls.map((tc) => ({
				name: tc.name,
				description: tc.description,
				status: tc.status as AIToolStatus,
				parameters: tc.parameters,
				result: tc.result,
				error: tc.error,
			}));

			setMessages((prev) => {
				const mediaOutputs = hasMediaShowcaseTool(convertedToolCalls)
					? getMediaShowcaseOutputs(convertedToolCalls)
					: undefined;

				return [
					...prev,
					{
						content: sseStream.finalContent!,
						from: "assistant",
						toolCalls:
							convertedToolCalls.length > 0 ? convertedToolCalls : undefined,
						reasoning: sseStream.reasoning || undefined,
						reasoningDuration: sseStream.reasoningDuration || undefined,
						mediaOutputs,
					},
				];
			});

			// Reset interrupted message guard when processing completes
			interruptedMessageAddedRef.current = false;
		}
	}, [
		sseStream.completed,
		sseStream.finalContent,
		sseStream.processing,
		session?.id,
	]);

	// Handle streaming errors
	useEffect(() => {
		if (sseStream.error) {
			const errorMessage = `Failed to send prompt: ${sseStream.error}`;
			setMessages((prev) => [
				...prev,
				{
					content: errorMessage,
					from: "assistant",
					frontend_only: true,
				},
			]);
		}
	}, [sseStream.error, session?.id]);

	// Declarative focus management - refocus chat input when all popups are closed
	useEffect(() => {
		if (!(showCommands || fileRef.show || showSlashCommands) && inputElement) {
			inputElement.focus();
		}
	}, [showCommands, fileRef.show, showSlashCommands, inputElement]);

	// Handle pause state changes - simplified since pausing is not implemented
	// (Keeping this for compatibility but it won't trigger since isPaused will always be false)

	const submitMessage = async (
		messageText: string,
		overridePlanMode?: boolean,
	) => {
		if (!(messageText && session?.id && sseStream.connected)) {
			return;
		}

		// Exit history mode if active
		historyNavigation.resetHistoryMode();

		// Add user message to conversation and clear input immediately
		setMessages((prev) => [
			...prev,
			{
				content: messageText,
				from: "user",
				attachments: attachments.length > 0 ? attachments : undefined,
			},
		]);
		setText("");
		clearAttachments();

		// Reset interrupted message guard for new message
		interruptedMessageAddedRef.current = false;

		// Send message via persistent SSE
		try {
			// Expand file references from display format to full paths
			const expandedText = expandFileReferences(messageText, referenceMap);

			const messageData: MessageData = {
				text: expandedText,
				media: attachments.filter((a) => a.path).map((a) => a.path!),
				apps: attachments
					.filter((a) => a.type === "app")
					.map((app) => app.name),
				plan_mode:
					overridePlanMode !== undefined ? overridePlanMode : isPlanMode,
			};
			await sseStream.sendMessage(JSON.stringify(messageData));
		} catch (error) {
			console.error("Failed to send message:", error);
			// Error will be handled by the error useEffect
		}
	};

	const handleSubmit: FormEventHandler<HTMLFormElement> = async (event) => {
		event.preventDefault();
		await submitMessage(text);
	};

	// Handle stop/cancel button clicks
	const handleCancelClick = async () => {
		try {
			await sseStream.cancelMessage();
			// Add cancellation message to conversation
			setMessages((prev) => [
				...prev,
				{
					content: "Execution paused",
					from: "assistant",
					frontend_only: true,
				},
			]);
		} catch (error) {
			console.error("Failed to cancel message:", error);
		}
	};

	// Handle new session creation
	const handleNewSession = async () => {
		try {
			// Create a new session with the current working directory
			const newSession = await createSession.mutateAsync({
				title: "New Session",
				workingDirectory: session?.workingDirectory,
			});

			// Navigate to the new session - this will automatically trigger UI updates
			navigate({
				to: "/$sessionId",
				params: { sessionId: newSession.id },
				replace: true,
			});
		} catch (error) {
			console.error("Failed to create new session:", error);
		}
	};

	// Handle plan actions from ConversationDisplay
	const handlePlanAction = (action: "proceed" | "keep-planning") => {
		if (action === "proceed") {
			setIsPlanMode(false);
			submitMessage(
				"Proceed with implementing the plan you just created. Begin implementation now.",
				false,
			);
		}
		// For 'keep-planning', no additional action needed
	};

	// Handle forking conversation at a specific message
	const handleForkMessage = async (messageIndex: number) => {
		const messageToFork = messages[messageIndex];
		if (!messageToFork || messageToFork.from !== "user" || !session?.id) {
			return;
		}

		try {
			// Call backend to fork session and copy messages
			const newSession = await forkSession.mutateAsync({
				sourceSessionId: session.id,
				messageIndex: messageIndex,
				title: `Forked: ${session.title || "Chat Session"}`,
			});

			// Extract media paths and app names from the message attachments
			const mediaPaths =
				messageToFork.attachments?.filter((a) => a.path).map((a) => a.path!) ||
				[];
			const appNames =
				messageToFork.attachments
					?.filter((a) => a.type === "app")
					.map((a) => a.name) || [];

			// Reconstruct attachment state from the historical message
			const { contractedText, attachments, referenceMap } =
				await reconstructAttachmentsFromHistory(
					messageToFork.content,
					mediaPaths,
					appNames,
				);

			// Navigate to the forked session
			navigate({
				to: "/$sessionId",
				params: { sessionId: newSession.id },
				replace: true,
			});

			// Queue fork text to be set after session switching completes
			setPendingForkText({ text: contractedText, attachments, referenceMap });
		} catch (error) {
			console.error("Failed to fork conversation:", error);
			setMessages((prev) => [
				...prev,
				{
					content: `Failed to fork conversation: ${error}`,
					from: "assistant",
					frontend_only: true,
				},
			]);
		}
	};

	// Calculate submit button status and disabled state
	const buttonStatus = sseStream.cancelling
		? "paused"
		: sseStream.cancelled
			? "streaming"
			: sseStream.processing
				? "streaming"
				: sseStream.error
					? "error"
					: "ready";

	// Ready state: need text/attachments and connection. Other states: only need connection for pause/resume
	const isSubmitDisabled =
		buttonStatus === "ready"
			? (!text && attachments.length === 0) ||
				!session?.id ||
				sessionLoading ||
				!sseStream.connected
			: buttonStatus === "paused"
				? true // Disable button completely during cancellation
				: !session?.id || sessionLoading || !sseStream.connected;

	return (
		<div className="flex fl h-full w-full">
			<div className="flex-1 overflow-y-auto">
				<div className="@container/main flex flex-1 flex-col gap-2 px mx-auto max-w-5xl mt-4 pb-24">
					{/* Conversation Display */}
					<ConversationDisplay
						messages={messages}
						onForkMessage={handleForkMessage}
						onPlanAction={handlePlanAction}
						sseStream={sseStream}
						setUserMessageRef={setUserMessageRef}
					/>

					{/* Attachment Preview Section */}
					<div className="z-20 mx-auto mb-0 w-full">
						<AttachmentPreview
							attachments={attachments}
							text={text}
							referenceMap={referenceMap}
							onTextChange={setText}
						/>
					</div>
				</div>
			</div>

			{/* AI Input Section - Fixed at bottom with sidebar awareness */}
			<div className="fixed bottom-0 left-0 right-0 z-50 px-2 pl-[calc(var(--sidebar-width,0px)+0.5rem)] before:pointer-events-none before:absolute before:top-[-60px] before:right-0 before:left-0 before:h-16 before:from-transparent before:to-black/50 before:content-[''] bg-gradient-to-t from-background/95 to-transparent ">
				<div className="relative border-none max-w-5xl mx-auto pb-4">
					<AIInput
						className="border bg-stone-200/60 dark:bg-stone-700/60 backdrop-blur-xl"
						onSubmit={handleSubmit}
					>
						<AIInputTextarea
							autoFocus
							availableApps={attachments
								.filter((a) => a.type === "app")
								.map((app) => app.name)}
							availableCommands={slashCommands.map((cmd) => cmd.name)}
							availableFiles={fileRef.files.map((file) => file.name)}
							onChange={(e) => {
								handleTextChange(e.target.value);
								if (!inputElement) {
									setInputElement(e.target);
								}
							}}
							onKeyDown={handleKeyDown}
							value={text}
						/>
						<AIInputToolbar>
							<AIInputTools>
								<div className="absolute bottom-1 left-1">
									<Select
										onValueChange={(value) => setIsPlanMode(value === "plan")}
										value={isPlanMode ? "plan" : "edit"}
									>
										<SelectTrigger
											className="border-none bg-transparent text-muted-foreground hover:bg-transparent focus:border-none focus:ring-0 dark:bg-transparent hover:dark:bg-transparent"
											size="sm"
										>
											<SelectValue />
										</SelectTrigger>
										<SelectContent>
											<SelectItem value="edit">create</SelectItem>
											<SelectItem value="plan">plan</SelectItem>
										</SelectContent>
									</Select>
								</div>
							</AIInputTools>
							<AIInputSubmit
								disabled={isSubmitDisabled}
								onPauseClick={handleCancelClick}
								status={buttonStatus}
							/>
						</AIInputToolbar>
					</AIInput>

					{/* Unified Command System */}
					{showCommands && (
						<CommandSlash
							onClose={() => handleCommand("close")}
							onExecuteCommand={(command) => handleCommand("execute", command)}
							sessionId={sessionId}
						/>
					)}

					{/* File Reference Dropdown with Command Component */}
					{fileRef.show && (
						<CommandFileReference
							apps={availableApps}
							fileRef={fileRef}
							text={text}
							onClose={fileRef.close}
							onTextUpdate={setText}
						/>
					)}
				</div>
			</div>

			{/* Permission Dialog - Show the first pending permission request */}
			{sseStream.permissionRequests.length > 0 && (
				<PermissionDialog
					permissionRequest={sseStream.permissionRequests[0]}
					onGrant={sseStream.grantPermission}
					onDeny={sseStream.denyPermission}
					onClose={() => {
						// Safely check if permission request still exists before denying
						if (sseStream.permissionRequests.length > 0) {
							sseStream.denyPermission(sseStream.permissionRequests[0].id);
						}
					}}
				/>
			)}
		</div>
	);
}
