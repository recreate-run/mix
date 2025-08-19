import { ArrowRightIcon } from 'lucide-react';

import { Badge } from '@/components/ui/badge';

export function Announcement({ content }: { content: string }) {
  return (
    <Badge className="rounded-full" variant="secondary">
      {content} <ArrowRightIcon />
    </Badge>
  );
}
