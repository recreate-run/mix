import { useMutation, useQueryClient } from '@tanstack/react-query';
import { mix } from '@/lib/mix-sdk';
import { invalidateSessionCaches } from '@/lib/session-cache';

interface SendMessageParams {
  content: string;
  sessionId: string;
  apps?: string[];
  media?: string[];
  planMode?: boolean;
}

interface MessageResponse {
  response: string;
}

async function sendMessage(
  params: SendMessageParams
): Promise<MessageResponse> {
  const response = await mix.messages.send({
    id: params.sessionId,
    requestBody: {
      text: params.content,
      apps: params.apps || [],
      media: params.media || [],
      planMode: params.planMode || false
    }
  });

  // The response is now a BackendMessage object containing the message data
  // Return the user input or assistant response as confirmation
  return { response: response.userInput || response.assistantResponse || 'Message sent successfully' };
}

export function useSendMessage() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: sendMessage,
    onSuccess: (_, variables) => {
      invalidateSessionCaches(queryClient, variables.sessionId);
    },
  });
}
