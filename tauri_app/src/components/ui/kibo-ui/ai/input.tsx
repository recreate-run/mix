'use client';

import { ArrowUpIcon, Loader2Icon, SquareIcon, XIcon, PlayIcon } from 'lucide-react';
import type {
  ComponentProps,
  HTMLAttributes,
  KeyboardEventHandler,
} from 'react';
import { Children, useCallback, useEffect, useRef, useState, useMemo } from 'react';
import { Button } from '@/components/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import { Textarea } from '@/components/ui/textarea';
import { cn } from '@/lib/utils';
import { TextParser, Token, TokenType } from '@/lib/textParser';

type UseAutoResizeTextareaProps = {
  minHeight: number;
  maxHeight?: number;
};

const useAutoResizeTextarea = ({
  minHeight,
  maxHeight,
}: UseAutoResizeTextareaProps) => {
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const adjustHeight = useCallback(
    (reset?: boolean) => {
      const textarea = textareaRef.current;
      if (!textarea) {
        return;
      }

      if (reset) {
        textarea.style.height = `${minHeight}px`;
        return;
      }

      // Temporarily shrink to get the right scrollHeight
      textarea.style.height = `${minHeight}px`;

      // Calculate new height
      const newHeight = Math.max(
        minHeight,
        Math.min(textarea.scrollHeight, maxHeight ?? Number.POSITIVE_INFINITY)
      );

      textarea.style.height = `${newHeight}px`;
    },
    [minHeight, maxHeight]
  );

  useEffect(() => {
    // Set initial height
    const textarea = textareaRef.current;
    if (textarea) {
      textarea.style.height = `${minHeight}px`;
    }
  }, [minHeight]);

  // Adjust height on window resize
  useEffect(() => {
    const handleResize = () => adjustHeight();
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, [adjustHeight]);

  return { textareaRef, adjustHeight };
};

export type AIInputProps = HTMLAttributes<HTMLFormElement>;

export const AIInput = ({ className, ...props }: AIInputProps) => (
  <form
    className={cn(
      'w-full overflow-hidden rounded-xl border bg-stone-900 shadow-sm',
      className
    )}
    {...props}
  />
);

export type AIInputTextareaProps = ComponentProps<typeof Textarea> & {
  minHeight?: number;
  maxHeight?: number;
  availableFiles?: string[];
  availableApps?: string[];
  availableCommands?: string[];
};

export const AIInputTextarea = ({
  onChange,
  onKeyDown,
  className,
  placeholder = 'What would you like to know?',
  minHeight = 48,
  maxHeight = 164,
  value = '',
  availableFiles = [],
  availableApps = [],
  availableCommands = [],
  ...props
}: AIInputTextareaProps) => {
  const { textareaRef, adjustHeight } = useAutoResizeTextarea({
    minHeight,
    maxHeight,
  });

  const overlayRef = useRef<HTMLDivElement>(null);
  const previousValueRef = useRef<string>(value || '');
  
  const parser = useMemo(() => new TextParser(availableFiles, availableCommands, availableApps), [availableFiles, availableCommands, availableApps]);
  const tokens = useMemo(() => parser.parse(value || ''), [parser, value]);
  
  // Update ref when value changes from outside
  useEffect(() => {
    previousValueRef.current = value || '';
  }, [value]);

  const syncScroll = () => {
    if (textareaRef.current && overlayRef.current) {
      overlayRef.current.scrollTop = textareaRef.current.scrollTop;
      overlayRef.current.scrollLeft = textareaRef.current.scrollLeft;
    }
  };

  const handleKeyDown: KeyboardEventHandler<HTMLTextAreaElement> = (e) => {
    if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
      e.preventDefault();
      e.currentTarget.form?.requestSubmit();
      return;
    }

    const textarea = textareaRef.current;
    if (textarea && (e.key === 'ArrowLeft' || e.key === 'ArrowRight') && textarea.selectionStart === textarea.selectionEnd) {
      const newCursor = parser.handleArrowKey(e.key, textarea.selectionStart, tokens);
      if (newCursor !== null) {
        e.preventDefault();
        textarea.setSelectionRange(newCursor, newCursor);
        return;
      }
    }

    onKeyDown?.(e);
  };

  const renderTokenOverlay = () => {
    const hasSpecialTokens = tokens.some(token => token.type !== 'text');
    if (!hasSpecialTokens) return null;

    return (
      <div
        ref={overlayRef}
        className="absolute inset-0 pointer-events-none overflow-hidden whitespace-pre-wrap break-words z-0"
        style={{
          font: 'inherit',
          lineHeight: 'inherit',
          padding: '12px',
          color: 'transparent',
        }}
      >
        {tokens.map((token, index) => (
          <span
            key={`${token.start}-${token.end}-${index}`}
            className={token.type !== 'text' ? parser.getTokenStyle(token.type) : ''}
            style={{ color: 'transparent' }}
          >
            {token.content}
          </span>
        ))}
      </div>
    );
  };

  return (
    <div className="relative">
      {renderTokenOverlay()}
      <Textarea
        className={cn(
          'w-full resize-none rounded-none border-none p-3 shadow-none outline-none ring-0',
          'bg-transparent dark:bg-transparent relative z-10',
          'focus-visible:ring-0 text-foreground',
          className
        )}
        name="message"
        onChange={(e) => {
          adjustHeight();
          
          const newValue = e.target.value;
          const previousValue = previousValueRef.current;
          const textarea = textareaRef.current;
          
          // Only check for token deletion if text got shorter (actual deletion)
          if (textarea && newValue.length < previousValue.length) {
            const deletion = parser.handleDeletion(newValue, textarea.selectionStart);
            if (deletion) {
              // Update ref with the cleaned value first
              previousValueRef.current = deletion.newText;
              
              // Set cursor position immediately
              textarea.setSelectionRange(deletion.newCursor, deletion.newCursor);
              
              // Create cleaned event and call onChange
              const cleanedEvent = {
                ...e,
                target: { ...e.target, value: deletion.newText },
                currentTarget: { ...e.currentTarget, value: deletion.newText }
              } as React.ChangeEvent<HTMLTextAreaElement>;
              
              onChange?.(cleanedEvent);
              return;
            }
          }
          
          // Update ref with new value for normal changes
          previousValueRef.current = newValue;
          onChange?.(e);
        }}
        onKeyDown={handleKeyDown}
        onScroll={syncScroll}
        placeholder={placeholder}
        ref={textareaRef}
        value={value}
        {...props}
      />
    </div>
  );
};

export type AIInputToolbarProps = HTMLAttributes<HTMLDivElement>;

export const AIInputToolbar = ({
  className,
  ...props
}: AIInputToolbarProps) => (
  <div
    className={cn('flex items-center justify-between p-1', className)}
    {...props}
  />
);

export type AIInputToolsProps = HTMLAttributes<HTMLDivElement>;

export const AIInputTools = ({ className, ...props }: AIInputToolsProps) => (
  <div
    className={cn(
      'flex items-center gap-1',
      '[&_button:first-child]:rounded-bl-xl',
      className
    )}
    {...props}
  />
);

export type AIInputButtonProps = ComponentProps<typeof Button>;

export const AIInputButton = ({
  variant = 'ghost',
  className,
  size,
  ...props
}: AIInputButtonProps) => {
  const newSize =
    (size ?? Children.count(props.children) > 1) ? 'default' : 'icon';

  return (
    <Button
      className={cn(
        'shrink-0 gap-1.5 rounded-lg',
        variant === 'ghost' && 'text-muted-foreground',
        newSize === 'default' && 'px-3',
        className
      )}
      size={newSize}
      type="button"
      variant={variant}
      {...props}
    />
  );
};

export type AIInputSubmitProps = ComponentProps<typeof Button> & {
  status?: 'submitted' | 'streaming' | 'ready' | 'error' | 'paused';
  onPauseClick?: () => void;
};

export const AIInputSubmit = ({
  className,
  variant = 'default',
  size = 'icon',
  status,
  children,
  onPauseClick,
  ...props
}: AIInputSubmitProps) => {
  let Icon = <ArrowUpIcon className='size-6' />;
  let buttonType: "submit" | "button" = "submit";
  let onClick = props.onClick;

  if (status === 'submitted') {
    Icon = <Loader2Icon className="animate-spin" />;
    buttonType = "button"; // Prevent form submission
    onClick = undefined; // Disable click handling
  } else if (status === 'streaming') {
    Icon = <SquareIcon />;
    buttonType = "button"; // Don't submit when streaming
    onClick = onPauseClick; // Use pause click handler
  } else if (status === 'paused') {
    Icon = <PlayIcon className='size-5' />;
    buttonType = "button"; // Don't submit when paused
    onClick = onPauseClick; // Use resume click handler
  } else if (status === 'error') {
    Icon = <XIcon />;
  }

  return (
    <Button
      className={cn('gap-1.5 rounded-full ', className)}
      size={size}
      type={buttonType}
      variant={variant}
      onClick={onClick}
      {...props}
    >
      {children ?? Icon}
    </Button>
  );
};

export type AIInputModelSelectProps = ComponentProps<typeof Select>;

export const AIInputModelSelect = (props: AIInputModelSelectProps) => (
  <Select {...props} />
);

export type AIInputModelSelectTriggerProps = ComponentProps<
  typeof SelectTrigger
>;

export const AIInputModelSelectTrigger = ({
  className,
  ...props
}: AIInputModelSelectTriggerProps) => (
  <SelectTrigger
    className={cn(
      'border-none bg-transparent font-medium text-muted-foreground shadow-none transition-colors',
      'hover:bg-accent hover:text-foreground [&[aria-expanded="true"]]:bg-accent [&[aria-expanded="true"]]:text-foreground',
      className
    )}
    {...props}
  />
);

export type AIInputModelSelectContentProps = ComponentProps<
  typeof SelectContent
>;

export const AIInputModelSelectContent = ({
  className,
  ...props
}: AIInputModelSelectContentProps) => (
  <SelectContent className={cn(className)} {...props} />
);

export type AIInputModelSelectItemProps = ComponentProps<typeof SelectItem>;

export const AIInputModelSelectItem = ({
  className,
  ...props
}: AIInputModelSelectItemProps) => (
  <SelectItem className={cn(className)} {...props} />
);

export type AIInputModelSelectValueProps = ComponentProps<typeof SelectValue>;

export const AIInputModelSelectValue = ({
  className,
  ...props
}: AIInputModelSelectValueProps) => (
  <SelectValue className={cn(className)} {...props} />
);
