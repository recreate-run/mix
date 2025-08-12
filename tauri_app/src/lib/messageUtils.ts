import { stat } from '@tauri-apps/plugin-fs';
import {
  type Attachment,
  createFileAttachment,
  createFolderAttachment,
} from '@/stores/attachmentSlice';

export interface BackendMessage {
  id: string;
  sessionId: string;
  role: string;
  content: string;
}

export interface UIMessage {
  content: string;
  from: 'user' | 'assistant';
  frontend_only?: boolean;
  toolCalls?: any[];
  attachments?: Attachment[];
  reasoning?: string;
  reasoningDuration?: number;
}

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
        if (fileStat.isDir) {
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

  return {
    content: text,
    from: backendMessage.role === 'user' ? 'user' : 'assistant',
    attachments: attachments.length > 0 ? attachments : undefined,
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
