import * as React from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/utils';

const buttonVariants = cva(
  'inline-flex items-center justify-center whitespace-nowrap rounded-md text-sm font-medium ring-offset-fd-background transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-fd-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        default: 'bg-fd-primary text-fd-primary-foreground hover:bg-fd-primary/90',
        destructive: 'bg-fd-destructive text-fd-destructive-foreground hover:bg-fd-destructive/90',
        outline: 'border border-fd-input bg-fd-background hover:bg-fd-accent hover:text-fd-accent-foreground',
        secondary: 'bg-fd-secondary text-fd-secondary-foreground hover:bg-fd-secondary/80',
        ghost: 'hover:bg-fd-accent hover:text-fd-accent-foreground',
        link: 'text-fd-primary underline-offset-4 hover:underline',
      },
      size: {
        default: 'h-10 px-4 py-2',
        sm: 'h-9 rounded-md px-3',
        lg: 'h-11 rounded-md px-8',
        icon: 'h-10 w-10',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  }
);

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, ...props }, ref) => {
    return (
      <button
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    );
  }
);
Button.displayName = 'Button';

export { Button, buttonVariants };