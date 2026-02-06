import { ThinkingLevel } from "mix-typescript-sdk/models/operations/sendmessage";
import { useState } from "react";
import { FileUploadButton } from "@/components/file-upload-button";
import { Badge } from "@/components/ui/badge";
import {
	AIInput,
	AIInputModelSelectTrigger,
	AIInputSubmit,
	AIInputTextarea,
	AIInputToolbar,
	AIInputTools,
} from "@/components/ui/kibo-ui/ai/input";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectValue,
} from "@/components/ui/select";
import { formatCurrentModel, usePreferences } from "@/hooks/usePreferences";
import { useSessionMessages } from "@/hooks/useSessionMessages";
import { DEFAULT_THINKING_LEVEL, EXAMPLE_PROMPTS } from "@/lib/data";
import { useBoundStore } from "@/stores";
import { AttachmentPreview } from "./attachment-preview";

interface PlaygroundWelcomeProps {
	sessionId: string;
	onSubmit: (text: string, thinkingLevel: ThinkingLevel) => void;
	onClear: () => Promise<void>;
}

export function PlaygroundWelcome({
	sessionId,
	onSubmit,
	onClear: _onClear,
}: PlaygroundWelcomeProps) {
	const [inputValue, setInputValue] = useState("");
	const [feedbackMessage, setFeedbackMessage] = useState<string | null>(null);
	const [thinkingLevel, setThinkingLevel] = useState<ThinkingLevel>(
		DEFAULT_THINKING_LEVEL,
	);
	const sessionMessages = useSessionMessages(sessionId);
	const messages = sessionMessages.data || [];
	const hasMessages = messages.length > 0;

	// Attachment store hooks
	const attachments = useBoundStore((state) => state.attachments);
	const referenceMap = useBoundStore((state) => state.referenceMap);
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
			onSubmit(inputValue, thinkingLevel);
			setInputValue("");
			// Don't clear attachments here - ChatApp will use them for the initial message submission
		}
	};

	const handleKeyDown = (e: React.KeyboardEvent<HTMLTextAreaElement>) => {
		// Handle Enter for form submission (without shift for new line)
		if (e.key === "Enter" && !e.shiftKey) {
			e.preventDefault();
			const form = e.currentTarget.form;
			if (form) {
				form.requestSubmit();
			}
		}
	};

	// Handle file upload success
	const handleFileUploadSuccess = (fileName: string, fileUrl: string) => {
		const displayReference = `@${fileName}`;
		const newText = inputValue
			? `${inputValue} ${displayReference} `
			: `${displayReference} `;
		setInputValue(newText);

		// Use URL from backend directly
		useBoundStore.getState().addReference(displayReference, fileUrl);

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
			<div className="relative flex h-screen w-full flex-col items-center justify-center overflow-hidden bg-background text-foreground">
				{/* Gradient Mesh Background */}
				<div className="gradient-mesh-bg" />

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

				<div className="fade-in slide-in-from-bottom-4 relative z-10 w-full max-w-2xl animate-in space-y-8 px-4 duration-700 md:space-y-12">
					{/* Logo/Title */}
					<div className="text-center">
						<h1 className="title-shadow mb-3 flex flex-col items-center justify-center gap-2 text-4xl leading-tight tracking-tight md:mb-4 md:gap-3 md:text-5xl lg:text-6xl">
							<div className="flex items-center gap-2 md:gap-4">
								<img
									src="/256x256.png"
									alt="Mix Logo"
									className="size-12 drop-shadow-xl transition-transform duration-300 hover:rotate-6 md:size-16"
								/>
								<span className="gradient-text">Mix Playground</span>
							</div>
						</h1>
						<p className="subtitle-text text-muted-foreground text-base md:text-lg lg:text-xl">
							try the open-source agents SDK for web-apps
						</p>
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
							className="input-glow-hover input-glow-focus border-2 bg-stone-200/80 shadow-lg backdrop-blur-xl transition-all duration-300 dark:bg-stone-700/80"
							onSubmit={handleSubmit}
						>
							<AIInputTextarea
								autoFocus
								onChange={(e) => handleTextChange(e.target.value)}
								onKeyDown={handleKeyDown}
								placeholder="What would you like to know?"
								value={inputValue}
								className="text-base"
							/>
							<AIInputToolbar>
								<AIInputTools>
									<div className="absolute bottom-1 left-1 flex items-center gap-1.5">
										{/* File Upload Button */}
										<FileUploadButton
											className="ml-1"
											onUploadError={handleFileUploadError}
											onUploadSuccess={handleFileUploadSuccess}
											sessionId={sessionId}
										/>

										{/* Thinking Level Selector */}
										<Select
											onValueChange={(value) =>
												setThinkingLevel(value as ThinkingLevel)
											}
											value={thinkingLevel}
										>
											<AIInputModelSelectTrigger
												className="h-8 w-auto min-w-[120px] text-xs"
												size="sm"
											>
												<SelectValue placeholder="Thinking: Off" />
											</AIInputModelSelectTrigger>
											<SelectContent align="start">
												<SelectItem value={ThinkingLevel.Off}>
													Thinking: Off
												</SelectItem>
												<SelectItem value={ThinkingLevel.Basic}>
													Thinking: Low
												</SelectItem>
												<SelectItem value={ThinkingLevel.Medium}>
													Thinking: Medium
												</SelectItem>
												<SelectItem value={ThinkingLevel.Maximum}>
													Thinking: High
												</SelectItem>
											</SelectContent>
										</Select>
									</div>

									{/* Current Model Display */}
									<div className="absolute right-14 bottom-1 hidden text-muted-foreground text-xs md:block">
										{formatCurrentModel(preferences)}
									</div>
								</AIInputTools>
								<AIInputSubmit disabled={!inputValue.trim()} status="ready" />
							</AIInputToolbar>
						</AIInput>
					</div>

					{/* Example Prompts */}
					<div className="flex flex-wrap justify-center gap-3">
						{EXAMPLE_PROMPTS.map((example) => (
							<Badge
								key={example.name}
								variant="outline"
								className="example-prompt-badge cursor-pointer px-4 py-2 font-medium text-sm"
								onClick={() => handleSuggestionClick(example.prompt)}
							>
								{example.name}
							</Badge>
						))}
					</div>
				</div>
			</div>
		);
	}

	// Return null when messages exist - parent will show ChatApp
	return null;
}
