import { Monitor } from 'lucide-react';
import { useAppIcon } from '@/hooks/useOpenApps';

interface AppIconProps {
  bundleId: string;
  name: string;
  className?: string;
}

export function AppIcon({ bundleId, name, className = "size-4" }: AppIconProps) {
  const { iconBase64, isLoading, error } = useAppIcon(bundleId);

  if (error || !iconBase64) {
    return <Monitor className={`${className} text-gray-500`} />;
  }

  if (isLoading) {
    return <Monitor className={`${className} text-gray-400 animate-pulse`} />;
  }

  return (
    <img 
      src={`data:image/png;base64,${iconBase64}`} 
      alt={`${name} icon`}
      className={`${className} rounded-sm`}
    />
  );
}