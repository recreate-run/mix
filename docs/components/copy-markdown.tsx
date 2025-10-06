'use client';

import { Copy, Check } from 'lucide-react';
import { useState } from 'react';

interface CopyMarkdownProps {
  filePath: string;
}

export function CopyMarkdown({ filePath }: CopyMarkdownProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = async () => {
    try {
      const response = await fetch(`/api/markdown/${filePath}`);
      if (!response.ok) {
        const error = await response.text();
        console.error('Failed to fetch markdown:', error);
        throw new Error('Failed to fetch markdown');
      }

      const markdown = await response.text();
      await navigator.clipboard.writeText(markdown);

      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch (error) {
      console.error('Failed to copy markdown:', error, 'Path:', filePath);
    }
  };

  return (
    <button
      onClick={handleCopy}
      className="inline-flex items-center gap-2 px-3 py-1.5 text-sm font-medium text-fd-muted-foreground hover:text-fd-foreground border border-fd-border rounded-lg hover:bg-fd-accent/50 transition-colors"
      aria-label="Copy page markdown"
    >
      {copied ? (
        <>
          <Check className="h-4 w-4" />
          Copied!
        </>
      ) : (
        <>
          <Copy className="h-4 w-4" />
          Copy Page
        </>
      )}
    </button>
  );
}
