import { Skeleton } from "@/components/ui/skeleton";

export function ProviderSkeleton() {
  return (
    <div className="p-4 border rounded-lg">
      <div className="flex items-center justify-between">
        <div className="space-y-2">
          <Skeleton className="h-5 w-24" />
          <div className="flex items-center gap-2">
            <Skeleton className="h-5 w-20" />
            <Skeleton className="h-5 w-16" />
          </div>
        </div>
        <Skeleton className="h-8 w-16" />
      </div>
    </div>
  );
}

export function ProvidersLoadingSkeleton() {
  return (
    <div className="space-y-3">
      {Array.from({ length: 4 }).map((_, i) => (
        <ProviderSkeleton key={i} />
      ))}
    </div>
  );
}