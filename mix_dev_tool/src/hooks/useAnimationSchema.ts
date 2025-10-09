import { useQuery } from "@tanstack/react-query";
import { CACHE_KEYS } from "@/lib/cache-keys";
import { getGsapUrl } from "@/utils/backendUrl";
import { type AnimationSchema, fetchAnimationSchema } from "@/utils/gsapApi";

async function loadAnimationSchema(
	animationName: string,
): Promise<AnimationSchema | null> {
	const baseServerUrl = getGsapUrl();

	if (!animationName) {
		throw new Error("Animation name is required");
	}

	if (!baseServerUrl) {
		throw new Error("GSAP server URL is not configured");
	}

	return await fetchAnimationSchema(animationName, baseServerUrl);
}

interface UseAnimationSchemaOptions {
	animationName?: string | null;
	enabled?: boolean;
}

export function useAnimationSchema({
	animationName,
	enabled = true,
}: UseAnimationSchemaOptions = {}) {
	return useQuery({
		queryKey: animationName ? CACHE_KEYS.animationSchema(animationName) : [],
		queryFn: () => loadAnimationSchema(animationName!),
		enabled: enabled && !!animationName,
		refetchOnWindowFocus: false,
	});
}
