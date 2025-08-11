export type TokenType = 'text' | 'file-ref' | 'app-ref' | 'slash-command';

export interface Token {
  type: TokenType;
  content: string;
  start: number;
  end: number;
}

export class TextParser {
  constructor(
    private availableFiles: string[],
    private availableCommands: string[],
    private availableApps: string[] = []
  ) {}

  parse(text: string): Token[] {
    if (!text) return [{ type: 'text', content: '', start: 0, end: 0 }];

    const matches: Array<{
      type: TokenType;
      start: number;
      end: number;
      content: string;
    }> = [];

    // Create regex patterns for efficient matching
    this.availableFiles.forEach((filename) => {
      const patterns = [`@${filename}`, `@../${filename}`];
      patterns.forEach((pattern) => {
        const regex = new RegExp(this.escapeRegex(pattern), 'g');
        let match;
        while ((match = regex.exec(text)) !== null) {
          matches.push({
            type: 'file-ref',
            start: match.index,
            end: match.index + match[0].length,
            content: match[0],
          });
        }
      });
    });

    // Match app references
    this.availableApps.forEach((appName) => {
      const pattern = `@${appName}`;
      const regex = new RegExp(this.escapeRegex(pattern), 'g');
      let match;
      while ((match = regex.exec(text)) !== null) {
        matches.push({
          type: 'app-ref',
          start: match.index,
          end: match.index + match[0].length,
          content: match[0],
        });
      }
    });

    this.availableCommands.forEach((command) => {
      const pattern = `/${command}`;
      const regex = new RegExp(this.escapeRegex(pattern), 'g');
      let match;
      while ((match = regex.exec(text)) !== null) {
        matches.push({
          type: 'slash-command',
          start: match.index,
          end: match.index + match[0].length,
          content: match[0],
        });
      }
    });

    // Sort matches and build tokens
    matches.sort((a, b) => a.start - b.start);

    const tokens: Token[] = [];
    let currentIndex = 0;

    for (const match of matches) {
      if (currentIndex < match.start) {
        tokens.push({
          type: 'text',
          content: text.slice(currentIndex, match.start),
          start: currentIndex,
          end: match.start,
        });
      }
      tokens.push(match);
      currentIndex = match.end;
    }

    if (currentIndex < text.length) {
      tokens.push({
        type: 'text',
        content: text.slice(currentIndex),
        start: currentIndex,
        end: text.length,
      });
    }

    return tokens.length
      ? tokens
      : [{ type: 'text', content: text, start: 0, end: text.length }];
  }

  getTokenAt(position: number, tokens: Token[]): Token | null {
    return (
      tokens.find((token) => position >= token.start && position < token.end) ||
      null
    );
  }

  getTokenForManipulation(position: number, tokens: Token[]): Token | null {
    return (
      tokens.find(
        (token) => position >= token.start && position <= token.end
      ) || null
    );
  }

  handleDeletion(
    text: string,
    cursorPos: number
  ): { newText: string; newCursor: number } | null {
    const tokens = this.parse(text);
    const token = this.getTokenForManipulation(cursorPos, tokens);

    if (token && this.isSpecialToken(token.type)) {
      return {
        newText: text.slice(0, token.start) + text.slice(token.end),
        newCursor: token.start,
      };
    }
    return null;
  }

  handleArrowKey(
    key: 'ArrowLeft' | 'ArrowRight',
    cursorPos: number,
    tokens: Token[]
  ): number | null {
    const token = this.getTokenForManipulation(cursorPos, tokens);
    if (token && this.isSpecialToken(token.type)) {
      if (key === 'ArrowLeft') {
        return token.start;
      }
      // ArrowRight: only jump to end if not already there
      return cursorPos === token.end ? null : token.end;
    }
    return null;
  }


  private isSpecialToken(type: TokenType): boolean {
    return (
      type === 'file-ref' || type === 'app-ref' || type === 'slash-command'
    );
  }

  private escapeRegex(string: string): string {
    return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
  }
}

// Legacy exports for backward compatibility during transition
export function parseTextIntoTokens(
  text: string,
  options: { availableFiles: string[]; availableCommands: string[] }
): Token[] {
  const parser = new TextParser(
    options.availableFiles,
    options.availableCommands
  );
  return parser.parse(text);
}

export function findTokenAtPosition(
  tokens: Token[],
  position: number
): Token | null {
  return (
    tokens.find((token) => position >= token.start && position < token.end) ||
    null
  );
}


export function isWholeTokenDeletion(type: TokenType): boolean {
  return type === 'file-ref' || type === 'app-ref' || type === 'slash-command';
}
