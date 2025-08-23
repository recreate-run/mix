import { AIResponse } from '@/components/ui/kibo-ui/ai/response';
import type { SessionData } from '@/types/common';
import { getTotalMessages, getExchangeCount } from '@/types/common';
import { formatTokens } from '@/lib/utils';

interface SessionsData {
  type: string;
  currentSession?: string;
  sessions: (SessionData & { isCurrent: boolean })[];
}

interface SessionsDisplayProps {
  data: SessionsData;
}

export function SessionsDisplay({ data }: SessionsDisplayProps) {
  const formatTimestamp = (timestamp: string) => {
    if (!timestamp) return '';
    const date = new Date(timestamp); // RFC3339 parses directly
    return date.toLocaleDateString();
  };

  // Generate markdown string
  let markdown = '# Available Sessions\n\n';

  if (data.sessions.length === 0) {
    markdown += 'No sessions found.\n';
    return <AIResponse>{markdown}</AIResponse>;
  }

  // Sort sessions by created date (most recent first)
  const sortedSessions = [...data.sessions].sort(
    (a, b) => new Date(b.createdAt).getTime() - new Date(a.createdAt).getTime()
  );

  sortedSessions.forEach((session) => {
    const currentIndicator = session.isCurrent ? ' **(current)**' : '';
    const totalTokens = session.promptTokens + session.completionTokens;
    const tokensDisplay =
      totalTokens > 0 ? formatTokens(totalTokens) : '0';

    markdown += `## ${session.title}${currentIndicator}\n`;
    markdown += `- **ID:** ${session.id}\n`;
    const totalMessages = getTotalMessages(session);
    const exchanges = getExchangeCount(session);
    
    if (session.toolCallCount === 0) {
      markdown += `- **Messages:** ${totalMessages}\n`;
    } else {
      markdown += `- **Messages:** ${exchanges} exchanges, ${session.toolCallCount} tools\n`;
    }
    markdown += `- **Tokens:** ${tokensDisplay}\n`;
    markdown += `- **Cost:** $${session.cost.toFixed(4)}\n`;
    
    if (session.createdAt) {
      markdown += `- **Created:** ${formatTimestamp(session.createdAt)}\n`;
    }

    markdown += '\n';
  });

  return <AIResponse>{markdown}</AIResponse>;
}
