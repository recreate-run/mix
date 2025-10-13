import { useState } from "react";
import { useSessionMessages } from "@/hooks/useSessionMessages";
import { toast } from "sonner";
import {
	AIInput,
	AIInputSubmit,
	AIInputTextarea,
	AIInputToolbar,
	AIInputTools,
} from "@/components/ui/kibo-ui/ai/input";
import { FileUploadButton } from "@/components/file-upload-button";
import { formatCurrentModel, usePreferences } from "@/hooks/usePreferences";
import { useBoundStore } from "@/stores";
import { buildSessionFileUrl } from "@/utils/attachmentUtils";
import { AttachmentPreview } from "./attachment-preview";

interface PlaygroundUIProps {
	sessionId: string;
	onSubmit: (text: string) => void;
}

const EXAMPLE_PROMPTS = [
	"Find the top cat video and create a 5 sec tiktok video from it. Add a title animation and export to a video",
	"First, find the top 3 karpathy LLM videos, then find the most important 10 second section from each video. After that, download the sections and show it.",
	"Look at my portfolio in the data and find the top winners and losers in Q4. Show the three most relevant plots.",
];

export function PlaygroundUI({ sessionId, onSubmit }: PlaygroundUIProps) {
	const [inputValue, setInputValue] = useState("");
	const [feedbackMessage, setFeedbackMessage] = useState<string | null>(null);
	const sessionMessages = useSessionMessages(sessionId);
	const messages = sessionMessages.data || [];
	const hasMessages = messages.length > 0;

	// Attachment store hooks
	const attachments = useBoundStore((state) => state.attachments);
	const referenceMap = useBoundStore((state) => state.referenceMap);
	const clearAttachments = useBoundStore((state) => state.clearAttachments);
	const syncWithText = useBoundStore((state) => state.syncWithText);

	// Preferences for model display
	const { data: preferences } = usePreferences();

	const handleSuggestionClick = (prompt: string) => {
		setInputValue(prompt);
	};

	const handleTextChange = (value: string) => {
		setInputValue(value);
		syncWithText(value);
	};

	const handleSubmit = (e: React.FormEvent<HTMLFormElement>) => {
		e.preventDefault();
		if (inputValue.trim()) {
			onSubmit(inputValue);
			setInputValue("");
			clearAttachments();
		}
	};

	// Handle file upload success
	const handleFileUploadSuccess = (fileName: string) => {
		const displayReference = `@${fileName}`;
		const newText = inputValue
			? `${inputValue} ${displayReference} `
			: `${displayReference} `;
		setInputValue(newText);

		const fullUrl = buildSessionFileUrl(sessionId, fileName);
		useBoundStore.getState().addReference(displayReference, fullUrl);

		setFeedbackMessage(`File uploaded successfully: ${fileName}`);
		setTimeout(() => setFeedbackMessage(null), 3000);
	};

	// Handle file upload error
	const handleFileUploadError = (error: string) => {
		setFeedbackMessage(`Error: File upload failed - ${error}`);
		setTimeout(() => setFeedbackMessage(null), 3000);
	};

	// Show centered welcome UI only when no messages
	if (!hasMessages) {
		return (
			<div className="flex h-screen w-full flex-col items-center justify-center bg-background text-foreground">
				{/* Feedback message notification */}
				{feedbackMessage && (
					<div
						className={`-translate-x-1/2 fade-in slide-in-from-top-5 fixed top-4 left-1/2 z-50 transform animate-in rounded-md px-4 py-2 shadow-md duration-300 ${
							feedbackMessage.startsWith("Error:")
								? "bg-destructive text-destructive-foreground"
								: "bg-primary text-primary-foreground"
						}`}
					>
						{feedbackMessage}
					</div>
				)}

				<div className="w-full max-w-2xl space-y-8 px-4">
					{/* Logo/Title */}
					<div className="text-center">
						<h1 className="mb-2 font-bold text-4xl">Mix Playground</h1>
						<p className="text-muted-foreground">
							Claude Code for Multimodal Tasks
						</p>
					</div>

					{/* Example Prompts */}
					<div className="space-y-3">
						{EXAMPLE_PROMPTS.map((prompt, index) => (
							<button
								key={index}
								onClick={() => handleSuggestionClick(prompt)}
								className="w-full rounded-lg border border-border bg-card p-4 text-left transition-colors hover:bg-accent"
								type="button"
							>
								<p className="text-card-foreground">{prompt}</p>
							</button>
						))}
					</div>

					{/* Input Box with attachments */}
					<div className="relative">
						<AttachmentPreview
							attachments={attachments}
							onTextChange={setInputValue}
							referenceMap={referenceMap}
							sessionId={sessionId}
							text={inputValue}
						/>

						<AIInput
							className="border bg-stone-200/60 backdrop-blur-xl dark:bg-stone-700/60"
							onSubmit={handleSubmit}
						>
							<AIInputTextarea
								autoFocus
								onChange={(e) => handleTextChange(e.target.value)}
								placeholder="What would you like to know?"
								value={inputValue}
							/>
							<AIInputToolbar>
								<AIInputTools>
									<div className="absolute bottom-1 left-1 flex">
										{/* File Upload Button */}
										<FileUploadButton
											className="ml-1"
											onUploadError={handleFileUploadError}
											onUploadSuccess={handleFileUploadSuccess}
											sessionId={sessionId}
										/>
									</div>

									{/* Current Model Display */}
									<div className="absolute right-14 bottom-1 text-muted-foreground text-xs">
										{formatCurrentModel(preferences)}
									</div>
								</AIInputTools>
								<AIInputSubmit disabled={!inputValue.trim()} status="ready" />
							</AIInputToolbar>
						</AIInput>
					</div>
				</div>
			</div>
		);
	}

	// Return null when messages exist - parent will show ChatApp
	return null;
}
