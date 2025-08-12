import { useQueryClient } from '@tanstack/react-query';
import { FolderIcon } from 'lucide-react';
import {
  type FormEventHandler,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import {
  AIInput,
  AIInputButton,
  AIInputModelSelect,
  AIInputModelSelectContent,
  AIInputModelSelectItem,
  AIInputModelSelectTrigger,
  AIInputModelSelectValue,
  AIInputSubmit,
  AIInputTextarea,
  AIInputToolbar,
  AIInputTools,
} from '@/components/ui/kibo-ui/ai/input';
import type { AIToolStatus } from '@/components/ui/kibo-ui/ai/tool';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { TooltipProvider } from '@/components/ui/tooltip';
import { useFileReference } from '@/hooks/useFileReference';
import { useFolderSelection } from '@/hooks/useFolderSelection';
import { useForkSession } from '@/hooks/useForkSession';
import { useMessageHistoryNavigation } from '@/hooks/useMessageHistoryNavigation';
import { useMessageScrolling } from '@/hooks/useMessageScrolling';
import { useAppList } from '@/hooks/useOpenApps';
import { usePersistentSSE } from '@/hooks/usePersistentSSE';
import { useActiveSession, useCreateSession } from '@/hooks/useSession';
import { useSessionMessages } from '@/hooks/useSessionMessages';
import {
  type Attachment,
  expandFileReferences,
  reconstructAttachmentsFromHistory,
  useAttachmentStore,
} from '@/stores/attachmentStore';
import {
  handleSlashCommandNavigation,
  shouldShowSlashCommands,
  slashCommands,
} from '@/utils/slash-commands';
import { AttachmentPreview } from './attachment-preview';
import { CommandFileReference } from './command-file-reference';
import { CommandSlash } from './command-slash';
import { ConversationDisplay } from './conversation-display';

type ToolCall = {
  name: string;
  description: string;
  status: AIToolStatus;
  parameters: Record<string, unknown>;
  result?: string;
  error?: string;
};

export type MessageData = {
  text: string;
  media: string[];
  apps: string[];
  plan_mode: boolean;
};

type MediaOutput = {
  path: string;
  type: 'image' | 'video' | 'audio';
  title: string;
  description?: string;
  sourceVideo?: string; // Original video path for highlights
  startTime?: number; // Highlight start time in seconds
  duration?: number; // Highlight duration in seconds
};

type Message = {
  content: string;
  from: 'user' | 'assistant';
  frontend_only?: boolean;
  toolCalls?: ToolCall[];
  attachments?: Attachment[];
  reasoning?: string;
  reasoningDuration?: number;
  mediaOutputs?: MediaOutput[];
};


// Helper function to check if a message contains media_showcase tool call
const hasMediaShowcaseTool = (toolCalls: any[]) => {
  return toolCalls?.some((tc) => tc.name === 'media_showcase');
};

// Helper function to extract media outputs from media_showcase tool call
const getMediaShowcaseOutputs = (toolCalls: any[]): MediaOutput[] => {
  const mediaShowcaseTool = toolCalls?.find((tc) => tc.name === 'media_showcase');
  if (!mediaShowcaseTool?.parameters?.outputs) return [];
  
  try {
    return mediaShowcaseTool.parameters.outputs as MediaOutput[];
  } catch {
    return [];
  }
};


const DEFAULT_WORKING_DIR = '/Users/sarathmenon/Desktop/a16z_demo/new_project';


export function ChatApp() {
  // Core conversation state
  const [text, setText] = useState<string>('');
  const [messages, setMessages] = useState<Message[]>([]);

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
  const [pendingForkText, setPendingForkText] = useState<{
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
  const attachments = useAttachmentStore((state) => state.attachments);
  const referenceMap = useAttachmentStore((state) => state.referenceMap);
  const clearAttachments = useAttachmentStore(
    (state) => state.clearAttachments
  );
  const syncWithText = useAttachmentStore((state) => state.syncWithText);

  const { selectedFolder, selectFolder } = useFolderSelection();
  const {
    data: session,
    isLoading: sessionLoading,
    error: sessionError,
    switchToSession,
  } = useActiveSession(selectedFolder || DEFAULT_WORKING_DIR);
  const sessionMessages = useSessionMessages(session?.id || null);
  const sseStream = usePersistentSSE(session?.id || '');
  const { apps: openApps, refreshApps } = useAppList();
  const forkSession = useForkSession();
  const createSession = useCreateSession();

  // Clear UI state when session changes (new working directory selected)
  useEffect(() => {
    if (session?.id && session.id !== previousSessionIdRef.current) {
      // Only clear if we're switching from one session to another (not initial load)
      if (previousSessionIdRef.current !== '') {
        setText('');
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
      useAttachmentStore
        .getState()
        .setHistoryState(
          pendingForkText.attachments,
          pendingForkText.referenceMap
        );
      setPendingForkText(null);
    }
  }, [pendingForkText, session?.id]);

  // Transform open apps to Attachment format and filter allowed apps
  const allowedApps = [
    'Notes',
    'Obsidian',
    'Blender',
    'Pixelmator Pro',
    'Final Cut Pro',
  ];
  const availableApps = useMemo(() => {
    return openApps
      .filter((app) =>
        allowedApps.some((allowed) =>
          app.name.toLowerCase().includes(allowed.toLowerCase())
        )
      )
      .map((app) => ({
        id: `app:${app.bundle_id}`,
        name: app.name,
        type: 'app' as const,
        icon: 'placeholder', // Icons loaded on-demand for performance
        isOpen: true,
        bundleId: app.bundle_id,
      }));
  }, [openApps]);

  const handleFolderSelect = async () => {
    try {
      const selectedFolderPath = await selectFolder();
      if (selectedFolderPath) {
        console.log('Working directory selected:', selectedFolderPath);
      }
    } catch (error) {
      console.error('Failed to select working directory:', error);
    }
  };

  const fileRef = useFileReference(
    text,
    setText,
    selectedFolder || DEFAULT_WORKING_DIR
  );


  // Initialize new hooks
  const historyNavigation = useMessageHistoryNavigation({
    text,
    setText,
    batchSize: 50,
  });

  const { conversationRef, setUserMessageRef } = useMessageScrolling(
    messages,
    sseStream.processing
  );

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

  // Unified command handler
  const handleCommand = (
    action: 'select' | 'execute' | 'close',
    data?: any
  ) => {
    switch (action) {
      case 'select': {
        setShowSlashCommands(false);
        setSelectedCommandIndex(0);
        setText(text.slice(0, -1));
        setShowCommands(true);
        break;
      }
      case 'execute': {
        const command = data as string;
        setShowSlashCommands(false);
        setShowCommands(false);

        if (command === 'clear') {
          // Create a new session instead of just clearing UI
          handleNewSession();
          return;
        }

        submitMessage(`/${command}`);
        break;
      }
      case 'close': {
        setShowSlashCommands(false);
        setShowCommands(false);

        break;
      }
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
      (command) => handleCommand('select', command),
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
        handleCommand('close');
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
            from: 'assistant',
            toolCalls:
              convertedToolCalls.length > 0 ? convertedToolCalls : undefined,
            reasoning: sseStream.reasoning,
            reasoningDuration: sseStream.reasoningDuration,
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
          from: 'assistant',
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
    overridePlanMode?: boolean
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
        from: 'user',
        attachments: attachments.length > 0 ? attachments : undefined,
      },
    ]);
    setText('');
    clearAttachments();

    // Reset interrupted message guard for new message
    interruptedMessageAddedRef.current = false;

    // Send message via persistent SSE
    try {
      // Expand file references from display format to full paths
      const expandedText = expandFileReferences(messageText, referenceMap);

      const messageData: MessageData = {
        text: expandedText,
        media: attachments.filter((a) => a.path).map((a) => a.path),
        apps: attachments
          .filter((a) => a.type === 'app')
          .map((app) => app.name),
        plan_mode:
          overridePlanMode !== undefined ? overridePlanMode : isPlanMode,
      };
      await sseStream.sendMessage(JSON.stringify(messageData));
    } catch (error) {
      console.error('Failed to send message:', error);
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
          content: 'Execution paused',
          from: 'assistant',
          frontend_only: true,
        },
      ]);
    } catch (error) {
      console.error('Failed to cancel message:', error);
    }
  };

  // Handle new session creation
  const handleNewSession = async () => {
    try {
      // Create a new session with the current working directory
      const newSession = await createSession.mutateAsync({
        title: 'New Session',
        workingDirectory: selectedFolder || DEFAULT_WORKING_DIR,
      });
      
      // Switch to the new session - this will automatically trigger UI updates
      switchToSession(newSession);
      
      // Clear current UI state
      setText('');
      clearAttachments();
      interruptedMessageAddedRef.current = false;
    } catch (error) {
      console.error('Failed to create new session:', error);
      // Fallback to old behavior if session creation fails
      setMessages([]);
      setText('');
      clearAttachments();
      interruptedMessageAddedRef.current = false;
    }
  };

  // Handle plan actions from ConversationDisplay
  const handlePlanAction = (action: 'proceed' | 'keep-planning', messageIndex: number) => {
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
        messageIndex: messageIndex,
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

      // Switch to the forked session
      switchToSession(newSession);

      // Queue fork text to be set after session switching completes
      setPendingForkText({ text: contractedText, attachments, referenceMap });
    } catch (error) {
      console.error('Failed to fork conversation:', error);
      setMessages((prev) => [
        ...prev,
        {
          content: `Failed to fork conversation: ${error}`,
          from: 'assistant',
          frontend_only: true,
        },
      ]);
    }
  };

  // Calculate submit button status and disabled state
  const buttonStatus = sseStream.cancelling
    ? 'cancelling'
    : sseStream.cancelled
      ? 'streaming'
      : sseStream.processing
        ? 'streaming'
        : sseStream.error
          ? 'error'
          : 'ready';

  // Ready state: need text/attachments and connection. Other states: only need connection for pause/resume
  const isSubmitDisabled =
    buttonStatus === 'ready'
      ? (!text && attachments.length === 0) ||
        !session?.id ||
        sessionLoading ||
        !sseStream.connected
      : buttonStatus === 'cancelling'
        ? true // Disable button completely during cancellation
        : !session?.id || sessionLoading || !sseStream.connected;

  return (
    <TooltipProvider>
      <div className="flex h-screen flex-col px-4 pb-4">
        {/* Header with Session ID and Folder Select Button */}
        <div className="mb-2 flex items-center justify-between">
          <div className="rounded bg-stone-800/50 px-2 py-1 font-mono text-stone-400 text-xs">
            Session: {session?.id?.slice(0, 8) || 'Loading...'}
          </div>
          <button
            className="flex items-center gap-2 rounded-lg p-2 font-medium text-sm text-stone-500 transition-colors hover:bg-stone-700/50 hover:text-stone-100"
            onClick={handleFolderSelect}
            title={
              selectedFolder
                ? `Current folder: ${selectedFolder}`
                : `Default folder: ${DEFAULT_WORKING_DIR}`
            }
          >
            <FolderIcon
              className={`size-5 ${selectedFolder ? 'text-blue-400' : ''}`}
            />
          </button>
        </div>

        {/* Conversation Display */}
        <ConversationDisplay
          conversationRef={conversationRef}
          messages={messages}
          onForkMessage={handleForkMessage}
          onPlanAction={handlePlanAction}
          setUserMessageRef={setUserMessageRef}
          sseStream={sseStream}
        />

        {/* Attachment Preview Section */}
        <div className="z-20 mx-auto mb-0 w-full max-w-4xl">
          <AttachmentPreview
            attachments={attachments}
            text={text}
            referenceMap={referenceMap}
            onTextChange={setText}
          />
        </div>

        {/* AI Input Section */}
        <div className="relative z-10 mx-auto w-full max-w-4xl shadow-[0_-20px_80px_rgba(0,0,0,0.7)] before:pointer-events-none before:absolute before:top-[-60px] before:right-0 before:left-0 before:h-16  before:from-transparent before:to-black/50 before:content-['']">
          <div className="relative">
            <AIInput
              className="border-[0.5px] border-neutral-600"
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
                value={text}
              />
              <AIInputToolbar>
                <AIInputTools></AIInputTools>
                <AIInputSubmit
                  disabled={isSubmitDisabled}
                  onPauseClick={handleCancelClick}
                  status={buttonStatus}
                />
              </AIInputToolbar>
            </AIInput>

            {/* Mode Selector */}
            <div className="absolute bottom-1 left-1">
              <Select
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
              </Select>
            </div>
          </div>

          {/* Unified Command System */}
          {showCommands && (
            <CommandSlash
              onClose={() => handleCommand('close')}
              onExecuteCommand={(command) => handleCommand('execute', command)}
            />
          )}

          {/* File Reference Dropdown with Command Component */}
          {fileRef.show && (
            <CommandFileReference
              apps={availableApps}
              currentFolder={fileRef.currentFolder}
              files={fileRef.files}
              isLoadingFolder={fileRef.isLoadingFolder}
              text={text}
              onClose={fileRef.close}
              onEnterFolder={fileRef.enterSelectedFolder}
              onGoBack={fileRef.goBack}
              onSelect={fileRef.selectFile}
              onTextUpdate={setText}
            />
          )}
        </div>

      </div>
    </TooltipProvider>
  );
}
