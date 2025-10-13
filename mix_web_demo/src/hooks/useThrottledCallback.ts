import { useCallback, useRef } from "react";

// Simple throttle implementation to replace @tanstack/react-pacer
export function useThrottledCallback<T extends (...args: any[]) => any>(
	callback: T,
	options: { wait: number },
): T {
	const lastRan = useRef<number>(Date.now());

	return useCallback(
		(...args: Parameters<T>) => {
			const now = Date.now();
			if (now - lastRan.current >= options.wait) {
				callback(...args);
				lastRan.current = now;
			}
		},
		[callback, options.wait],
	) as T;
}
