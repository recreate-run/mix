import { useEffect, useState } from "react";
import { useBoundStore } from "@/stores";
import type { Attachment } from "@/stores/attachmentSlice";
import { buildSessionFileUrl } from "@/utils/attachmentUtils";
import { useSessionFiles } from "./useSessionFiles";

export function useFileReference(
	text: string,
	setText: (text: string) => void,
	sessionId?: string,
) {
	const [selected, setSelected] = useState(0);

	const addAttachment = useBoundStore((state) => state.addAttachment);
	const addReference = useBoundStore((state) => state.addReference);

	// Get session files using our new hook
	const { data: sessionFiles = [], isLoading: isLoadingFolder } =
		useSessionFiles(sessionId);

	// Filter files to exclude directories (session files are flat)
	const files = sessionFiles.filter((file) => !file.isDirectory);

	// Detect "@" trigger
	const words = text.split(" ");
	const lastWord = words[words.length - 1];
	const show =
		lastWord.startsWith("@") && !lastWord.includes("/") && !!sessionId;

	// Reset selection when files change
	useEffect(() => {
		setSelected(0);
	}, []);

	// Handle file selection
	const handleSelection = (selectedFile?: Attachment) => {
		const file = selectedFile || files[selected];
		if (!file || !sessionId) return;

		const words = text.split(" ");
		const displayReference = `@${file.name}`;
		words[words.length - 1] = `${displayReference} `;
		const newText = words.join(" ");

		// Build full URL using centralized utility
		const sessionFilePath = `/api/sessions/${sessionId}/files/${file.name}`;
		const fullUrl = buildSessionFileUrl(sessionId, file.name);

		// Add to attachment store
		addAttachment({
			...file,
			// Ensure proper session file path for serving
			path: sessionFilePath,
		});

		// Add reference mapping with full URL to ensure consistency with media array
		addReference(displayReference, fullUrl);
		setText(newText);
	};

	// Handle escape key
	const handleEscape = () => {
		const words = text.split(" ");
		words[words.length - 1] = "";
		const newText = words.join(" ").trim();
		setText(newText);
	};

	// Close dropdown
	const closeDropdown = () => {
		// Session files don't need state cleanup like local files
	};

	return {
		show,
		files,
		selected,
		selectFile: handleSelection,
		currentFolder: null, // Session files are flat - no folders
		isLoadingFolder,
		goBack: undefined, // No folder navigation
		enterSelectedFolder: undefined, // No folder navigation
		close: handleEscape,
		closeDropdown,
	};
}
