import { useMutation } from '@tanstack/react-query';
import { rpcCall } from '@/lib/rpc';

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
  const result = await rpcCall<any>('messages.send', params);
  const assistantResponse = result?.response || 'No response from server';
  return { response: assistantResponse };
};

export const useSendMessage = () => {
  return useMutation({
    mutationFn: sendMessage,
  });
};
