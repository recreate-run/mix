import {
  IconSettings,
} from '@tabler/icons-react';
import { useState } from 'react';
import { SettingsDialog } from '@/components/settings-dialog';

import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar';
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from '@/components/ui/sidebar';

export function NavUser({
  user,
}: {
  user: {
    name: string;
    email: string;
    avatar: string;
  };
}) {
  const [showSettingsDialog, setShowSettingsDialog] = useState(false);

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton
          onClick={() => setShowSettingsDialog(true)}
          className="flex items-center gap-3"
          size="lg"
        >
          <Avatar className="h-8 w-8 rounded-lg grayscale">
            <AvatarImage alt={user.name} src={user.avatar} />
            <AvatarFallback className="rounded-lg">CN</AvatarFallback>
          </Avatar>
          <div className="grid flex-1 text-left text-sm leading-tight">
            <span className="truncate font-medium">{user.name}</span>
            <span className="truncate text-muted-foreground text-xs">
              {user.email}
            </span>
          </div>
          <IconSettings className="ml-auto h-4 w-4" />
        </SidebarMenuButton>
      </SidebarMenuItem>
      <SettingsDialog
        open={showSettingsDialog}
        onOpenChange={setShowSettingsDialog}
      />
    </SidebarMenu>
  );
}
