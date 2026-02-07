import type { Attachment } from "@/stores/attachmentSlice";
import { getFileType, type SupportedFileTypes } from "@/utils/fileTypes";

// Attachment creation utilities
export const createFileAttachment = (
	filePath: string,
	supportedTypes?: SupportedFileTypes,
): Attachment | null => {
	const fileName = filePath.split("/").pop() || filePath;
	const fileType = getFileType(fileName, supportedTypes);

	if (!fileType) {
		console.warn(`Unsupported file type: ${fileName}`);
		return null;
	}

	return {
		id: `file:${filePath}`,
		name: fileName,
		type: fileType,
		path: filePath,
		// Note: Preview URL will be generated when sessionStorageDirectory is available
		extension: fileName.split(".").pop()?.toLowerCase(),
	};
};

// Text reference utilities
export const expandFileReferences = (
	text: string,
	referenceMap: Map<string, string>,
): string => {
	let expandedText = text;

	for (const [displayName, fullUrl] of referenceMap) {
		// Handle file/folder references
		expandedText = expandedText.replace(displayName, fullUrl);
	}

	// Check for any remaining unresolved references and throw exception
	const unresolvedMatches = expandedText.match(/@[^\s]+/g);
	if (unresolvedMatches) {
		const hasHttpReference = unresolvedMatches.some((ref) =>
			ref.startsWith("@http"),
		);
		const errorMessage = hasHttpReference
			? "Cannot use @http:// URLs directly. Use @ to autocomplete session files (e.g., @filename.csv)"
			: `Unresolved file references: ${unresolvedMatches.join(", ")}. Use @ to autocomplete session files.`;

		throw new Error(errorMessage);
	}

	return expandedText;
};

export const removeFileReferences = (
	text: string,
	referenceMap: Map<string, string>,
	fullPath: string,
): string => {
	let updatedText = text;

	for (const [displayName, mappedPath] of referenceMap) {
		if (mappedPath === fullPath) {
			updatedText = updatedText.replace(
				new RegExp(
					`${displayName.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}\\s*`,
					"g",
				),
				"",
			);
		}
	}

	return updatedText;
};
