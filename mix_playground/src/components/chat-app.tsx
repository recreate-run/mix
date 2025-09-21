import { useNavigate } from '@tanstack/react-router';
import { useQueryClient } from '@tanstack/react-query';
import {
  type FormEventHandler,
  useEffect,
  useRef,
  useState,
} from 'react';
import {
  AIInput,
  AIInputSubmit,
  AIInputTextarea,
  AIInputToolbar,
  AIInputTools,
} from '@/components/ui/kibo-ui/ai/input';
import { useFileReference } from '@/hooks/useFileReference';
import { useForkSession } from '@/hooks/useForkSession';
import { useMessageHistoryNavigation } from '@/hooks/useMessageHistoryNavigation';
// import { useAppList } from '@/hooks/useOpenApps';
import { usePersistentSSE } from '@/hooks/usePersistentSSE';
import { useActiveSession, useCreateSession } from '@/hooks/useSession';
import { useSessionMessages } from '@/hooks/useSessionMessages';
import { usePreferences, formatCurrentModel } from '@/hooks/usePreferences';
import { useBoundStore } from '@/stores';
import {
  type Attachment,
  reconstructAttachmentsFromHistory,
} from '@/stores/attachmentSlice';
import { buildSessionFileUrl } from '@/utils/attachmentUtils';
// import type { ToolCall } from '@/types/common';
// import type { MediaOutput } from '@/types/media';
import type { UIMessage } from '@/types/message';
import {
  handleSlashCommandNavigation,
  shouldShowSlashCommands,
  slashCommands,
} from '@/utils/slash-commands';
import { AttachmentPreview } from './attachment-preview';
import { CommandFileReference } from './command-file-reference';
import { CommandSlash } from './command-slash';
import { ConversationDisplay } from './conversation-display';
import { FileUploadButton } from './file-upload-button';
import { PermissionDialog } from './permission-dialog';

// Helper function to extract media outputs from show_media tool call
// const getMediaShowcaseOutputs = (toolCalls: ToolCall[]): MediaOutput[] => {
//   const mediaShowcaseTool = toolCalls?.find(
//     (tc) => tc.name === 'show_media'
//   );
//   if (!mediaShowcaseTool?.parameters?.outputs) return [];
//   try {
//     return mediaShowcaseTool.parameters.outputs as MediaOutput[];
//   } catch {
//     return [];
//   }
// };

interface ChatAppProps {
  sessionId: string;
}

export function ChatApp({ sessionId }: ChatAppProps) {
  // Core conversation state
  const [text, setText] = useState<string>('');
  const [messages, setMessages] = useState<UIMessage[]>([]);

  // Feedback notification state
  const [feedbackMessage, setFeedbackMessage] = useState<string | null>(null);

  // UI Interaction Mode 1: Slash Commands (dropdown when typing "/help", "/clear" etc.)
  const [showSlashCommands, setShowSlashCommands] = useState(false);
  const [selectedCommandIndex, setSelectedCommandIndex] = useState(0);

  // UI Interaction Mode 2: Command Palette (full modal triggered by "/" alone)
  const [showCommands, setShowCommands] = useState(false);


  // Input management and focus handling
  const [inputElement, setInputElement] = useState<HTMLTextAreaElement | null>(
    null
  );

  // Mode toggles and session management
  const [isPlanMode, setIsPlanMode] = useState(false);
  const pendingForkTextRef = useRef<{
    text: string;
    attachments: Attachment[];
    referenceMap: Map<string, string>;
  } | null>(null);

  // Component lifecycle refs
  const interruptedMessageAddedRef = useRef(false);
  const previousSessionIdRef = useRef<string>('');

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
  const sseStream = usePersistentSSE(session?.id || '');
  // const { apps: openApps } = useAppList();
  const forkSession = useForkSession();
  const createSession = useCreateSession();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const { data: preferences } = usePreferences();

  // Handle session changes: fork text loading and UI state clearing
  useEffect(() => {
    if (session?.id && session.id !== previousSessionIdRef.current) {
      // Handle pending fork first (deterministic order)
      if (pendingForkTextRef.current) {
        setText(pendingForkTextRef.current.text);
        useBoundStore
          .getState()
          .setHistoryState(
            pendingForkTextRef.current.attachments,
            pendingForkTextRef.current.referenceMap
          );
        pendingForkTextRef.current = null;
      }
      // Then handle session clearing (only if no fork was processed and not initial load)
      else if (previousSessionIdRef.current !== '') {
        setText('');
        clearAttachments();
        interruptedMessageAddedRef.current = false;
      }
      previousSessionIdRef.current = session.id;
    }
  }, [session?.id]);


  // Load messages when session messages data changes
  useEffect(() => {
    if (sessionMessages.data && sessionId) {
      setMessages(sessionMessages.data);
    } else if (sessionMessages.error) {
      // Show error message in chat
      setMessages([
        {
          content: `Failed to load messages: ${sessionMessages.error.message}`,
          from: 'assistant',
          frontend_only: true,
        },
      ]);
    } else if (!sessionMessages.isLoading) {
      // Clear messages only if not loading (avoid flash of empty state)
      setMessages([]);
    }
  }, [sessionMessages.data, sessionMessages.error, sessionMessages.isLoading, sessionId]);

  // // Transform open apps to Attachment format and filter allowed apps
  // const allowedApps = [
  //   'Notes',
  //   'Obsidian',
  //   'Blender',
  //   'Pixelmator Pro',
  //   'Final Cut Pro',
  // ];
  // const availableApps = useMemo(() => {
  //   return openApps
  //     .filter((app) =>
  //       allowedApps.some((allowed) =>
  //         app.name.toLowerCase().includes(allowed.toLowerCase())
  //       )
  //     )
  //     .map((app) => ({
  //       id: `app:${app.bundle_id}`,
  //       name: app.name,
  //       type: 'app' as const,
  //       icon: 'placeholder', // Icons loaded on-demand for performance
  //       isOpen: true,
  //       bundleId: app.bundle_id,
  //     }));
  // }, [openApps]);
  const availableApps: any[] = [];

  const fileRef = useFileReference(text, setText, session?.id);













  // Handle file upload success
  const handleFileUploadSuccess = (fileName: string) => {
    // Add file reference to text input (same behavior as "@" menu)
    const displayReference = `@${fileName}`;
    const newText = text ? `${text} ${displayReference} ` : `${displayReference} `;
    setText(newText);

    // Add reference mapping with full URL to ensure consistency with media array
    const fullUrl = buildSessionFileUrl(session!.id, fileName);
    useBoundStore.getState().addReference(displayReference, fullUrl);

    setMessages((prev) => [
      ...prev,
      {
        content: `File uploaded successfully: ${fileName}`,
        from: "assistant",
        frontend_only: true,
      },
    ]);
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

  useEffect(() => {
    const lastUserMessageIndex = messages.findLastIndex(
      (m) => m.from === 'user'
    );
    if (
      lastUserMessageIndex !== -1 &&
      userMessageRefs.current[lastUserMessageIndex]
    ) {
      setTimeout(() => {
        userMessageRefs.current[lastUserMessageIndex]?.scrollIntoView({
          behavior: 'smooth',
          block: 'start',
        });
      }, 100);
    }
  }, [messages, sseStream.processing]);

  const setUserMessageRef = (index: number) => (el: HTMLDivElement | null) => {
    userMessageRefs.current[index] = el;
  };

  // Handle paste events to detect video URLs
  const handlePaste = (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
    const pastedText = e.clipboardData.getData('text');

    // Only process if pasted content might contain URLs
    if (pastedText.includes('http') || pastedText.includes('youtu') || pastedText.includes('vimeo')) {
      const textarea = e.currentTarget;
      const selectionStart = textarea.selectionStart;
      const selectionEnd = textarea.selectionEnd;
      const currentText = text; // Use React state for reliability

      // Calculate what the text will be after paste operation
      const finalText = currentText.substring(0, selectionStart) +
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
      value.endsWith('/') &&
      value.length > 0 &&
      value[value.length - 1] === '/'
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
    if (e.key === 'Tab' && e.shiftKey) {
      e.preventDefault();
      setIsPlanMode((prev) => !prev);
      return;
    }

    // Handle Enter for form submission (without shift for new line)
    if (e.key === 'Enter' && !e.shiftKey) {
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
      () => setShowSlashCommands(false)
    );
    if (slashHandled) return;

    // Handle Escape key to stop processing or close popups
    if (e.key === 'Escape') {
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
      isInUIMode
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
  }, [
    sseStream.completed,
    sseStream.finalContent,
    sseStream.processing,
  ]);

  // Handle streaming errors - simple and clean
  useEffect(() => {
    if (sseStream.error && !sseStream.error.includes('cancelled') && !sseStream.cancelled) {
      setMessages((prev) => [
        ...prev,
        {
          content: `Failed to send prompt: ${sseStream.error}`,
          from: 'assistant',
          frontend_only: true,
        },
      ]);
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
    overridePlanMode?: boolean
  ) => {
    if (!(messageText && session?.id && sseStream.connected)) {
      return;
    }

    // Exit history mode if active
    historyNavigation.resetHistoryMode();

    // Reset interrupted message guard for new message
    interruptedMessageAddedRef.current = false;

    // Clear input immediately - optimistic UI update
    setText('');
    clearAttachments();

    try {
      // Use clean submitMessage method
      await sseStream.submitMessage({
        text: messageText,
        attachments,
        referenceMap,
        planMode: overridePlanMode !== undefined ? overridePlanMode : isPlanMode,
        onUserMessage: (userMessage) => {
          setMessages((prev) => [...prev, userMessage]);
        },
        onCancelledContentPersist: (cancelledMessage) => {
          setMessages((prev) => [...prev, cancelledMessage]);
        },
      });
    } catch (error) {
      // Restore input on error
      setText(messageText);
      // Note: attachments are already cleared, would need more complex state management to restore them
      console.error('Failed to submit message:', error);
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
      console.error('Failed to cancel message:', error);
    }
  };

  // Handle new session creation
  const handleNewSession = async () => {
    try {
      // Create a new session
      const newSession = await createSession.mutateAsync({
        title: 'New Session',
      });

      // Navigate to the new session - this will automatically trigger UI updates
      navigate({
        to: '/$sessionId',
        params: { sessionId: newSession.id },
        replace: true,
      });
    } catch (error) {
      console.error('Failed to create new session:', error);
    }
  };

  // Handle plan actions from ConversationDisplay
  const handlePlanAction = (action: 'proceed' | 'keep-planning') => {
    if (action === 'proceed') {
      setIsPlanMode(false);
      submitMessage(
        'Proceed with implementing the plan you just created. Begin implementation now.',
        false
      );
    }
    // For 'keep-planning', no additional action needed
  };

  // Handle forking conversation at a specific message
  const handleForkMessage = async (messageIndex: number) => {
    const messageToFork = messages[messageIndex];
    if (!messageToFork || messageToFork.from !== 'user' || !session?.id) {
      return;
    }

    try {
      // Call backend to fork session and copy messages
      const newSession = await forkSession.mutateAsync({
        sourceSessionId: session.id,
        messageIndex,
        title: `Forked: ${session.title || 'Chat Session'}`,
      });

      // Extract media paths and app names from the message attachments
      const mediaPaths =
        messageToFork.attachments?.filter((a) => a.path).map((a) => a.path!) ||
        [];
      const appNames =
        messageToFork.attachments
          ?.filter((a) => a.type === 'app')
          .map((a) => a.name) || [];

      // Reconstruct attachment state from the historical message
      const { contractedText, attachments, referenceMap } =
        await reconstructAttachmentsFromHistory(
          messageToFork.content,
          mediaPaths,
          appNames
        );

      // Queue fork text BEFORE navigation to prevent race condition
      pendingForkTextRef.current = { text: contractedText, attachments, referenceMap };

      // Navigate to the forked session
      navigate({
        to: '/$sessionId',
        params: { sessionId: newSession.id },
        replace: true,
      });
    } catch (error) {
      console.error('Failed to fork conversation:', error);
      // Show error feedback
      setFeedbackMessage(`Error: Failed to fork conversation`);

      // Auto-hide after 3 seconds
      setTimeout(() => {
        setFeedbackMessage(null);
      }, 3000);

      // Don't add error to chat - handled by notification
    }
  };

  // Button status and disabled state now computed by enhanced hook
  const isSubmitDisabled = sseStream.buttonStatus === 'ready'
    ? (!text && attachments.length === 0) ||
      !session?.id ||
      sessionLoading ||
      sseStream.isSubmitDisabled
    : sseStream.buttonStatus === 'paused'
      ? true // Disable button completely during cancellation
      : !session?.id || sessionLoading || sseStream.isSubmitDisabled;

  return (
    <div className="fl flex h-full w-full p-8">
      <div className="flex-1 overflow-y-auto">
        {/* Feedback message notification */}
        {feedbackMessage && (
          <div className={`fixed top-4 left-1/2 transform -translate-x-1/2 z-50 px-4 py-2 rounded-md shadow-md animate-in fade-in slide-in-from-top-5 duration-300 ${feedbackMessage.startsWith("Error:")
            ? "bg-destructive text-destructive-foreground"
            : "bg-primary text-primary-foreground"
            }`}>
            {feedbackMessage}
          </div>
        )}

        <div className="@container/main px mx-auto mt-4 flex max-w-4xl flex-1 flex-col gap-2 pb-24">
          {/* Loading indicator for messages */}
          {sessionMessages.isLoading && (
            <div className="flex items-center justify-center p-4 text-muted-foreground">
              <div className="flex items-center gap-2">
                <div className="h-4 w-4 animate-spin rounded-full border-2 border-muted border-t-foreground"></div>
                Loading messages...
              </div>
            </div>
          )}

          {/* Conversation Display */}
          <ConversationDisplay
            messages={messages}
            onForkMessage={handleForkMessage}
            onPlanAction={handlePlanAction}
            onUpdateMessage={(index, updatedMessage) => {
              setMessages(prev => [
                ...prev.slice(0, index),
                updatedMessage,
                ...prev.slice(index + 1)
              ]);
            }}
            sessionId={session?.id}
            setUserMessageRef={setUserMessageRef}
            sseStream={sseStream}
          />
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
              text={text}
              sessionId={session.id}
            />
          )}

          <AIInput
            className="border bg-stone-200/60 backdrop-blur-xl dark:bg-stone-700/60"
            onSubmit={handleSubmit}
          >
            <AIInputTextarea
              autoFocus
              availableApps={attachments
                .filter((a) => a.type === 'app')
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
              onPaste={handlePaste}
              value={text}
            />
            <AIInputToolbar>
              <AIInputTools>
                <div className="absolute bottom-1 left-1 flex">

                  {/* File Upload Button */}
                  {session?.id && (
                    <FileUploadButton
                      sessionId={session.id}
                      onUploadSuccess={handleFileUploadSuccess}
                      onUploadError={handleFileUploadError}
                      className="ml-1"
                    />
                  )}

                  {/* Mode selection temporarily hidden */}
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
                <div className="absolute bottom-1 right-14 text-xs text-muted-foreground">
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

          {/* Unified Command System */}
          {showCommands && (
            <CommandSlash
              onClose={() => {
                // Close the command palette UI
                setShowCommands(false);
                setShowSlashCommands(false);
              }}
              sessionId={sessionId}
              onFeedbackMessage={setFeedbackMessage}
              onNewSession={handleNewSession}
              onQueryClientInvalidate={(keys) => queryClient.invalidateQueries({ queryKey: keys })}
              onSubmitMessage={submitMessage}
              onAddMessage={(message) => setMessages((prev) => [...prev, message])}
            />
          )}

          {/* File Reference Dropdown with Command Component */}
          {fileRef.show && session?.id && (
            <CommandFileReference
              apps={availableApps}
              fileRef={fileRef}
              onClose={fileRef.close}
              onTextUpdate={setText}
              text={text}
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
