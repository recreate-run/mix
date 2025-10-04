import type { LucideIcon } from 'lucide-react';
import { CommandGroup, CommandItem } from '@/components/ui/command';
import { slashCommands } from '@/utils/slash-commands';

interface CommandsListViewProps {
  onCommandExecute: (commandId: string) => void;
}

export function CommandsListView({ onCommandExecute }: CommandsListViewProps) {
  const handleCommandSelect = (commandId: string) => {
    const command = slashCommands.find((c) => c.id === commandId);
    if (command) {
      onCommandExecute(commandId);
    }
  };

  return (
    <CommandGroup heading="Commands">
      {slashCommands.map((command) => {
        const Icon = command.icon as LucideIcon;
        return (
          <CommandItem
            key={command.id}
            onSelect={() => handleCommandSelect(command.id)}
            value={command.id}
          >
            {Icon && <Icon className="size-4 text-muted-foreground" />}
            <div className="flex-1">
              <div className="flex items-center gap-2 font-medium text-sm">
                {command.name}
              </div>
              {command.description && (
                <div className="text-muted-foreground text-xs">
                  {command.description}
                </div>
              )}
            </div>
          </CommandItem>
        );
      })}
    </CommandGroup>
  );
}
