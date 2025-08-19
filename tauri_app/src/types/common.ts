import type { AIToolStatus } from '@/components/ui/kibo-ui/ai/tool';

export type ToolCall = {
  name: string;
  description: string;
  status: AIToolStatus;
  parameters: Record<string, unknown>;
  result?: string;
  error?: string;
};

export interface Session {
  id: string;
  title: string;
  workingDirectory?: string;
}

export interface ToolCallData {
  id: string;
  name: string;
  input: string;
  type: string;
  finished: boolean;
}
