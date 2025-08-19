import Link from "next/link";
import { Badge } from "@/components/ui/badge";

export function Announcement() {
  return (
    <Link
      href="/docs"
      className="inline-flex items-center rounded-lg bg-fd-muted px-3 py-1 text-sm font-medium"
    >
      <Badge variant="secondary" className="mr-2">
        Beta
      </Badge>
      <span>
        Mix is now available for early access →
      </span>
    </Link>
  );
}