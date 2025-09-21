import { AIResponse } from '@/components/ui/kibo-ui/ai/response';
import type { SessionData } from '@/types/common';
import { getTotalMessages, getExchangeCount } from '@/types/common';
import { formatTokens } from '@/lib/utils';

interface SessionDisplayProps {
  data: SessionData;
}

export function SessionDisplay({ data }: SessionDisplayProps) {
  const formatTimestamp = (timestamp: string) => {
    if (!timestamp) return '';
    const date = new Date(timestamp); // RFC3339 parses directly
    return date.toLocaleString();
  };

  // Generate markdown string
  let markdown = '## Current Session Information\n\n';
  markdown += `- **ID:** ${data.id}\n`;
  markdown += `- **Title:** ${data.title}\n`;
  const totalMessages = getTotalMessages(data);
  const exchanges = getExchangeCount(data);
  
  if (data.toolCallCount === 0) {
    markdown += `- **Messages:** ${totalMessages}\n`;
  } else {
    markdown += `- **Messages:** ${exchanges} exchanges, ${data.toolCallCount} tools\n`;
  }

  const totalTokens = data.promptTokens + data.completionTokens;
  if (totalTokens > 0) {
    const totalK = formatTokens(totalTokens);
    const inputK = formatTokens(data.promptTokens);
    const outputK = formatTokens(data.completionTokens);
    markdown += `- **Tokens:** ${totalK} (${inputK} in / ${outputK} out)\n`;
  } else {
    markdown += '- **Tokens:** 0\n';
  }

  markdown += `- **Cost:** $${data.cost.toFixed(4)}\n`;

  if (data.createdAt) {
    markdown += `- **Created:** ${formatTimestamp(data.createdAt)}\n`;
  }

  return <AIResponse>{markdown}</AIResponse>;
}
