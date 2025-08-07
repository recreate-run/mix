import { AIResponse } from '@/components/ui/kibo-ui/ai/response';
import { ContextDisplay } from './context-display';
import { HelpDisplay } from './help-display';
import { McpDisplay } from './mcp-display';
import { SessionDisplay } from './session-display';
import { SessionsDisplay } from './sessions-display';

interface ResponseRendererProps {
  content: string;
}

export function ResponseRenderer({ content }: ResponseRendererProps) {
  // Handle empty responses (e.g., from /clear command)
  if (!content || content.trim() === '') {
    return null;
  }

  // All slash commands return JSON, parse and route to appropriate component
  try {
    const parsedData = JSON.parse(content);

    // Check if it's a context response by looking for expected fields
    if (
      parsedData.model &&
      parsedData.components &&
      Array.isArray(parsedData.components)
    ) {
      return <ContextDisplay data={parsedData} />;
    }

    // Check if it's a help response by looking for type field
    if (
      parsedData.type === 'help' &&
      parsedData.commands &&
      Array.isArray(parsedData.commands)
    ) {
      return <HelpDisplay data={parsedData} />;
    }

    // Check if it's a session response by looking for type field
    if (parsedData.type === 'session' && parsedData.id) {
      return <SessionDisplay data={parsedData} />;
    }

    // Check if it's a sessions response by looking for type field
    if (
      parsedData.type === 'sessions' &&
      parsedData.sessions &&
      Array.isArray(parsedData.sessions)
    ) {
      return <SessionsDisplay data={parsedData} />;
    }

    // Check if it's an MCP response by looking for type field
    if (
      parsedData.type === 'mcp' &&
      parsedData.servers &&
      Array.isArray(parsedData.servers)
    ) {
      return <McpDisplay data={parsedData} />;
    }

    // If we reach here, it's an unknown JSON structure - log and render as text
    console.warn('Unknown JSON response structure:', parsedData);
    return <AIResponse>{content}</AIResponse>;
  } catch (error) {
    // If JSON parsing fails, it's likely regular chat content
    return <AIResponse>{content}</AIResponse>;
  }
}
