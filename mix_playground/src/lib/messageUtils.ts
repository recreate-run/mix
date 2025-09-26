import { stat } from '@tauri-apps/plugin-fs';
import { type Attachment } from '@/stores/attachmentSlice';
import {
  createFileAttachment,
  createFolderAttachment,
} from '@/utils/attachmentUtils';
import type { ToolCall, ToolCallData } from '@/types/common';
import type { MediaOutput } from '@/types/media';
import type { UIMessage, TimelineEntry } from '@/types/message';
import type { BackendMessage } from 'mix-typescript-sdk/models';

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
    const mediaUrlRegex = /https?:\/\/[^\s]+\/api\/sessions\/[^\s]+\/files\/[^\s]+/g;
    const mediaUrls = content.match(mediaUrlRegex) || [];

    // Remove media URLs from text to get clean text
    let cleanText = content;
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
  mediaPaths: string[]
): Promise<Attachment[]> => {
  const attachments: Attachment[] = [];

  for (const mediaPath of mediaPaths) {
    try {
      let attachment: Attachment | null = null;

      // Check if this is a server URL (from reloaded session) - can be full URL or relative path
      if ((mediaPath.startsWith('http') || mediaPath.startsWith('/api/sessions/')) && mediaPath.includes('/api/sessions/') && mediaPath.includes('/files/')) {
        // Extract filename from server URL
        const urlParts = mediaPath.split('/');
        const filename = decodeURIComponent(urlParts[urlParts.length - 1]);

        // Create attachment with just the filename as path
        attachment = createFileAttachment(filename);
      } else {
        // Handle local file paths (during upload)
        try {
          const fileStat = await stat(mediaPath);
          if (fileStat.isDirectory) {
            attachment = await createFolderAttachment(mediaPath);
          } else {
            attachment = createFileAttachment(mediaPath);
          }
        } catch (statError) {
          // If stat fails, try to create as file based on file extension
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
      parameters = JSON.parse(tc.input || '{}');
    } catch {
      // If input is not valid JSON, treat as empty parameters
      parameters = {};
    }

    return {
      id: tc.id,
      name: tc.name,
      description: tc.name, // Use name as description since we don't have a separate description
      status: tc.finished ? 'completed' : 'pending',
      parameters,
      result: tc.result || undefined,
      error: tc.isError ? tc.result : undefined,
    };
  });
};

export const convertBackendMessageToUI = async (
  backendMessage: BackendMessage
): Promise<UIMessage> => {
  const { text, media } = extractContentData(backendMessage.userInput);

  // Convert media paths to attachments
  const attachments = await convertMediaToAttachments(media);

  // Convert tool calls if present
  const toolCalls = backendMessage.toolCalls
    ? convertToolCallsToUI(backendMessage.toolCalls)
    : undefined;

  // Extract media outputs from show_media tool calls
  const mediaOutputs = toolCalls?.find((tc) => tc.name === 'show_media')
    ?.parameters?.outputs as MediaOutput[] | undefined;

  // Create timeline from stored reasoning if available
  let timeline: TimelineEntry[] | undefined;
  if (backendMessage.reasoning && backendMessage.reasoning.trim()) {
    timeline = [{
      type: 'thinking',
      timestamp: Date.now(), // Could be derived from message timestamp
      content: backendMessage.reasoning,
      id: `stored-reasoning-${backendMessage.id}`
    }];
  }

  return {
    content: text,
    from: backendMessage.role === 'user' ? 'user' : 'assistant',
    toolCalls: toolCalls && toolCalls.length > 0 ? toolCalls : undefined,
    attachments: attachments.length > 0 ? attachments : undefined,
    timeline,
    mediaOutputs:
      mediaOutputs && mediaOutputs.length > 0 ? mediaOutputs : undefined,
    reasoning: backendMessage.reasoning,
    reasoningDuration: backendMessage.reasoningDuration,
  };
};

export const convertBackendMessagesToUI = async (
  backendMessages: BackendMessage[]
): Promise<UIMessage[]> => {
  const uiMessages: UIMessage[] = [];

  for (const backendMessage of backendMessages) {
    const uiMessage = await convertBackendMessageToUI(backendMessage);
    uiMessages.push(uiMessage);
  }

  return uiMessages;
};
