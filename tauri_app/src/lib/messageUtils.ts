import { stat } from '@tauri-apps/plugin-fs';
import {
  type Attachment,
  createFileAttachment,
  createFolderAttachment,
} from '@/stores/attachmentSlice';
import type { ToolCall, ToolCallData } from '@/types/common';
import type { MediaOutput } from '@/types/media';
import type { BackendMessage, UIMessage } from '@/types/message';

interface ParsedContent {
  text: string;
  media: string[];
  apps: string[];
}

const extractContentData = (content: string): ParsedContent => {
  try {
    const parsed = JSON.parse(content);
    return {
      text: parsed.text || content,
      media: parsed.media || [],
      apps: parsed.apps || [],
    };
  } catch {
    return {
      text: content,
      media: [],
      apps: [],
    };
  }
};

const createAppAttachment = (appName: string): Attachment => {
  return {
    id: `app:${appName}`,
    name: appName,
    type: 'app',
    icon: 'placeholder',
    isOpen: true,
  };
};

const convertMediaToAttachments = async (
  mediaPaths: string[]
): Promise<Attachment[]> => {
  const attachments: Attachment[] = [];

  for (const mediaPath of mediaPaths) {
    try {
      let attachment: Attachment | null = null;

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
      name: tc.name,
      description: tc.name, // Use name as description since we don't have a separate description
      status: tc.finished ? 'completed' : 'pending',
      parameters,
      result: undefined, // Backend doesn't provide result for persisted tool calls
      error: undefined, // Backend doesn't provide error for persisted tool calls
    };
  });
};

export const convertBackendMessageToUI = async (
  backendMessage: BackendMessage
): Promise<UIMessage> => {
  const { text, media, apps } = extractContentData(backendMessage.content);

  // Convert media paths to attachments
  const mediaAttachments = await convertMediaToAttachments(media);

  // Convert app names to attachments
  const appAttachments = apps.map((appName) => createAppAttachment(appName));

  // Combine all attachments
  const attachments = [...mediaAttachments, ...appAttachments];

  // Convert tool calls if present
  const toolCalls = backendMessage.toolCalls
    ? convertToolCallsToUI(backendMessage.toolCalls)
    : undefined;

  // Extract media outputs from media_showcase tool calls
  const mediaOutputs = toolCalls?.find((tc) => tc.name === 'media_showcase')
    ?.parameters?.outputs as MediaOutput[] | undefined;

  return {
    content: text,
    from: backendMessage.role === 'user' ? 'user' : 'assistant',
    toolCalls: toolCalls && toolCalls.length > 0 ? toolCalls : undefined,
    attachments: attachments.length > 0 ? attachments : undefined,
    mediaOutputs:
      mediaOutputs && mediaOutputs.length > 0 ? mediaOutputs : undefined,
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
