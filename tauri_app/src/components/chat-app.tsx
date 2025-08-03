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
import { AIMessage, AIMessageContent } from '@/components/ui/kibo-ui/ai/message';
import {
  AIReasoning,
  AIReasoningContent,
  AIReasoningTrigger,
} from '@/components/ui/kibo-ui/ai/reasoning';
import {
  AITool,
  AIToolContent,
  AIToolHeader,
  AIToolParameters,
  AIToolResult,
  AIToolLadder,
  AIToolStep,
  type AIToolStatus,
} from '@/components/ui/kibo-ui/ai/tool';
import { FolderIcon } from 'lucide-react';
import { type FormEventHandler, useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { TooltipProvider } from '@/components/ui/tooltip';

import { useSession, useCreateSession } from '@/hooks/useSession';
import { useSendMessage } from '@/hooks/useMessages';
import { usePersistentSSE } from '@/hooks/usePersistentSSE';
import { type FileEntry } from '@/hooks/useFileSystem';
import { useOpenApps } from '@/hooks/useOpenApps';
import { useFileReference } from '@/hooks/useFileReference';
import { CommandFileReference } from './command-file-reference';
import { useAttachmentStore, type Attachment, expandFileReferences, removeFileReferences, createFileAttachment, createFolderAttachment } from '@/stores/attachmentStore';
import { useFolderSelection } from '@/hooks/useFolderSelection';
import { useMessageHistoryNavigation } from '@/hooks/useMessageHistoryNavigation';
import { useMessageScrolling } from '@/hooks/useMessageScrolling';
import { LoadingDots } from './loading-dots';
import { AttachmentPreview } from './attachment-preview';
import { CommandSlash, shouldShowSlashCommands, handleSlashCommandNavigation, slashCommands } from './command-slash';
import { ResponseRenderer } from './response-renderer';
import { MessageAttachmentDisplay } from './message-attachment-display';
import { TodoList } from './todo-list';
import { PlanDisplay } from './plan-display';
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
  toolCalls?: ToolCall[];
  attachments?: Attachment[];
  reasoning?: string;
  reasoningDuration?: number;
};

// Helper function to extract todos from todo_write tool calls (works with both ToolCall and SSE formats)
const extractTodosFromToolCalls = (toolCalls: any[]) => {
  return toolCalls
    .filter(tc => tc.name === 'todo_write')
    .map(tc => {
      try {
        const todos = tc.parameters?.todos;
        return Array.isArray(todos) ? todos : [];
      } catch {
        return [];
      }
    })
    .flat();
};

// Helper function to extract plan content from exit_plan_mode tool calls (works with both ToolCall and SSE formats)
const extractPlanFromToolCalls = (toolCalls: any[]) => {
  const planTool = toolCalls.find(tc => tc.name === 'exit_plan_mode');
  if (!planTool) return '';
  
  try {
    return planTool.parameters?.plan || '';
  } catch {
    return '';
  }
};

// Helper function to filter out special tools (todo_write, exit_plan_mode) from toolCalls
const filterNonSpecialTools = (toolCalls: any[]) => {
  return toolCalls.filter(tc => tc.name !== 'todo_write' && tc.name !== 'exit_plan_mode');
};


// Helper function to check if a message contains exit_plan_mode tool call
const hasExitPlanModeTool = (toolCalls: any[]) => {
  return toolCalls?.some(tc => tc.name === 'exit_plan_mode') || false;
};

const DEFAULT_WORKING_DIR = "/Users/sarathmenon/Desktop/a16z_demo/new_project";

export function ChatApp() {
  const [text, setText] = useState<string>('');
  const [messages, setMessages] = useState<Message[]>([]);
  const [showSlashCommands, setShowSlashCommands] = useState(false);
  const [selectedCommandIndex, setSelectedCommandIndex] = useState(0);
  const [inputElement, setInputElement] = useState<HTMLTextAreaElement | null>(null);
  const [showCommands, setShowCommands] = useState(false);
  const [isPlanMode, setIsPlanMode] = useState(false);
  const [showPlanOptions, setShowPlanOptions] = useState<number | null>(null);
  const interruptedMessageAddedRef = useRef(false);
  const previousSessionIdRef = useRef<string>('');

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
  const { data: session, isLoading: sessionLoading, error: sessionError } = useSession(selectedFolder || DEFAULT_WORKING_DIR);
  const sseStream = usePersistentSSE(session?.id || '');
  const { apps: openApps } = useOpenApps();
  
  // Clear UI state when session changes (new working directory selected)
  useEffect(() => {
    if (session?.id && session.id !== previousSessionIdRef.current) {
      // Only clear if we're switching from one session to another (not initial load)
      if (previousSessionIdRef.current !== '') {
        setMessages([]);
        setText('');
        clearAttachments();
        setShowPlanOptions(null);
        interruptedMessageAddedRef.current = false;
      }
      previousSessionIdRef.current = session.id;
    }
  }, [session?.id]);
  
  // Transform open apps to Attachment format and filter allowed apps
  const allowedApps = ['Notes', 'Obsidian', 'Blender', 'Pixelmator Pro', 'Final Cut Pro'];
  const availableApps = useMemo(() => {
    return openApps
      .filter(app => allowedApps.some(allowed => app.name.toLowerCase().includes(allowed.toLowerCase())))
      .map(app => ({
        id: `app:${app.name}`,
        name: app.name,
        type: 'app' as const,
        icon: app.icon_png_base64,
        isOpen: true
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
    sessionId: session?.id || '',
    text,
    setText,
    batchSize: 50,
  });

  const { conversationRef, setUserMessageRef } = useMessageScrolling(messages, sseStream.processing);

  const handleTextChange = (value: string) => {
    setText(value);
    
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
    
    // Handle slash commands using utility function (for other cases)
    const shouldShow = shouldShowSlashCommands(value);
    setShowSlashCommands(shouldShow);
    if (!shouldShow) {
      setShowCommands(false);
    }
  };

  const handleSlashCommandSelect = async (command: typeof slashCommands[0]) => {
    setShowSlashCommands(false);
    // Remove the slash from the text and open Command-K menu
    setText(text.slice(0, -1)); // Remove the trailing slash
    setShowCommands(true);
  };

  const handleCommandExecute = (command: string) => {
    setShowCommands(false);
    
    // Handle special commands directly
    if (command === 'clear') {
      handleNewSession();
      return;
    }
    
    submitMessage(`/${command}`);
  };

  const handleCommandClose = () => {
    setShowCommands(false);
  };



  const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
    // Handle Shift+Tab for plan mode toggle
    if (e.key === 'Tab' && e.shiftKey) {
      e.preventDefault();
      setIsPlanMode(prev => !prev);
      return;
    }

    // Handle Cmd+Enter for form submission (fallback)
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
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
      handleSlashCommandSelect,
      () => setShowSlashCommands(false)
    );
    if (slashHandled) return;

    // Handle Escape key to close file reference popup
    if (fileRef.show && e.key === 'Escape') {
      e.preventDefault();
      fileRef.close();
      return;
    }

    // Handle history navigation when not in other modes
    const historyHandled = historyNavigation.handleHistoryNavigation(
      e,
      showSlashCommands || fileRef.show
    );
    if (historyHandled) return;
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
  }, [sseStream.completed, sseStream.finalContent, sseStream.processing]);

  // Handle streaming errors
  useEffect(() => {
    if (sseStream.error) {
      const errorMessage = `Failed to send prompt: ${sseStream.error}`;
      setMessages(prev => [...prev, { content: errorMessage, from: 'assistant' }]);
    }
  }, [sseStream.error]);

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

  // Handle pause/resume button clicks
  const handlePauseResumeClick = async () => {
    console.log('pausing not implemented');
  };

  // Handle new session creation
  const handleNewSession = () => {
    setMessages([]);
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

  // Calculate submit button status and disabled state
  const buttonStatus = sseStream.processing ? 'streaming' : 
                      sseStream.error ? 'error' : 'ready';
  
  // Ready state: need text/attachments and connection. Other states: only need connection for pause/resume
  const isSubmitDisabled = buttonStatus === 'ready' 
    ? ((!text && attachments.length === 0) || !session?.id || sessionLoading || !sseStream.connected)
    : (!session?.id || sessionLoading || !sseStream.connected);

  return (
    <TooltipProvider>
      <div className="flex flex-col h-screen px-4 pb-4">
      {/* Header with Folder Select Button */}
      <div className="flex justify-end mb-2">
        <button
          onClick={handleFolderSelect}
          className="flex items-center gap-2 text-sm font-medium text-stone-500 hover:text-stone-100 hover:bg-stone-700/50 rounded-lg p-2 transition-colors"
          title={selectedFolder ? `Current folder: ${selectedFolder}` : `Default folder: ${DEFAULT_WORKING_DIR}`}
        >
          <FolderIcon className={`size-5 ${selectedFolder ? 'text-blue-400' : ''}`} />
        </button>
      </div>
      
      {/* Conversation Display */}
      <div ref={conversationRef} className="relative h-full flex-1 overflow-y-auto">
        <div className="">
          {messages.map((message, index) => (
            <AIMessage 
              from={message.from} 
              key={index}
              ref={message.from === 'user' ? setUserMessageRef(index) : undefined}
            >
              <AIMessageContent >
                {message.from === 'assistant' ? (
                  <>
                    {message.reasoning && (
                      <AIReasoning className="w-full mb-4" isStreaming={false} duration={message.reasoningDuration || undefined}>
                        <AIReasoningTrigger />
                        <AIReasoningContent>{message.reasoning}</AIReasoningContent>
                      </AIReasoning>
                    )}
                    <ResponseRenderer content={message.content} />
                  </>
                ) : (
                  <div>
                    <MessageAttachmentDisplay attachments={message.attachments || []} />
                    {message.content}
                  </div>
                )}
                {/* Render todos inline without tool wrapper */}
                {message.toolCalls && message.toolCalls.length > 0 && (
                  <>
                    {extractTodosFromToolCalls(message.toolCalls).length > 0 && (
                      <div className="mt-4">
                        <TodoList todos={extractTodosFromToolCalls(message.toolCalls)} />
                      </div>
                    )}
                    {/* Render plan content */}
                    {extractPlanFromToolCalls(message.toolCalls) && (
                      <PlanDisplay 
                        planContent={extractPlanFromToolCalls(message.toolCalls)}
                        showOptions={showPlanOptions === index}
                        onProceed={() => handlePlanProceed(index)}
                        onKeepPlanning={() => handlePlanKeepPlanning(index)}
                      />
                    )}
                    {/* Render non-special tools in ladder */}
                    {filterNonSpecialTools(message.toolCalls).length > 0 && (
                      <AIToolLadder className="mt-4">
                        {filterNonSpecialTools(message.toolCalls).map((toolCall, toolIndex) => (
                          <AIToolStep
                            key={`${index}-${toolCall.name}-${toolIndex}`}
                            status={toolCall.status}
                            stepNumber={toolIndex + 1}
                            isLast={toolIndex === filterNonSpecialTools(message.toolCalls).length - 1}
                          >
                            <AIToolHeader
                              description={toolCall.description}
                              name={toolCall.name}
                              status={toolCall.status}
                            />
                            <AIToolContent toolCall={toolCall} />
                          </AIToolStep>
                        ))}
                      </AIToolLadder>
                    )}
                  </>
                )}
              </AIMessageContent>
            </AIMessage>
          ))}
          {sseStream.processing && (
            <AIMessage 
              from="assistant"
            >
              <AIMessageContent>
                {/* Show reasoning during streaming if available */}
                {sseStream.reasoning && (
                  <AIReasoning className="w-full mb-4" isStreaming={true} duration={sseStream.reasoningDuration || undefined}>
                    <AIReasoningTrigger />
                    <AIReasoningContent>{sseStream.reasoning}</AIReasoningContent>
                  </AIReasoning>
                )}
                {sseStream.toolCalls.length > 0 ? (
                  <>
                    {/* Render streaming todos inline without tool wrapper */}
                    {extractTodosFromToolCalls(sseStream.toolCalls).length > 0 && (
                      <div className="mt-4">
                        <TodoList todos={extractTodosFromToolCalls(sseStream.toolCalls)} />
                      </div>
                    )}
                    {/* Render streaming plan content */}
                    {extractPlanFromToolCalls(sseStream.toolCalls) && (
                      <PlanDisplay 
                        planContent={extractPlanFromToolCalls(sseStream.toolCalls)}
                        showOptions={false}
                      />
                    )}
                    {/* Render streaming non-special tools in ladder */}
                    {filterNonSpecialTools(sseStream.toolCalls).length > 0 && (
                      <AIToolLadder >
                        {filterNonSpecialTools(sseStream.toolCalls).map((toolCall, toolIndex) => (
                          <AIToolStep
                            key={`streaming-${toolCall.id}-${toolIndex}`}
                            status={toolCall.status}
                            stepNumber={toolIndex + 1}
                            isLast={toolIndex === filterNonSpecialTools(sseStream.toolCalls).length - 1}
                          >
                            <AIToolHeader
                              description={toolCall.description}
                              name={toolCall.name}
                              status={toolCall.status}
                            />
                            <AIToolContent toolCall={toolCall} />
                          </AIToolStep>
                        ))}
                      </AIToolLadder>
                    )}
                    {!sseStream.completed && <LoadingDots />}
                  </>
                ) : (
                  <LoadingDots />
                )}
              </AIMessageContent>
            </AIMessage>
          )}
        </div>
      </div>


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
              onPauseClick={handlePauseResumeClick}
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
            onExecuteCommand={handleCommandExecute}
            onClose={handleCommandClose}
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