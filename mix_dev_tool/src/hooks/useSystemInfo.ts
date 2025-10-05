import { useQuery } from "@tanstack/react-query";

type SystemInfo = {
	storageBasePath: string;
};

export function useSystemInfo() {
	return useQuery<SystemInfo>({
		queryKey: ["system", "info"],
		queryFn: async () => {
			// Temporary workaround: Call endpoint directly until SDK is regenerated with getSystemInfo
			const response = await fetch("http://localhost:8080/api/system/info");
			if (!response.ok) {
				throw new Error(`Failed to fetch system info: ${response.statusText}`);
			}
			return response.json();
		},
		staleTime: Number.POSITIVE_INFINITY, // System info never changes during app lifetime
		gcTime: Number.POSITIVE_INFINITY,
	});
}
