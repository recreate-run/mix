import { AIResponse } from '@/components/ui/kibo-ui/ai/response';

interface SessionSummary {
  id: string;
  title: string;
  messageCount: number;
  totalTokens: number;
  cost: number;
  createdAt: number;
  updatedAt: number;
  parentSessionId?: string;
  isCurrent: boolean;
}

interface SessionsData {
  type: string;
  currentSession?: string;
  sessions: SessionSummary[];
}

interface SessionsDisplayProps {
  data: SessionsData;
}

export function SessionsDisplay({ data }: SessionsDisplayProps) {
  // Format tokens in K
  const formatTokens = (tokens: number) => {
    if (tokens >= 1000) {
      return `${(tokens / 1000).toFixed(1)}K`;
    }
    return tokens.toString();
  };

  const formatTimestamp = (timestamp: number) => {
    if (timestamp === 0) return '';
    const date = new Date(timestamp * 1000);
    return date.toLocaleDateString();
  };

  // Generate markdown string
  let markdown = '# Available Sessions\n\n';

  if (data.sessions.length === 0) {
    markdown += 'No sessions found.\n';
    return <AIResponse>{markdown}</AIResponse>;
  }

  // Sort sessions by updated date (most recent first)
  const sortedSessions = [...data.sessions].sort(
    (a, b) => b.updatedAt - a.updatedAt
  );

  sortedSessions.forEach((session) => {
    const currentIndicator = session.isCurrent ? ' **(current)**' : '';
    const tokensDisplay =
      session.totalTokens > 0 ? formatTokens(session.totalTokens) : '0';

    markdown += `## ${session.title}${currentIndicator}\n`;
    markdown += `- **ID:** ${session.id}\n`;
    markdown += `- **Messages:** ${session.messageCount}\n`;
    markdown += `- **Tokens:** ${tokensDisplay}\n`;
    markdown += `- **Cost:** $${session.cost.toFixed(4)}\n`;
    markdown += `- **Created:** ${formatTimestamp(session.createdAt)}\n`;

    if (session.updatedAt > 0) {
      markdown += `- **Last Updated:** ${formatTimestamp(session.updatedAt)}\n`;
    }

    if (session.parentSessionId) {
      markdown += `- **Parent Session:** ${session.parentSessionId}\n`;
    }

    markdown += '\n';
  });

  return <AIResponse>{markdown}</AIResponse>;
}
