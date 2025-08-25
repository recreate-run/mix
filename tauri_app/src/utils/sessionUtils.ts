import type { SessionData } from '@/types/common';
import { TITLE_TRUNCATE_LENGTH } from '@/hooks/useSessionsList';

/**
 * Helper function to get display title for a session.
 * Extracts text from the first user message or falls back to session title.
 * 
 * @param session - The session data object
 * @returns A truncated display title string
 */
export const getDisplayTitle = (session: SessionData): string => {
  if (!session.firstUserMessage || session.firstUserMessage.trim() === '') {
    // New sessions won't have a first user message yet - use session title as fallback
    return session.title;
  }

  // Try to parse JSON and extract text from the parts structure
  let displayText = session.firstUserMessage;
  try {
    const parsed = JSON.parse(session.firstUserMessage);
    // Find the first text part in the parts array
    const textPart = parsed.find((part: any) => part.type === 'text');
    if (textPart?.data?.text) {
      displayText = textPart.data.text;
    }
  } catch {
    // If parsing fails, use the raw message as fallback (likely plain text message)
    displayText = session.firstUserMessage;
  }

  const truncated =
    displayText.length > TITLE_TRUNCATE_LENGTH
      ? `${displayText.substring(0, TITLE_TRUNCATE_LENGTH)}...`
      : displayText;

  return truncated;
};