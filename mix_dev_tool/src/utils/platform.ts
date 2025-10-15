/**
 * Platform detection utilities
 *
 * Feature detection for platform-specific capabilities
 */

/**
 * Platform-specific feature detection
 * Use these helpers to check if specific features are available
 */
export const PlatformFeatures = {
	/**
	 * Check if native file picker dialogs are available
	 */
	hasNativeFilePicker: (): boolean => false,

	/**
	 * Check if native system dialogs are available
	 */
	hasNativeDialogs: (): boolean => false,

	/**
	 * Check if advanced file system access is available
	 */
	hasFileSystemAccess: (): boolean => false,

	/**
	 * Check if shell access is available
	 */
	hasShellAccess: (): boolean => false,

	/**
	 * Check if local file write operations are available
	 */
	hasFileWrite: (): boolean => false,
};
