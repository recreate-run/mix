import { ReactNode } from 'react';
import { LucideIcon } from 'lucide-react';
import { CommandItem } from '@/components/ui/command';

interface CommandItemWrapperProps {
  id: string;
  value?: string;
  onSelect: (value: string) => void;
  icon?: LucideIcon;
  title: string;
  description?: string;
  badge?: ReactNode;
  className?: string;
  disabled?: boolean;
  children?: ReactNode;
}

export function CommandItemWrapper({
  id,
  value,
  onSelect,
  icon: Icon,
  title,
  description,
  badge,
  className = '',
  disabled = false,
  children,
}: CommandItemWrapperProps) {
  return (
    <CommandItem
      key={id}
      onSelect={() => onSelect(id)}
      value={value || id}
      className={`${className} ${disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
    >
      {Icon && <Icon className="size-4 text-muted-foreground" />}
      <div className="flex-1">
        <div className="flex items-center gap-2 font-medium text-sm">
          {title}
          {badge}
        </div>
        {description && (
          <div className="text-muted-foreground text-xs">
            {description}
          </div>
        )}
      </div>
      {children}
    </CommandItem>
  );
}