import { Skeleton } from "@/components/ui/skeleton";

function ProviderSkeleton() {
	return (
		<div className="rounded-lg border p-4">
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
				// biome-ignore lint/suspicious/noArrayIndexKey: Static skeleton items that never reorder
				<ProviderSkeleton key={`skeleton-${i}`} />
			))}
		</div>
	);
}
