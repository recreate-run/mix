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
import { type AIToolStatus } from '@/components/ui/kibo-ui/ai/tool';
import { FolderIcon } from 'lucide-react';
import { type FormEventHandler, useState, useEffect, useRef, useMemo } from 'react';
import { TooltipProvider } from '@/components/ui/tooltip';
import { useQueryClient } from '@tanstack/react-query';

import {  useActiveSession } from '@/hooks/useSession';
import { useForkSession } from '@/hooks/useForkSession';
import { useSessionMessages } from '@/hooks/useSessionMessages';
import { usePersistentSSE } from '@/hooks/usePersistentSSE';
import { useAppList } from '@/hooks/useOpenApps';
import { useFileReference } from '@/hooks/useFileReference';
import { CommandFileReference } from './command-file-reference';
import { useAttachmentStore, type Attachment, expandFileReferences, removeFileReferences, reconstructAttachmentsFromHistory } from '@/stores/attachmentStore';
import { useFolderSelection } from '@/hooks/useFolderSelection';
import { useMessageHistoryNavigation } from '@/hooks/useMessageHistoryNavigation';
import { useMessageScrolling } from '@/hooks/useMessageScrolling';
import { AttachmentPreview } from './attachment-preview';
import { CommandSlash, shouldShowSlashCommands, handleSlashCommandNavigation, slashCommands } from './command-slash';
import { ConversationDisplay } from './conversation-display';
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select';


type ToolCall = {
  name: string;
  description: string;
  status: AIToolStatus;
  parameters: Record<string, unknown>;
  result?: string;
  error?: string;
};

type Message = {
  content: string;
  from: 'user' | 'assistant';
  frontend_only?: boolean;
  toolCalls?: ToolCall[];
  attachments?: Attachment[];
  reasoning?: string;
  reasoningDuration?: number;
};

// Helper function to check if a message contains exit_plan_mode tool call
const hasExitPlanModeTool = (toolCalls: any[]) => {
  return toolCalls?.some(tc => tc.name === 'exit_plan_mode') || false;
};

const DEFAULT_WORKING_DIR = "/Users/sarathmenon/Desktop/a16z_demo/new_project";
const DEFAULT_ASSISTANT_MESSAGE = "Hello! I'm Mix, you AI agent for multimodal workflows. How can I help you today?";

const createDefaultMessage = (): Message => ({
  content: DEFAULT_ASSISTANT_MESSAGE,
  from: 'assistant',
  frontend_only: true
});

export function ChatApp() {
  // Core conversation state
  const [text, setText] = useState<string>('');
  const [messages, setMessages] = useState<Message[]>([
    createDefaultMessage()
  ]);
  
  // UI Interaction Mode 1: Slash Commands (dropdown when typing "/help", "/clear" etc.)
  const [showSlashCommands, setShowSlashCommands] = useState(false);
  const [selectedCommandIndex, setSelectedCommandIndex] = useState(0);
  
  // UI Interaction Mode 2: Command Palette (full modal triggered by "/" alone)
  const [showCommands, setShowCommands] = useState(false);
  
  // UI Interaction Mode 3: Plan Options (action buttons after exit_plan_mode)
  const [showPlanOptions, setShowPlanOptions] = useState<number | null>(null);
  
  // Input management and focus handling
  const [inputElement, setInputElement] = useState<HTMLTextAreaElement | null>(null);
  
  // Mode toggles and session management
  const [isPlanMode, setIsPlanMode] = useState(false);
  const [pendingForkText, setPendingForkText] = useState<{text: string, attachments: Attachment[], referenceMap: Map<string, string>} | null>(null);
  
  // Component lifecycle refs
  const interruptedMessageAddedRef = useRef(false);
  const previousSessionIdRef = useRef<string>('');
  
  // UI Mode 4: File Reference (managed in useFileReference hook)  
  // UI Mode 5: Normal Input (default when all others are false)

  // All attachment store hooks at top to avoid temporal dead zone
  const attachments = useAttachmentStore(state => state.attachments);
  const referenceMap = useAttachmentStore(state => state.referenceMap);
  const addAttachment = useAttachmentStore(state => state.addAttachment);
  const removeAttachment = useAttachmentStore(state => state.removeAttachment);
  const clearAttachments = useAttachmentStore(state => state.clearAttachments);
  const addReference = useAttachmentStore(state => state.addReference);
  const removeReference = useAttachmentStore(state => state.removeReference);
  const syncWithText = useAttachmentStore(state => state.syncWithText);

  const { selectedFolder, selectFolder } = useFolderSelection();
  const { data: session, isLoading: sessionLoading, error: sessionError, switchToSession } = useActiveSession(selectedFolder || DEFAULT_WORKING_DIR);
  const sessionMessages = useSessionMessages(session?.id || null);
  const sseStream = usePersistentSSE(session?.id || '');
  const { apps: openApps, refreshApps } = useAppList();
  const forkSession = useForkSession();
  
  // Clear UI state when session changes (new working directory selected)
  useEffect(() => {
    if (session?.id && session.id !== previousSessionIdRef.current) {
      // Only clear if we're switching from one session to another (not initial load)
      if (previousSessionIdRef.current !== '') {
        setText('');
        clearAttachments();
        setShowPlanOptions(null);
        interruptedMessageAddedRef.current = false;
      }
      previousSessionIdRef.current = session.id;
    }
  }, [session?.id]);

  // Load messages when session messages data changes
  useEffect(() => {
    if (sessionMessages.data && session?.id) {
      setMessages([createDefaultMessage(), ...sessionMessages.data]);
    } else {
      setMessages([createDefaultMessage()]);
    }
  }, [sessionMessages.data, session?.id]);

  // Set fork text after session switching completes
  useEffect(() => {
    if (pendingForkText && session?.id) {
      setText(pendingForkText.text);
      useAttachmentStore.getState().setHistoryState(pendingForkText.attachments, pendingForkText.referenceMap);
      setPendingForkText(null);
    }
  }, [pendingForkText, session?.id]);
  
  // Transform open apps to Attachment format and filter allowed apps
  const allowedApps = ['Notes', 'Obsidian', 'Blender', 'Pixelmator Pro', 'Final Cut Pro'];
  const availableApps = useMemo(() => {
    return openApps
      .filter(app => allowedApps.some(allowed => app.name.toLowerCase().includes(allowed.toLowerCase())))
      .map(app => ({
        id: `app:${app.bundle_id}`,
        name: app.name,
        type: 'app' as const,
        icon: 'placeholder', // Icons loaded on-demand for performance
        isOpen: true,
        bundleId: app.bundle_id
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

  const fileRef = useFileReference(text, setText, selectedFolder || DEFAULT_WORKING_DIR);


  const handleAppSelect = (app: Attachment) => {
    // Update text with app reference (similar to file selection)
    const words = text.split(' ');
    const displayReference = `@${app.name}`;
    words[words.length - 1] = `${displayReference} `;
    const newText = words.join(' ');
    
    // Add app to attachment store and create reference mapping
    addAttachment(app);
    addReference(displayReference, `app:${app.name}`);
    setText(newText);
    
  };


  
  // Initialize new hooks
  const historyNavigation = useMessageHistoryNavigation({
    text,
    setText,
    batchSize: 50,
  });

  const { conversationRef, setUserMessageRef } = useMessageScrolling(messages, sseStream.processing);

  const handleTextChange = (value: string) => {
    setText(value);
    
    // Reset cancelled state when user starts typing after cancellation
    if (sseStream.cancelled && value.length > 0) {
      sseStream.resetCancelledState();
    }
    
    // Sync media store with text changes (bidirectional sync)
    syncWithText(value);
    
    // Check if user just typed a slash to open Command-K menu
    if (value.endsWith('/') && value.length > 0 && value[value.length - 1] === '/') {
      // Remove the slash and open Command-K menu
      setText(value.slice(0, -1));
      setShowCommands(true);
      setShowSlashCommands(false);
      
      return;
    }
    
    // Show slash commands dropdown when conditions are met
    const shouldShowDropdown = shouldShowSlashCommands(value) && !showCommands && !fileRef.show;
    
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
  const handleCommand = (action: 'select' | 'execute' | 'close', data?: any) => {
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
        setShowPlanOptions(null);
        
        
        if (command === 'clear') {
          handleNewSession();
          return;
        }
        
        submitMessage(`/${command}`);
        break;
      }
      case 'close': {
        setShowSlashCommands(false);
        setShowCommands(false);
        setShowPlanOptions(null);
        
        break;
      }
    }
  };



  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Handle Shift+Tab for plan mode toggle
    if (e.key === 'Tab' && e.shiftKey) {
      e.preventDefault();
      setIsPlanMode(prev => !prev);
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
    const historyHandled = historyNavigation.handleHistoryNavigation(e, isInUIMode);
    if (historyHandled) {
      return;
    }
  };


  // Handle completion of streaming
  useEffect(() => {
    if (sseStream.completed && (sseStream.finalContent || sseStream.toolCalls.length > 0) && !sseStream.processing) {
      // Convert SSE tool calls to our Message format
      const convertedToolCalls: ToolCall[] = sseStream.toolCalls.map(tc => ({
        name: tc.name,
        description: tc.description,
        status: tc.status as AIToolStatus,
        parameters: tc.parameters,
        result: tc.result,
        error: tc.error,
      }));
      
      setMessages(prev => {
        const newMessages = [...prev, { 
          content: sseStream.finalContent!, 
          from: 'assistant',
          toolCalls: convertedToolCalls.length > 0 ? convertedToolCalls : undefined,
          reasoning: sseStream.reasoning,
          reasoningDuration: sseStream.reasoningDuration
        }];
        
        // Check if this message contains an exit_plan_mode tool and show options
        if (hasExitPlanModeTool(convertedToolCalls)) {
          setShowPlanOptions(newMessages.length - 1);
        }
        
        return newMessages;
      });
      
      
      // Reset interrupted message guard when processing completes
      interruptedMessageAddedRef.current = false;
    }
  }, [sseStream.completed, sseStream.finalContent, sseStream.processing, session?.id]);

  // Handle streaming errors
  useEffect(() => {
    if (sseStream.error) {
      const errorMessage = `Failed to send prompt: ${sseStream.error}`;
      setMessages(prev => [...prev, { 
        content: errorMessage, 
        from: 'assistant',
        frontend_only: true
      }]);
      
    }
  }, [sseStream.error, session?.id]);

  // Declarative focus management - refocus chat input when all popups are closed
  useEffect(() => {
    if (!showCommands && !fileRef.show && !showSlashCommands && inputElement) {
      inputElement.focus();
    }
  }, [showCommands, fileRef.show, showSlashCommands, inputElement]);

  // Handle pause state changes - simplified since pausing is not implemented
  // (Keeping this for compatibility but it won't trigger since isPaused will always be false)

  const submitMessage = async (messageText: string, overridePlanMode?: boolean) => {
    if (!messageText || !session?.id || !sseStream.connected) {
      return;
    }
    
    // Exit history mode if active
    historyNavigation.resetHistoryMode();
    
    // Add user message to conversation and clear input immediately
    setMessages(prev => [...prev, { 
      content: messageText, 
      from: 'user',
      attachments: attachments.length > 0 ? attachments : undefined
    }]);
    setText('');
    clearAttachments();
    setShowPlanOptions(null); // Clear any shown plan options
    
    // Reset interrupted message guard for new message
    interruptedMessageAddedRef.current = false;
    
    
    // Send message via persistent SSE
    try {
      // Expand file references from display format to full paths
      const expandedText = expandFileReferences(messageText, referenceMap);
      
      const messageData = {
        text: expandedText,
        media: attachments.filter(a => a.path).map(a => a.path),
        apps: attachments.filter(a => a.type === 'app').map(app => app.name),
        plan_mode: overridePlanMode !== undefined ? overridePlanMode : isPlanMode
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
      setMessages(prev => [...prev, {
        content: "Execution paused",
        from: 'assistant',
        frontend_only: true
      }]);
    } catch (error) {
      console.error('Failed to cancel message:', error);
    }
  };

  // Handle new session creation
  const handleNewSession = () => {
    setMessages([
      createDefaultMessage()
    ]);
    setText('');
    clearAttachments();
    interruptedMessageAddedRef.current = false;
    
    setShowPlanOptions(null);
  };

  // Handle plan option button clicks
  const handlePlanProceed = (messageIndex: number) => {
    setIsPlanMode(false);
    setShowPlanOptions(null);
    submitMessage("Proceed with implementing the plan you just created. Begin implementation now.", false);
  };

  const handlePlanKeepPlanning = (messageIndex: number) => {
    setShowPlanOptions(null);
  };

  // Handle forking conversation at a specific message
  const handleForkMessage = async (messageIndex: number) => {
    const messageToFork = messages[messageIndex];
    if (!messageToFork || messageToFork.from !== 'user' || !session?.id) {
      return;
    }

    // Prevent forking the first message (no history to copy)
    if (messageIndex <= 1) {
      console.log('Cannot fork the first message - no conversation history to copy');
      return;
    }

    try {
      // Call backend to fork session and copy messages
      const newSession = await forkSession.mutateAsync({
        sourceSessionId: session.id,
        messageIndex: messageIndex - 1, // Account for default message offset
        title: `Forked: ${session.title || 'Chat Session'}`
      });

      // Extract media paths and app names from the message attachments  
      const mediaPaths = messageToFork.attachments?.filter(a => a.path).map(a => a.path!) || [];
      const appNames = messageToFork.attachments?.filter(a => a.type === 'app').map(a => a.name) || [];
      
      // Reconstruct attachment state from the historical message
      const { contractedText, attachments, referenceMap } = await reconstructAttachmentsFromHistory(
        messageToFork.content,
        mediaPaths,
        appNames
      );
      
      // Switch to the forked session
      switchToSession(newSession);
      
      // Queue fork text to be set after session switching completes
      setPendingForkText({ text: contractedText, attachments, referenceMap });
      setShowPlanOptions(null);
      
      
    } catch (error) {
      console.error('Failed to fork conversation:', error);
      setMessages(prev => [...prev, {
        content: `Failed to fork conversation: ${error}`,
        from: 'assistant',
        frontend_only: true
      }]);
    }
  };

  // Calculate submit button status and disabled state
  const buttonStatus = sseStream.cancelling ? 'cancelling' :
                      sseStream.cancelled ? 'streaming' :
                      sseStream.processing ? 'streaming' : 
                      sseStream.error ? 'error' : 'ready';
  
  // Ready state: need text/attachments and connection. Other states: only need connection for pause/resume
  const isSubmitDisabled = buttonStatus === 'ready' 
    ? ((!text && attachments.length === 0) || !session?.id || sessionLoading || !sseStream.connected)
    : buttonStatus === 'cancelling'
    ? true // Disable button completely during cancellation
    : (!session?.id || sessionLoading || !sseStream.connected);

  return (
    <TooltipProvider>
      <div className="flex flex-col h-screen px-4 pb-4">
      {/* Header with Session ID and Folder Select Button */}
      <div className="flex justify-between items-center mb-2">
        <div className="text-xs font-mono text-stone-400 px-2 py-1 bg-stone-800/50 rounded">
          Session: {session?.id?.slice(0, 8) || 'Loading...'}
        </div>
        <button
          onClick={handleFolderSelect}
          className="flex items-center gap-2 text-sm font-medium text-stone-500 hover:text-stone-100 hover:bg-stone-700/50 rounded-lg p-2 transition-colors"
          title={selectedFolder ? `Current folder: ${selectedFolder}` : `Default folder: ${DEFAULT_WORKING_DIR}`}
        >
          <FolderIcon className={`size-5 ${selectedFolder ? 'text-blue-400' : ''}`} />
        </button>
      </div>
      
      {/* Conversation Display */}
      <ConversationDisplay
        messages={messages}
        sseStream={sseStream}
        showPlanOptions={showPlanOptions}
        conversationRef={conversationRef}
        setUserMessageRef={setUserMessageRef}
        onPlanProceed={handlePlanProceed}
        onPlanKeepPlanning={handlePlanKeepPlanning}
        onForkMessage={handleForkMessage}
      />


      {/* Attachment Preview Section */}
      <div className="max-w-4xl mx-auto w-full mb-0">
        <AttachmentPreview 
          attachments={attachments} 
          onRemoveItem={(index) => {
            const attachmentToRemove = attachments[index];
            if (attachmentToRemove) {
              const fullPath = attachmentToRemove.type === 'app' 
                ? `app:${attachmentToRemove.name}` 
                : attachmentToRemove.path!;
              const updatedText = removeFileReferences(text, referenceMap, fullPath);
              setText(updatedText);
              
              // Remove the reference from the map
              for (const [displayName, mappedPath] of referenceMap) {
                if (mappedPath === fullPath) {
                  removeReference(displayName);
                  break;
                }
              }
              
            }
            removeAttachment(index);
          }} 
        />
      </div>

      {/* AI Input Section */}
      <div className="max-w-4xl mx-auto w-full relative">
        <div className="relative">
          <AIInput onSubmit={handleSubmit} className='border-neutral-600 border-[0.5px]'>
            <AIInputTextarea
            onChange={(e) => {
              handleTextChange(e.target.value);
              if (!inputElement) {
                setInputElement(e.target);
              }
            }} 
            onKeyDown={handleKeyDown}
            value={text}
            availableFiles={fileRef.files.map(file => file.name)}
            availableApps={attachments.filter(a => a.type === 'app').map(app => app.name)}
            availableCommands={slashCommands.map(cmd => cmd.name)}
            autoFocus/>
          <AIInputToolbar>
            <AIInputTools>
            </AIInputTools>
            <AIInputSubmit 
              disabled={isSubmitDisabled}
              status={buttonStatus}
              onPauseClick={handleCancelClick}
            />
          </AIInputToolbar>
        </AIInput>
        
        {/* Mode Selector */}
        <div className="absolute bottom-1 left-1">
          <Select
          value={isPlanMode ? 'plan' : 'edit'} onValueChange={(value) => setIsPlanMode(value === 'plan')}>
            <SelectTrigger size="sm" className="text-muted-foreground border-none bg-transparent dark:bg-transparent hover:bg-transparent  hover:dark:bg-transparent focus:ring-0 focus:border-none">
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
            onExecuteCommand={(command) => handleCommand('execute', command)}
            onClose={() => handleCommand('close')}
          />
        )}

        {/* File Reference Dropdown with Command Component */}
        {fileRef.show && (
          <CommandFileReference
            files={fileRef.files}
            apps={availableApps}
            onSelect={fileRef.selectFile}
            onSelectApp={handleAppSelect}
            currentFolder={fileRef.currentFolder}
            isLoadingFolder={fileRef.isLoadingFolder}
            onGoBack={fileRef.goBack}
            onEnterFolder={fileRef.enterSelectedFolder}
            onClose={fileRef.close}
          />
        )}
      </div>
    </div>
    </TooltipProvider>
  );
};