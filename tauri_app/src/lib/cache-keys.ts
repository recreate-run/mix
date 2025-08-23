export const CACHE_KEYS = {
  sessions: ['sessions'] as const,
  session: (id: string) => ['sessions', 'session', id] as const,
  sessionMessages: (id: string) => ['sessions', 'messages', id] as const,
} as const;