import { useMutation, useQueryClient } from '@tanstack/react-query';
import { mix } from '@/lib/mix-sdk';
import { invalidateSessionCaches } from '@/lib/session-cache';

interface SendMessageParams {
  content: string;
  sessionId: string;
}

interface MessageResponse {
  response: string;
}

const sendMessage = async (
  params: SendMessageParams
): Promise<MessageResponse> => {
  const response = await mix.messages.send({
    id: params.sessionId,
    requestBody: {
      content: params.content
    }
  });

  const assistantResponse = response.assistantResponse || 'No response from server';
  return { response: assistantResponse };
};

export const useSendMessage = () => {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: sendMessage,
    onSuccess: (_, variables) => {
      invalidateSessionCaches(queryClient, variables.sessionId);
    },
  });
};
