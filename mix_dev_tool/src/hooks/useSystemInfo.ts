import { useQuery } from "@tanstack/react-query";
import { mix } from "@/lib/mix-sdk";

export function useSystemInfo() {
	return useQuery({
		queryKey: ["system", "info"],
		queryFn: async () => {
			const response = await mix.system.getSystemInfo();
			return response;
		},
		staleTime: Number.POSITIVE_INFINITY, // System info never changes during app lifetime
		gcTime: Number.POSITIVE_INFINITY,
	});
}
