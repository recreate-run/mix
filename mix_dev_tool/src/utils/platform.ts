/**
 * Platform detection utilities for Tauri vs Browser environment
 *
 * This module provides utilities to detect whether the app is running
 * in a Tauri desktop environment or a web browser, and check for
 * platform-specific features.
 */

/**
 * Detects if the app is running in a Tauri desktop environment
 * @returns true if running in Tauri, false if running in browser
 */
export const isTauriEnvironment = (): boolean => {
	if (typeof window === "undefined") {
		return false;
	}

	// Check for Tauri internals object which is only available in Tauri apps
	return "__TAURI_INTERNALS__" in window;
};

/**
 * Platform-specific feature detection
 * Use these helpers to check if specific features are available
 */
export const PlatformFeatures = {
	/**
	 * Check if native file picker dialogs are available
	 */
	hasNativeFilePicker: (): boolean => isTauriEnvironment(),

	/**
	 * Check if native system dialogs (alert, confirm, save) are available
	 */
	hasNativeDialogs: (): boolean => isTauriEnvironment(),

	/**
	 * Check if file system access (readDir, stat, etc.) is available
	 */
	hasFileSystemAccess: (): boolean => isTauriEnvironment(),

	/**
	 * Check if shell access (open URLs, files, folders) is available
	 */
	hasShellAccess: (): boolean => isTauriEnvironment(),

	/**
	 * Check if local file write operations are available
	 */
	hasFileWrite: (): boolean => isTauriEnvironment(),
};

/**
 * Get a human-readable platform name
 */
export const getPlatformName = (): "desktop" | "browser" => {
	return isTauriEnvironment() ? "desktop" : "browser";
};
