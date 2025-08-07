import { AIResponse } from '@/components/ui/kibo-ui/ai/response';

interface SessionData {
  type: string;
  id: string;
  title: string;
  messageCount: number;
  totalTokens: number;
  promptTokens: number;
  completionTokens: number;
  cost: number;
  createdAt: number;
  updatedAt: number;
  parentSessionId?: string;
}

interface SessionDisplayProps {
  data: SessionData;
}

export function SessionDisplay({ data }: SessionDisplayProps) {
  // Format tokens in K with input/output split
  const formatTokens = (tokens: number) => {
    if (tokens >= 1000) {
      return `${(tokens / 1000).toFixed(1)}K`;
    }
    return tokens.toString();
  };

  const formatTimestamp = (timestamp: number) => {
    if (timestamp === 0) return '';
    const date = new Date(timestamp * 1000);
    return date.toLocaleString();
  };

  // Generate markdown string
  let markdown = '## Current Session Information\n\n';
  markdown += `- **ID:** ${data.id}\n`;
  markdown += `- **Title:** ${data.title}\n`;
  markdown += `- **Messages:** ${data.messageCount}\n`;

  if (data.totalTokens > 0) {
    const totalK = formatTokens(data.totalTokens);
    const inputK = formatTokens(data.promptTokens);
    const outputK = formatTokens(data.completionTokens);
    markdown += `- **Tokens:** ${totalK} (${inputK} in / ${outputK} out)\n`;
  } else {
    markdown += '- **Tokens:** 0\n';
  }

  markdown += `- **Cost:** $${data.cost.toFixed(4)}\n`;

  if (data.createdAt > 0) {
    markdown += `- **Created:** ${formatTimestamp(data.createdAt)}\n`;
  }

  if (data.updatedAt > 0) {
    markdown += `- **Last Updated:** ${formatTimestamp(data.updatedAt)}\n`;
  }

  if (data.parentSessionId) {
    markdown += `- **Parent Session:** ${data.parentSessionId}\n`;
  }

  return <AIResponse>{markdown}</AIResponse>;
}
