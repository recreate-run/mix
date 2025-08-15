import type { ComponentProps, HTMLAttributes } from 'react';
import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import { cn } from '@/lib/utils';

export type AIMessageProps = HTMLAttributes<HTMLDivElement> & {
  from: 'user' | 'assistant';
};

export const AIMessage = ({ className, from, ...props }: AIMessageProps) => (
  <div
    className={cn(
      'group flex w-full items-end justify-end my-4',
      from === 'user' ? 'is-user' : 'is-assistant flex-row-reverse justify-end',
      '[&>div]:max-w-[100%]',
      className
    )}
    {...props}
  />
);

export type AIMessageContentProps = HTMLAttributes<HTMLDivElement>;

export function AIMessageContent({
  className,
  children,
  ...props
}: AIMessageContentProps) {
  return (
    <div className={cn(' px-4 ', 'text-foreground', className)} {...props}>
      {children}
    </div>
  );
}

export type AIMessageContentInnerProps = HTMLAttributes<HTMLDivElement>;

function AIMessageContentInner({
  className,
  children,
  ...props
}: AIMessageContentInnerProps) {
  return (
    <div
      className={cn(
        ' is-user:dark rounded-xl px-4 py-2 text group-[.is-user]:bg-secondary/80',
        className
      )}
      {...props}
    >
      {children}
    </div>
  );
}

export type AIMessageContentToolbarProps = HTMLAttributes<HTMLDivElement>;

function AIMessageContentToolbar({
  className,
  children,
  ...props
}: AIMessageContentToolbarProps) {
  return (
    <div
      className={cn(
        'px-4 opacity-0 transition-opacity duration-200 group-hover:opacity-100',
        className
      )}
      {...props}
    >
      {children}
    </div>
  );
}

AIMessageContent.Content = AIMessageContentInner;
AIMessageContent.Toolbar = AIMessageContentToolbar;

export type AIMessageAvatarProps = ComponentProps<typeof Avatar> & {
  src: string;
  name?: string;
};

export const AIMessageAvatar = ({
  src,
  name,
  className,
  ...props
}: AIMessageAvatarProps) => (
  <Avatar className={cn('size-8', className)} {...props}>
    <AvatarImage alt="" className="mt-0 mb-0" src={src} />
    <AvatarFallback>{name?.slice(0, 2) || 'ME'}</AvatarFallback>
  </Avatar>
);
