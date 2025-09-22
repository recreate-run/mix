import {
  Accessibility,
  Folder,
  Mic,
  Monitor,
} from 'lucide-react';
import { CommandGroup } from '@/components/ui/command';
import { Switch } from '@/components/ui/switch';
import {
  useAccessibilityPermission,
  useFullDiskAccessPermission,
  useMicrophonePermission,
  useScreenRecordingPermission,
} from '@/hooks/usePermissions';
import { BackButton } from './shared/BackButton';
import { CommandItemWrapper } from './shared/CommandItemWrapper';

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
          <CommandItemWrapper
            key={permission.id}
            id={permission.id}
            value={permission.label}
            onSelect={handlePermissionSelect}
            icon={Icon}
            title={permission.label}
            description={permission.hook.isGranted ? 'Granted' : 'Not granted'}
            className="flex items-center justify-between"
          >
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
          </CommandItemWrapper>
        );
      })}
    </CommandGroup>
  );
}