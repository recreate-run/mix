import { useMutation, useQueryClient } from "@tanstack/react-query";
import { CACHE_KEYS } from "@/lib/cache-keys";
import { mix } from "@/lib/mix-sdk";

interface UploadFileParams {
	sessionId: string;
	file: File;
}

interface FileUploadResult {
	name: string;
	size: number;
	modified: number;
	isDir: boolean;
	url: string; // Full absolute URL returned from backend
}

async function uploadFile({
	sessionId,
	file,
}: UploadFileParams): Promise<FileUploadResult> {
	const response = await mix.files.upload({
		id: sessionId,
		requestBody: {
			file,
		},
	});

	return {
		name: response.name,
		size: response.size,
		modified: response.modified,
		isDir: response.isDir,
		url: response.url,
	};
}

export function useFileUpload() {
	const queryClient = useQueryClient();

	return useMutation({
		mutationFn: uploadFile,
		onSuccess: (_, variables) => {
			// Invalidate session files cache to refresh @ menu
			queryClient.invalidateQueries({
				queryKey: CACHE_KEYS.sessionFiles(variables.sessionId),
			});

			// Optionally invalidate session messages to update attachment references
			queryClient.invalidateQueries({
				queryKey: CACHE_KEYS.sessionMessages(variables.sessionId),
			});
		},
		onError: (error) => {
			console.error("File upload failed:", error);
		},
	});
}
