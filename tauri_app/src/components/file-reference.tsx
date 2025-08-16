import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip';

interface FileReferenceProps {
  fullPath: string;
  children: React.ReactNode;
}

export function FileReference({
  fullPath,
  children,
}: FileReferenceProps) {
  return (
    <Tooltip>
      <TooltipTrigger>
        <span className="rounded bg-blue-50 px-1 font-medium text-blue-600">
          {children}
        </span>
      </TooltipTrigger>
      <TooltipContent>{fullPath}</TooltipContent>
    </Tooltip>
  );
}
