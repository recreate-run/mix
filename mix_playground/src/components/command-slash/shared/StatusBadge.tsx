interface StatusBadgeProps {
  status: 'current' | 'preferred' | 'authenticated' | 'not-authenticated' | 'connected' | 'disconnected';
  label?: string;
}

export function StatusBadge({ status, label }: StatusBadgeProps) {
  const getBadgeStyles = () => {
    switch (status) {
      case 'current':
        return 'rounded-full bg-primary px-1.5 py-0.5 text-primary-foreground text-xs';
      case 'preferred':
        return 'rounded-full bg-primary px-1.5 py-0.5 text-primary-foreground text-xs';
      case 'authenticated':
        return 'rounded-full bg-green-100 px-1.5 py-0.5 text-green-800 text-xs dark:bg-green-800/20 dark:text-green-400';
      case 'not-authenticated':
        return 'rounded-full bg-red-100 px-1.5 py-0.5 text-red-800 text-xs dark:bg-red-800/20 dark:text-red-400';
      case 'connected':
        return 'rounded-full px-2 py-0.5 text-xs bg-green-100 text-green-800 dark:bg-green-800/20 dark:text-green-400';
      case 'disconnected':
        return 'rounded-full px-2 py-0.5 text-xs bg-red-100 text-red-800 dark:bg-red-800/20 dark:text-red-400';
      default:
        return 'rounded-full bg-gray-100 px-1.5 py-0.5 text-gray-800 text-xs dark:bg-gray-800/20 dark:text-gray-400';
    }
  };

  const getDefaultLabel = () => {
    switch (status) {
      case 'current':
        return 'current';
      case 'preferred':
        return 'preferred';
      case 'authenticated':
        return 'authenticated';
      case 'not-authenticated':
        return 'not authenticated';
      case 'connected':
        return 'connected';
      case 'disconnected':
        return 'disconnected';
      default:
        return '';
    }
  };

  return (
    <span className={getBadgeStyles()}>
      {label || getDefaultLabel()}
    </span>
  );
}