import {
  Accessibility,
  Folder,
  Mic,
  Monitor,
} from 'lucide-react';
import { CommandGroup, CommandItem } from '@/components/ui/command';
import { Switch } from '@/components/ui/switch';
import {
  useAccessibilityPermission,
  useFullDiskAccessPermission,
  useMicrophonePermission,
  useScreenRecordingPermission,
} from '@/hooks/usePermissions';
import { BackButton } from './shared/BackButton';

interface PermissionsViewProps {
  onBackToCommands: () => void;
}

export function PermissionsView({
  onBackToCommands,
}: PermissionsViewProps) {
  const accessibility = useAccessibilityPermission(true);
  const fullDiskAccess = useFullDiskAccessPermission(true);
  const screenRecording = useScreenRecordingPermission(true);
  const microphone = useMicrophonePermission(true);

  const permissions = [
    {
      id: 'accessibility',
      label: 'Accessibility',
      icon: Accessibility,
      hook: accessibility,
    },
    {
      id: 'fullDiskAccess',
      label: 'Full Disk Access',
      icon: Folder,
      hook: fullDiskAccess,
    },
    {
      id: 'screenRecording',
      label: 'Screen Recording',
      icon: Monitor,
      hook: screenRecording,
    },
    {
      id: 'microphone',
      label: 'Microphone',
      icon: Mic,
      hook: microphone,
    },
  ];


  const handlePermissionSelect = (permissionId: string) => {
    const permission = permissions.find((p) => p.id === permissionId);
    if (permission && !permission.hook.isGranted) {
      permission.hook.request();
    }
  };

  return (
    <CommandGroup heading="System Permissions">
        <BackButton
          label="Back to Commands"
          onSelect={onBackToCommands}
          value="back-to-commands"
        />

        {permissions.map((permission) => {
        const Icon = permission.icon;
        return (
          <CommandItem
            key={permission.id}
            onSelect={() => handlePermissionSelect(permission.id)}
            value={permission.label}
            className="flex items-center justify-between"
          >
            <Icon className="size-4 text-muted-foreground" />
            <div className="flex-1">
              <div className="flex items-center gap-2 font-medium text-sm">
                {permission.label}
              </div>
              <div className="text-muted-foreground text-xs">
                {permission.hook.isGranted ? 'Granted' : 'Not granted'}
              </div>
            </div>
            <Switch
              checked={permission.hook.isGranted}
              disabled={
                permission.hook.isLoading || permission.hook.isRequesting
              }
              onCheckedChange={(checked) => {
                if (!checked) return; // Only allow requesting, not revoking
                if (!permission.hook.isGranted) {
                  permission.hook.request();
                }
              }}
              onClick={(e) => e.stopPropagation()}
            />
          </CommandItem>
        );
      })}
    </CommandGroup>
  );
}