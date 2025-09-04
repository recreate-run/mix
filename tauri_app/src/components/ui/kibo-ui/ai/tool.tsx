import { ChevronDownIcon, ClockIcon, XCircleIcon } from 'lucide-react';
import type { ComponentProps, ReactNode } from 'react';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible';
import { cn } from '@/lib/utils';

const TOOL_CONTENT_TRUNCATE_LIMIT = 60;

export type AIToolStatus = 'pending' | 'running' | 'completed' | 'error';

export type AIToolProps = ComponentProps<typeof Collapsible> & {
  status?: AIToolStatus;
};

export const AITool = ({
  className,
  status = 'pending',
  ...props
}: AIToolProps) => (
  <Collapsible
    defaultOpen={true}
    className={cn('not-prose mb-4 w-full rounded-md border', className)}
    {...props}
  />
);

// Helper function to extract and format tool content for display
const extractToolContent = (toolCall?: {
  parameters: Record<string, unknown>;
  result?: string;
  error?: string;
}): string => {
  if (!toolCall) return '';

  // Priority: result > parameters > error
  let content = '';
  if (toolCall.result) {
    content = toolCall.result;
  } else if (toolCall.parameters && Object.keys(toolCall.parameters).length > 0) {
    content = JSON.stringify(toolCall.parameters);
  } else if (toolCall.error) {
    content = toolCall.error;
  }

  if (!content) return '';

  // Remove outer brackets if they exist (parentheses, curly brackets, square brackets)
  const trimmed = content.trim();
  const withoutBrackets = (() => {
    if (trimmed.startsWith('(') && trimmed.endsWith(')')) {
      return trimmed.slice(1, -1);
    }
    if (trimmed.startsWith('{') && trimmed.endsWith('}')) {
      return trimmed.slice(1, -1);
    }
    if (trimmed.startsWith('[') && trimmed.endsWith(']')) {
      return trimmed.slice(1, -1);
    }
    return trimmed;
  })();

  // Truncate to configured limit
  return withoutBrackets.length > TOOL_CONTENT_TRUNCATE_LIMIT
    ? `${withoutBrackets.substring(0, TOOL_CONTENT_TRUNCATE_LIMIT)}...`
    : withoutBrackets;
};

export type AIToolHeaderProps = ComponentProps<typeof CollapsibleTrigger> & {
  status?: AIToolStatus;
  name: string;
  description?: string;
  toolCall?: {
    name: string;
    parameters: Record<string, unknown>;
    result?: string;
    error?: string;
  };
};

export const AIToolHeader = ({
  className,
  status = 'pending',
  name,
  description,
  toolCall,
  ...props
}: AIToolHeaderProps) => {
  const toolContent = extractToolContent(toolCall);

  return (
    <CollapsibleTrigger
      className={cn(
        'flex w-full items-center justify-between gap-4 hover:cursor-pointer',
        className
      )}
      {...props}
    >
      <div className="flex items-center gap-2">
        <span className="font-medium text-xs">{name}</span>
        {toolContent && (
          <span className="text-xs text-muted-foreground">
            {toolContent}
          </span>
        )}
        {status === 'running' && description && (
          <span className="text-xs text-muted-foreground animate-pulse">
            {description}
          </span>
        )}
      </div>
      <ChevronDownIcon className="size-4 text-muted-foreground transition-transform group-data-[state=open]:rotate-180" />
    </CollapsibleTrigger>
  );
};

export type AIToolContentProps = ComponentProps<typeof CollapsibleContent> & {
  toolCall?: {
    name: string;
    parameters: Record<string, unknown>;
    result?: string;
    error?: string;
  };
};

export const AIToolContent = ({
  className,
  toolCall,
  children,
  ...props
}: AIToolContentProps) => (
  <CollapsibleContent
    className={cn('grid gap-4 overflow-x-auto  p-4 text-sm', className)}
    {...props}
  >
    {toolCall && (
      <>
        <AIToolParameters parameters={toolCall.parameters} />
        {(toolCall.result || toolCall.error) && (
          <AIToolResult error={toolCall.error} result={toolCall.result} />
        )}
      </>
    )}
    {children}
  </CollapsibleContent>
);

export type AIToolParametersProps = ComponentProps<'div'> & {
  parameters: Record<string, unknown>;
};

export const AIToolParameters = ({
  className,
  parameters,
  ...props
}: AIToolParametersProps) => (
  <div className={cn('space-y-2', className)} {...props}>
    <div className="rounded-md">
      <pre className="overflow-x-scroll whitespace-pre text-muted-foreground text-xs">
        {JSON.stringify(parameters, null, 2)}
      </pre>
    </div>
  </div>
);

export type AIToolResultProps = ComponentProps<'div'> & {
  result?: ReactNode;
  error?: string;
};

export const AIToolResult = ({
  className,
  result,
  error,
  ...props
}: AIToolResultProps) => {
  if (!(result || error)) {
    return null;
  }

  return (
    <div className={cn('space-y-2', className)} {...props}>
      <h4 className="font-medium text-muted-foreground text-xs uppercase tracking-wide">
        {error ? 'Error' : 'Result'}
      </h4>
      <div
        className={cn(
          'overflow-x-scroll whitespace-pre-wrap rounded-md p-3 text-xs',
          error
            ? 'bg-destructive/10 text-destructive'
            : 'bg-muted/50 text-foreground'
        )}
      >
        {error ? <div>{error}</div> : <div>{result}</div>}
      </div>
    </div>
  );
};

// Ladder View Components
export type AIToolLadderProps = ComponentProps<'div'>;

export const AIToolLadder = ({
  className,
  children,
  ...props
}: AIToolLadderProps) => (
  <div className={cn('relative mb-2 space-y-2', className)} {...props}>
    {children}
  </div>
);

export type AIToolStepProps = ComponentProps<typeof Collapsible> & {
  status?: AIToolStatus;
  stepNumber: number;
  isLast?: boolean;
};

export const AIToolStep = ({
  className,
  status = 'pending',
  stepNumber,
  isLast = false,
  children,
  ...props
}: AIToolStepProps) => (
  <div className="relative">
    <div className="flex items-center gap-2">
      {/* Step indicator */}

      <div
        className={cn(
          'flex size-4 items-center justify-center rounded-full font-medium text-xs',
          status === 'completed' && 'text-green-700',
          status === 'running' && 'animate-pulse text-blue-700',
          status === 'error' && ' text-red-700',
          status === 'pending' && ' text-muted-foreground'
        )}
      >
        {status === 'completed'}
        {status === 'error' && <XCircleIcon className="" />}
        {status === 'running' && <ClockIcon className="" />}
        {status === 'pending' && stepNumber}
      </div>

      {/* Tool content */}
      <div className="min-w-0 flex-1">
        <Collapsible
          defaultOpen={false}
          className={cn('not-prose w-full rounded-md ', className)}
          {...props}
        >
          {children}
        </Collapsible>
      </div>
    </div>
  </div>
);
