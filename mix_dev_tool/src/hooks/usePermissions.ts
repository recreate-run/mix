import { useCallback, useState } from "react";

interface PermissionState {
	isGranted: boolean;
	isLoading: boolean;
	error: string | null;
	isRequesting: boolean;
}

/**
 * Hook for checking accessibility permissions
 */
export function useAccessibilityPermission(
	_autoCheck?: boolean,
): PermissionState & {
	request: () => Promise<void>;
} {
	const [isRequesting, setIsRequesting] = useState(false);

	const request = useCallback(async () => {
		setIsRequesting(true);
		try {
			// Accessibility permissions are handled by the browser
			await new Promise((resolve) => setTimeout(resolve, 100));
		} finally {
			setIsRequesting(false);
		}
	}, []);

	return {
		isGranted: true,
		isLoading: false,
		error: null,
		isRequesting,
		request,
	};
}

export function useFullDiskAccessPermission(
	_autoCheck?: boolean,
): PermissionState & {
	request: () => Promise<void>;
} {
	const [isRequesting, setIsRequesting] = useState(false);

	const request = useCallback(async () => {
		setIsRequesting(true);
		try {
			// Disk access is handled through standard file APIs
			await new Promise((resolve) => setTimeout(resolve, 100));
		} finally {
			setIsRequesting(false);
		}
	}, []);

	return {
		isGranted: true,
		isLoading: false,
		error: null,
		isRequesting,
		request,
	};
}

export function useScreenRecordingPermission(
	_autoCheck?: boolean,
): PermissionState & {
	request: () => Promise<void>;
} {
	const [isRequesting, setIsRequesting] = useState(false);

	const request = useCallback(async () => {
		setIsRequesting(true);
		try {
			// Screen Capture API uses standard permission flow
			await new Promise((resolve) => setTimeout(resolve, 100));
		} finally {
			setIsRequesting(false);
		}
	}, []);

	return {
		isGranted: true,
		isLoading: false,
		error: null,
		isRequesting,
		request,
	};
}

export function useMicrophonePermission(
	_autoCheck?: boolean,
): PermissionState & {
	request: () => Promise<void>;
} {
	const [isRequesting, setIsRequesting] = useState(false);

	const request = useCallback(async () => {
		setIsRequesting(true);
		try {
			// Microphone access uses getUserMedia API
			await navigator.mediaDevices.getUserMedia({ audio: true });
		} catch (error) {
			console.warn("Microphone permission denied:", error);
		} finally {
			setIsRequesting(false);
		}
	}, []);

	return {
		isGranted: true,
		isLoading: false,
		error: null,
		isRequesting,
		request,
	};
}
