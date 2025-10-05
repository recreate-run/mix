import { IconDownload } from "@tabler/icons-react";
import { useQueryClient } from "@tanstack/react-query";
import { useNavigate } from "@tanstack/react-router";
import { type FormEventHandler, useEffect, useRef, useState } from "react";
import { Button } from "@/components/ui/button";
import {
	AIInput,
	AIInputSubmit,
	AIInputTextarea,
	AIInputToolbar,
	AIInputTools,
} from "@/components/ui/kibo-ui/ai/input";
import { useFileReference } from "@/hooks/useFileReference";
import { useMessageHistoryNavigation } from "@/hooks/useMessageHistoryNavigation";
// import { useAppList } from '@/hooks/useOpenApps';
import { usePersistentSSE } from "@/hooks/usePersistentSSE";
import { formatCurrentModel, usePreferences } from "@/hooks/usePreferences";
import { useRewindSession } from "@/hooks/useRewindSession";
import { useActiveSession, useCreateSession } from "@/hooks/useSession";
import { useSessionExport } from "@/hooks/useSessionExport";
import { useSessionMessages } from "@/hooks/useSessionMessages";
import { CACHE_KEYS } from "@/lib/cache-keys";
import { useBoundStore } from "@/stores";
// import type { ToolCall } from '@/types/common';
// import type { MediaOutput } from '@/types/media';
import { buildSessionFileUrl } from "@/utils/attachmentUtils";
import { getDisplayTitle } from "@/utils/sessionUtils";
import {
	handleSlashCommandNavigation,
	shouldShowSlashCommands,
	slashCommands,
} from "@/utils/slash-commands";
import { AttachmentPreview } from "./attachment-preview";
import { CommandFileReference } from "./command-file-reference";
import { CommandSlash } from "./command-slash";
import { ConversationDisplay } from "./conversation-display";
import { FileUploadButton } from "./file-upload-button";
import { PermissionDialog } from "./permission-dialog";

interface ChatAppProps {
	sessionId: string;
}

export function ChatApp({ sessionId }: ChatAppProps) {
	// Core conversation state
	const [text, setText] = useState<string>("");

	// Feedback notification state
	const [feedbackMessage, setFeedbackMessage] = useState<string | null>(null);

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
	const sessionMessages = useSessionMessages(sessionId);
	const sseStream = usePersistentSSE(sessionId);
	// const { apps: openApps } = useAppList();
	const rewindSession = useRewindSession();
	const createSession = useCreateSession();
	const exportSessionMutation = useSessionExport();
	const navigate = useNavigate();
	const queryClient = useQueryClient();
	const { data: preferences } = usePreferences();

	// Handle session changes: clear UI state when switching sessions
	useEffect(() => {
		if (session?.id && session.id !== previousSessionIdRef.current) {
			// Clear input when switching to a different session (but not on initial load)
			if (previousSessionIdRef.current !== "") {
				setText("");
				clearAttachments();
				interruptedMessageAddedRef.current = false;
			}
			previousSessionIdRef.current = session.id;

			// Invalidate preferences to fetch fresh data for the new session
			queryClient.invalidateQueries({ queryKey: CACHE_KEYS.preferences });
		}
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [session?.id]);

	// Handle navigation to newly created sessions
	useEffect(() => {
		if (
			sseStream.newlyCreatedSessionId &&
			sseStream.newlyCreatedSessionId !== sessionId
		) {
			// Navigate to the newly created session
			navigate({
				to: "/$sessionId",
				params: { sessionId: sseStream.newlyCreatedSessionId },
				replace: true,
			});

			// Clear the state after navigation
			sseStream.clearNewlyCreatedSession();
		}
	}, [
		sseStream.newlyCreatedSessionId,
		sessionId,
		navigate,
		sseStream.clearNewlyCreatedSession,
	]);

	// Handle streaming completion: invalidate cache, then clear streaming UI
	useEffect(() => {
		if (
			sseStream.completed &&
			!sseStream.processing &&
			(sseStream.finalContent || sseStream.toolCalls.length > 0) &&
			session?.id
		) {
			// First, invalidate cache to fetch fresh messages from backend
			queryClient.invalidateQueries({
				queryKey: CACHE_KEYS.sessionMessages(session.id),
			});
		}
	}, [
		sseStream.completed,
		sseStream.processing,
		sseStream.finalContent,
		sseStream.toolCalls.length,
		session?.id,
		queryClient,
	]);

	// Clear streaming UI after cache refetch completes (runs after above effect)
	useEffect(() => {
		if (
			sseStream.completed &&
			!sseStream.processing &&
			!sessionMessages.isLoading &&
			sessionMessages.data
		) {
			// Cache now has fresh data - safe to clear streaming UI
			sseStream.clearStreamingContent();
		}
	}, [
		sseStream.completed,
		sseStream.processing,
		sessionMessages.isLoading,
		sessionMessages.data,
		sseStream.clearStreamingContent,
	]);

	// Apps functionality removed - UI attachment system is separate from API fields

	const fileRef = useFileReference(text, setText, session?.id);

	// Handle file upload success
	const handleFileUploadSuccess = (fileName: string) => {
		// Add file reference to text input (same behavior as "@" menu)
		const displayReference = `@${fileName}`;
		const newText = text
			? `${text} ${displayReference} `
			: `${displayReference} `;
		setText(newText);

		// Add reference mapping with full URL to ensure consistency with media array
		const fullUrl = buildSessionFileUrl(session!.id, fileName);
		useBoundStore.getState().addReference(displayReference, fullUrl);

		// Show success notification
		setFeedbackMessage(`File uploaded successfully: ${fileName}`);
		setTimeout(() => setFeedbackMessage(null), 3000);
	};

	// Handle file upload error
	const handleFileUploadError = (error: string) => {
		// Show error feedback with specific error message
		setFeedbackMessage(`Error: File upload failed - ${error}`);

		// Auto-hide after 3 seconds
		setTimeout(() => {
			setFeedbackMessage(null);
		}, 3000);

		// Don't add error message to chat interface
		// Just show the notification
	};

	// Initialize new hooks
	const historyNavigation = useMessageHistoryNavigation({
		text,
		setText,
		batchSize: 50,
	});

	// Simple auto-scroll to last user message
	const userMessageRefs = useRef<(HTMLDivElement | null)[]>([]);
	const messages = sessionMessages.data || [];

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

	// Handle paste events to detect video URLs
	const handlePaste = (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
		const pastedText = e.clipboardData.getData("text");

		// Only process if pasted content might contain URLs
		if (
			pastedText.includes("http") ||
			pastedText.includes("youtu") ||
			pastedText.includes("vimeo")
		) {
			const textarea = e.currentTarget;
			const selectionStart = textarea.selectionStart;
			const selectionEnd = textarea.selectionEnd;
			const currentText = text; // Use React state for reliability

			// Calculate what the text will be after paste operation
			const finalText =
				currentText.substring(0, selectionStart) +
				pastedText +
				currentText.substring(selectionEnd);

			// Use setTimeout to avoid blocking the paste operation
			setTimeout(() => {
				useBoundStore.getState().addUrlAttachments(finalText);
			}, 0);
		}
	};

	const handleTextChange = (value: string) => {
		setText(value);

		// Don't reset cancelled state while typing - let it persist until new message is submitted
		// This keeps the cancelled message visible while user is composing their next message

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
			() => {
				setShowSlashCommands(false);
				setSelectedCommandIndex(0);
				setText(text.slice(0, -1));
				setShowCommands(true);
			},
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
				setShowCommands(false);
				setShowSlashCommands(false);
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

	// Handle completion of streaming - now handled by enhanced hook
	useEffect(() => {
		if (
			sseStream.completed &&
			(sseStream.finalContent || sseStream.toolCalls.length > 0) &&
			!sseStream.processing
		) {
			// Reset interrupted message guard when processing completes
			interruptedMessageAddedRef.current = false;
		}
	}, [sseStream.completed, sseStream.finalContent, sseStream.processing]);

	// Handle streaming errors - simple and clean
	useEffect(() => {
		if (
			sseStream.error &&
			!sseStream.error.includes("cancelled") &&
			!sseStream.cancelled
		) {
			setFeedbackMessage(`Error: Failed to send prompt - ${sseStream.error}`);
			setTimeout(() => setFeedbackMessage(null), 5000);
		}
	}, [sseStream.error, sseStream.cancelled]);

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

		// Reset interrupted message guard for new message
		interruptedMessageAddedRef.current = false;

		// Clear input immediately - optimistic UI update
		setText("");
		clearAttachments();

		try {
			// Use clean submitMessage method
			await sseStream.submitMessage({
				text: messageText,
				attachments,
				referenceMap,
				planMode:
					overridePlanMode !== undefined ? overridePlanMode : isPlanMode,
			});

			// Don't invalidate cache immediately - optimistic UI will show the message
			// Cache will be invalidated when streaming completes (see useEffect above)
		} catch (error) {
			// Restore input on error
			setText(messageText);
			// Note: attachments are already cleared, would need more complex state management to restore them
			console.error("Failed to submit message:", error);
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
			// Note: "Execution paused" will be shown in the streaming section
			// Don't add it to permanent messages to avoid ordering conflicts
		} catch (error) {
			console.error("Failed to cancel message:", error);
		}
	};

	// Handle new session creation
	const handleNewSession = async () => {
		try {
			// Create a new session
			const newSession = await createSession.mutateAsync({
				title: "New Session",
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

	// Handle editing (rewinding) conversation to a specific message
	const handleEditMessage = async (messageIndex: number) => {
		const messageToEdit = messages[messageIndex];
		if (
			!messageToEdit ||
			messageToEdit.from !== "user" ||
			!session?.id ||
			!messageToEdit.id
		) {
			return;
		}

		// Get the message BEFORE the one we want to edit (to rewind to that point)
		const previousMessageIndex = messageIndex - 1;
		if (previousMessageIndex < 0) {
			// If this is the first message, we need to clear the entire session
			// For now, just pre-populate and let user know they need to delete messages manually
			setText(messageToEdit.content);
			setFeedbackMessage("This is the first message. Edit and resubmit.");
			setTimeout(() => setFeedbackMessage(null), 3000);
			return;
		}

		const previousMessage = messages[previousMessageIndex];
		if (!previousMessage?.id) {
			return;
		}

		try {
			// Rewind to the message BEFORE the one we want to edit
			// This deletes the message we're editing and everything after it
			await rewindSession.mutateAsync({
				sessionId: session.id,
				messageId: previousMessage.id,
				cleanupMedia: true,
			});

			// Pre-populate input with the message content for editing
			setText(messageToEdit.content);

			// Copy attachments if any
			if (messageToEdit.attachments && messageToEdit.attachments.length > 0) {
				// TODO: Handle attachments if needed
			}

			// Show success feedback
			setFeedbackMessage("Ready to edit message");
			setTimeout(() => {
				setFeedbackMessage(null);
			}, 2000);
		} catch (error) {
			console.error("Failed to rewind conversation:", error);
			setFeedbackMessage("Error: Failed to rewind conversation");
			setTimeout(() => {
				setFeedbackMessage(null);
			}, 3000);
		}
	};

	// Handle session export
	const handleExport = async () => {
		if (!session?.id || exportSessionMutation.isPending) return;

		try {
			await exportSessionMutation.mutateAsync({
				sessionId: session.id,
				sessionTitle: getDisplayTitle(session),
			});
		} catch (error) {
			console.error("Failed to export session:", error);
		}
	};

	// Button status and disabled state now computed by enhanced hook
	const isSubmitDisabled =
		sseStream.buttonStatus === "ready"
			? (!text && attachments.length === 0) ||
				!session?.id ||
				sessionLoading ||
				sseStream.isSubmitDisabled
			: sseStream.buttonStatus === "paused"
				? true // Disable button completely during cancellation
				: !session?.id || sessionLoading || sseStream.isSubmitDisabled;

	return (
		<div className="relative flex h-full w-full p-8">
			<div className="flex-1 overflow-y-auto">
				{/* Feedback message notification */}
				{feedbackMessage && (
					<div
						className={`-translate-x-1/2 fade-in slide-in-from-top-5 fixed top-4 left-1/2 z-50 transform animate-in rounded-md px-4 py-2 shadow-md duration-300 ${
							feedbackMessage.startsWith("Error:")
								? "bg-destructive text-destructive-foreground"
								: "bg-primary text-primary-foreground"
						}`}
					>
						{feedbackMessage}
					</div>
				)}

				<div className="@container/main px mx-auto mt-4 flex max-w-4xl flex-1 flex-col gap-2 pb-24">
					{/* Session header with export button */}
					{session && (
						<div className="mb-4 flex items-center justify-end">
							<Button
								className="ml-2"
								disabled={exportSessionMutation.isPending}
								onClick={handleExport}
								size="sm"
								title="Export session transcript"
								variant="ghost"
							>
								<IconDownload className="h-4 w-4" />
							</Button>
						</div>
					)}

					{/* Loading indicator for messages */}
					{sessionMessages.isLoading && (
						<div className="flex items-center justify-center p-4 text-muted-foreground">
							<div className="flex items-center gap-2">
								<div className="h-4 w-4 animate-spin rounded-full border-2 border-muted border-t-foreground" />
								Loading messages...
							</div>
						</div>
					)}

					{/* Error display for messages */}
					{sessionMessages.error && (
						<div className="flex items-center justify-center p-4 text-destructive">
							<div className="rounded-md bg-destructive/10 p-4">
								Failed to load messages: {sessionMessages.error.message}
							</div>
						</div>
					)}

					{/* Conversation Display */}
					{!sessionMessages.error && (
						<ConversationDisplay
							messages={messages}
							onEditMessage={handleEditMessage}
							onPlanAction={handlePlanAction}
							sessionId={session?.id}
							setUserMessageRef={setUserMessageRef}
							sseStream={sseStream}
						/>
					)}
				</div>
			</div>

			{/* AI Input Section - Fixed at bottom with sidebar awareness */}
			<div className="absolute right-0 bottom-0 left-0 z-50 p-4 before:pointer-events-none before:absolute before:top-[-60px] before:right-0 before:left-0 before:h-16 before:from-transparent before:to-black/50 before:content-[''] ">
				<div className="relative mx-auto max-w-4xl border-none">
					{session?.id && (
						<AttachmentPreview
							attachments={attachments}
							onTextChange={setText}
							referenceMap={referenceMap}
							sessionId={session.id}
							text={text}
						/>
					)}

					{session?.id && (
						<AIInput
							className="border bg-stone-200/60 backdrop-blur-xl dark:bg-stone-700/60"
							onSubmit={handleSubmit}
						>
							<AIInputTextarea
								autoFocus
								availableCommands={slashCommands.map((cmd) => cmd.name)}
								availableFiles={fileRef.files.map((file) => file.name)}
								onChange={(e) => {
									handleTextChange(e.target.value);
									if (!inputElement) {
										setInputElement(e.target);
									}
								}}
								onKeyDown={handleKeyDown}
								onPaste={handlePaste}
								value={text}
							/>
							<AIInputToolbar>
								<AIInputTools>
									<div className="absolute bottom-1 left-1 flex">
										{/* File Upload Button */}
										<FileUploadButton
											className="ml-1"
											onUploadError={handleFileUploadError}
											onUploadSuccess={handleFileUploadSuccess}
											sessionId={session.id}
										/>

										{/* Plan Mode selection, hidden for now */}
										{/* <Select
                        onValueChange={(value) => setIsPlanMode(value === 'plan')}
                        value={isPlanMode ? 'plan' : 'edit'}
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
                      </Select> */}
									</div>

									{/* Current Model Display */}
									<div className="absolute right-14 bottom-1 text-muted-foreground text-xs">
										{formatCurrentModel(preferences)}
									</div>
								</AIInputTools>
								<AIInputSubmit
									disabled={isSubmitDisabled}
									onPauseClick={handleCancelClick}
									status={sseStream.buttonStatus}
								/>
							</AIInputToolbar>
						</AIInput>
					)}

					{/* Unified Command System */}
					{showCommands && (
						<CommandSlash
							onClose={() => {
								// Close the command palette UI
								setShowCommands(false);
								setShowSlashCommands(false);
							}}
							onFeedbackMessage={setFeedbackMessage}
							onNewSession={handleNewSession}
							onQueryClientInvalidate={(keys) =>
								queryClient.invalidateQueries({ queryKey: keys })
							}
							onSubmitMessage={submitMessage}
							sessionId={sessionId}
						/>
					)}

					{/* File Reference Dropdown with Command Component */}
					{fileRef.show && session?.id && (
						<CommandFileReference
							fileRef={fileRef}
							onClose={fileRef.close}
							sessionId={session.id}
						/>
					)}
				</div>
			</div>

			{/* Permission Dialog - Show the first pending permission request */}
			{sseStream.permissionRequests.length > 0 && (
				<PermissionDialog
					onClose={() => {
						// Safely check if permission request still exists before denying
						if (sseStream.permissionRequests.length > 0) {
							sseStream.denyPermission(sseStream.permissionRequests[0].id);
						}
					}}
					onDeny={sseStream.denyPermission}
					onGrant={sseStream.grantPermission}
					permissionRequest={sseStream.permissionRequests[0]}
				/>
			)}
		</div>
	);
}
