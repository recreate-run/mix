import { useNavigate } from '@tanstack/react-router';
import { useQueryClient } from '@tanstack/react-query';
import {
  type FormEventHandler,
  useEffect,
  useMemo,
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { useFileReference } from '@/hooks/useFileReference';
import { useForkSession } from '@/hooks/useForkSession';
import { useMessageHistoryNavigation } from '@/hooks/useMessageHistoryNavigation';
import { useAppList } from '@/hooks/useOpenApps';
import { usePersistentSSE } from '@/hooks/usePersistentSSE';
import { useActiveSession, useCreateSession } from '@/hooks/useSession';
import { useSessionMessages } from '@/hooks/useSessionMessages';
import { useBoundStore } from '@/stores';
import {
  type Attachment,
  reconstructAttachmentsFromHistory,
} from '@/stores/attachmentSlice';
import { expandFileReferences } from '@/utils/attachmentUtils';
import { invalidateMessageHistoryCache } from '@/lib/session-cache';
import type { ToolCall } from '@/types/common';
import type { MediaOutput } from '@/types/media';
import type { MessageData, UIMessage } from '@/types/message';
import type { HierarchicalModelData } from '@/types/provider';
import {
  handleSlashCommandNavigation,
  shouldShowSlashCommands,
  slashCommands,
} from '@/utils/slash-commands';
import { AttachmentPreview } from './attachment-preview';
import { CommandFileReference } from './command-file-reference';
import { CommandSlash } from './command-slash';
import { ConversationDisplay } from './conversation-display';
import { PermissionDialog } from './permission-dialog';
import { handleStatusCommand } from '@/handlers/status-command-handler';
import { handleLoginCommand } from '@/handlers/login-command-handler';
import { handleLogoutCommand, logoutProvider } from '@/handlers/logout-command-handler';
import { handleUnifiedModelCommand, updateProviderPreference, handleModelSelectionInHierarchy } from '@/handlers/unified-model-command-handler';

// Helper function to check if a message contains show_media tool call
const hasMediaShowcaseTool = (toolCalls: ToolCall[]) => {
  return toolCalls?.some((tc) => tc.name === 'show_media');
};

// Helper function to extract media outputs from show_media tool call
const getMediaShowcaseOutputs = (toolCalls: ToolCall[]): MediaOutput[] => {
  const mediaShowcaseTool = toolCalls?.find(
    (tc) => tc.name === 'show_media'
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
  const [text, setText] = useState<string>('');
  const [messages, setMessages] = useState<UIMessage[]>([]);

  // UI Interaction Mode 1: Slash Commands (dropdown when typing "/help", "/clear" etc.)
  const [showSlashCommands, setShowSlashCommands] = useState(false);
  const [selectedCommandIndex, setSelectedCommandIndex] = useState(0);

  // UI Interaction Mode 2: Command Palette (full modal triggered by "/" alone)
  const [showCommands, setShowCommands] = useState(false);
  
  // Hierarchical model data for CMDK
  const [hierarchicalModelData, setHierarchicalModelData] = useState<HierarchicalModelData | undefined>(undefined);
  
  // Logout provider data for CMDK
  const [logoutData, setLogoutData] = useState<{
    providers: {
      id: string;
      displayName: string;
      authenticated: boolean;
      authMethod?: 'api_key' | 'oauth';
      isPreferred?: boolean;
    }[];
  } | undefined>(undefined);

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
  const { apps: openApps } = useAppList();
  const forkSession = useForkSession();
  const createSession = useCreateSession();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

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

  const fileRef = useFileReference(text, setText, session?.id);

  // Handle the status command with our SDK implementation
  const handleStatusCommandSpecial = async () => {
    try {
      // Add user message to show that the command was executed
      setMessages((prev) => [
        ...prev,
        {
          content: "/status",
          from: "user",
        },
      ]);
      
      // Execute the enhanced status command handler with UI
      const statusResult = await handleStatusCommand();
      
      // Add response message returned by the handler
      setMessages((prev) => [
        ...prev,
        statusResult
      ]);
    } catch (error) {
      console.error('Status command failed:', error);
      setMessages((prev) => [
        ...prev,
        {
          content: `Failed to check authentication status: ${error}`,
          from: "assistant",
          frontend_only: true,
        },
      ]);
    }
  };
  
  // Handle the login command with our SDK implementation
  const handleLoginCommandSpecial = async () => {
    try {
      // Add user message to show that the command was executed
      setMessages((prev) => [
        ...prev,
        {
          content: "/login",
          from: "user",
        },
      ]);
      
      // Execute the login command handler
      const loginResult = await handleLoginCommand();
      
      // Add response message returned by the handler
      setMessages((prev) => [
        ...prev,
        loginResult
      ]);
    } catch (error) {
      console.error('Login command failed:', error);
      setMessages((prev) => [
        ...prev,
        {
          content: `Failed to start login flow: ${error}`,
          from: "assistant",
          frontend_only: true,
        },
      ]);
    }
  };
  
  // Handle the logout command with our SDK implementation
  const handleLogoutCommandSpecial = async () => {
    try {
      // Add user message to show that the command was executed
      setMessages((prev) => [
        ...prev,
        {
          content: "/logout",
          from: "user",
        },
      ]);
      
      // Execute the logout command handler to get data
      const logoutData = await handleLogoutCommand();
      
      // If there's no logoutData (no authenticated providers), show the error message
      if (!logoutData.logoutData) {
        setMessages((prev) => [
          ...prev,
          logoutData,
        ]);
        return;
      }
      
      // Set logout data and show commands (CMDK will detect the data and show logout view)
      setLogoutData({
        providers: logoutData.logoutData.providers
      });
      setShowCommands(true);
      
    } catch (error) {
      console.error('Logout command failed:', error);
      setMessages((prev) => [
        ...prev,
        {
          content: `Failed to start logout flow: ${error}`,
          from: "assistant",
          frontend_only: true,
        },
      ]);
    }
  };

  // Handle the unified model command with our SDK implementation
  const handleUnifiedModelCommandSpecial = async () => {
    try {
      // Execute the unified model command handler to get data
      const modelData = await handleUnifiedModelCommand();
      
      // If there's an error (no hierarchicalModel), show the error message
      if (!modelData.hierarchicalModel) {
        setMessages((prev) => [
          ...prev,
          {
            content: "/model",
            from: "user",
          },
          modelData,
        ]);
        return;
      }
      
      // Add user message to show that the command was executed
      setMessages((prev) => [
        ...prev,
        {
          content: "/model",
          from: "user",
        },
      ]);
      
      // Set hierarchical model data and show commands (CMDK will detect the data and show hierarchical view)
      setHierarchicalModelData({
        providers: modelData.hierarchicalModel.providers,
        currentProvider: modelData.hierarchicalModel.currentProvider,
        currentModel: modelData.hierarchicalModel.currentModel
      });
      setShowCommands(true);
      
    } catch (error) {
      console.error('Unified model command failed:', error);
      setMessages((prev) => [
        ...prev,
        {
          content: "/model",
          from: "user",
        },
        {
          content: `Failed to handle model selection: ${error}`,
          from: "assistant",
          frontend_only: true,
        },
      ]);
    }
  };

  // Handle provider selection in hierarchical view
  const handleProviderSelectionSpecial = async (providerId: string) => {
    try {
      // Just update the preferences, don't refresh the entire hierarchical data
      // The CMDK component will handle the state transition from providers to models
      await updateProviderPreference(providerId);
    } catch (error) {
      console.error('Provider selection failed:', error);
      // Close CMDK and show error message
      setShowCommands(false);
      setHierarchicalModelData(undefined);
      setMessages((prev) => [
        ...prev,
        {
          content: `Failed to update provider preference: ${error}`,
          from: "assistant",
          frontend_only: true,
        },
      ]);
    }
  };

  // Handle model selection in hierarchical view
  const handleModelSelectionSpecial = async (providerId: string, modelId: string) => {
    try {
      const result = await handleModelSelectionInHierarchy(providerId, modelId);
      
      // Close CMDK and clear hierarchical data
      setShowCommands(false);
      setHierarchicalModelData(undefined);
      
      // Add success message
      setMessages((prev) => [
        ...prev,
        result
      ]);
    } catch (error) {
      console.error('Model selection failed:', error);
      // Close CMDK and show error message
      setShowCommands(false);
      setHierarchicalModelData(undefined);
      setMessages((prev) => [
        ...prev,
        {
          content: `Failed to update model preference: ${error}`,
          from: "assistant",
          frontend_only: true,
        },
      ]);
    }
  };
  
  // Handle provider selection in logout view
  const handleLogoutProviderSelectionSpecial = async (providerId: string) => {
    try {
      // Call logoutProvider to log out from the selected provider
      const result = await logoutProvider(providerId);
      
      // Close CMDK and clear logout data
      setShowCommands(false);
      setLogoutData(undefined);
      
      // Add success message
      setMessages((prev) => [
        ...prev,
        result
      ]);
    } catch (error) {
      console.error('Logout failed:', error);
      // Close CMDK and show error message
      setShowCommands(false);
      setLogoutData(undefined);
      setMessages((prev) => [
        ...prev,
        {
          content: `Failed to log out: ${error}`,
          from: "assistant",
          frontend_only: true,
        },
      ]);
    }
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

        if (command === 'status') {
          // Handle status command using our SDK implementation
          handleStatusCommandSpecial();
          return;
        }
        
        if (command === 'login') {
          // Handle login command using our SDK implementation
          handleLoginCommandSpecial();
          return;
        }
        
        if (command === 'logout') {
          // Handle logout command using our SDK implementation
          handleLogoutCommandSpecial();
          return;
        }
        
        if (command === 'model') {
          // Handle unified model command using our SDK implementation
          handleUnifiedModelCommandSpecial();
          return;
        }
        
        submitMessage(`/${command}`);
        break;
      }
      case 'close': {
        setShowSlashCommands(false);
        setShowCommands(false);
        setHierarchicalModelData(undefined); // Clear hierarchical data when closing

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
      // SSE tool calls are already in ToolCall format
      const convertedToolCalls: ToolCall[] = sseStream.toolCalls;

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
            timeline: sseStream.timeline && sseStream.timeline.length > 0 ? sseStream.timeline : undefined,
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
        media: attachments.filter((a) => a.path).map((a) => a.path!),
        apps: attachments
          .filter((a) => a.type === 'app')
          .map((app) => app.name),
        plan_mode:
          overridePlanMode !== undefined ? overridePlanMode : isPlanMode,
      };
      await sseStream.sendMessage(JSON.stringify(messageData));
      
      // Invalidate message history cache to ensure fresh data on next navigation
      invalidateMessageHistoryCache(queryClient);
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
    ? 'paused'
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
      : buttonStatus === 'paused'
        ? true // Disable button completely during cancellation
        : !session?.id || sessionLoading || !sseStream.connected;

  return (
    <div className="fl flex h-full w-full p-8">
      <div className="flex-1 overflow-y-auto">
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
              onClose={() => {
                handleCommand('close');
                setHierarchicalModelData(undefined); // Clear hierarchical data when closing
                setLogoutData(undefined); // Clear logout data when closing
              }}
              onExecuteCommand={(command) => handleCommand('execute', command)}
              sessionId={sessionId}
              hierarchicalModelData={hierarchicalModelData}
              logoutData={logoutData}
              onProviderSelect={handleProviderSelectionSpecial}
              onModelSelect={handleModelSelectionSpecial}
              onLogoutProviderSelect={handleLogoutProviderSelectionSpecial}
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
