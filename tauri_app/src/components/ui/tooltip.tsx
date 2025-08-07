import * as React from 'react';
import { cn } from '@/lib/utils';

interface TooltipProps {
  children: React.ReactNode;
}

interface TooltipTriggerProps {
  asChild?: boolean;
  children: React.ReactNode;
}

interface TooltipContentProps {
  className?: string;
  children: React.ReactNode;
}

const TooltipProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  return <>{children}</>;
};

const Tooltip: React.FC<TooltipProps> = ({ children }) => {
  const [isVisible, setIsVisible] = React.useState(false);

  return (
    <div
      className="relative inline-block"
      onMouseEnter={() => setIsVisible(true)}
      onMouseLeave={() => setIsVisible(false)}
    >
      {React.Children.map(children, (child) => {
        if (React.isValidElement(child)) {
          if (child.type === TooltipTrigger) {
            return React.cloneElement(child as React.ReactElement<any>, {
              isVisible,
            });
          }
          if (child.type === TooltipContent) {
            return React.cloneElement(child as React.ReactElement<any>, {
              isVisible,
            });
          }
        }
        return child;
      })}
    </div>
  );
};

const TooltipTrigger: React.FC<
  TooltipTriggerProps & { isVisible?: boolean }
> = ({ asChild, children, isVisible }) => {
  return <>{children}</>;
};

const TooltipContent: React.FC<
  TooltipContentProps & { isVisible?: boolean }
> = ({ className, children, isVisible }) => {
  if (!isVisible) return null;

  return (
    <div
      className={cn(
        '-translate-x-1/2 absolute bottom-full left-1/2 z-50 mb-2 transform overflow-hidden rounded-md bg-gray-900 px-3 py-1.5 text-white text-xs shadow-lg',
        className
      )}
    >
      {children}
      <div className="-translate-x-1/2 absolute top-full left-1/2 h-0 w-0 transform border-transparent border-t-4 border-t-gray-900 border-r-4 border-l-4" />
    </div>
  );
};

export { Tooltip, TooltipTrigger, TooltipContent, TooltipProvider };
