/**
 * SDK Code Snippet Component
 * Shows a modal dialog with TypeScript and Python code examples for reproducing a message
 */

import { CodeIcon } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import {
	Dialog,
	DialogContent,
	DialogHeader,
	DialogTitle,
	DialogFooter,
} from "@/components/ui/dialog";
import {
	CodeBlock,
	CodeBlockBody,
	CodeBlockContent,
	CodeBlockCopyButton,
	CodeBlockHeader,
	CodeBlockItem,
	CodeBlockSelect,
	CodeBlockSelectContent,
	CodeBlockSelectItem,
	CodeBlockSelectTrigger,
	CodeBlockSelectValue,
} from "@/components/ui/kibo-ui/code-block";
import type { Attachment } from "@/stores/attachmentSlice";
import {
	generatePythonCode,
	generateTypeScriptCode,
} from "@/utils/sdkCodeGenerator";

interface SdkCodeSnippetProps {
	sessionId: string;
	message: string;
	attachments?: Attachment[];
	serverUrl?: string;
}

export function SdkCodeSnippet({
	sessionId,
	message,
	attachments = [],
	serverUrl,
}: SdkCodeSnippetProps) {
	const [isOpen, setIsOpen] = useState(false);

	// Generate code for both languages
	const tsCode = generateTypeScriptCode({
		sessionId,
		message,
		attachments,
		serverUrl,
	});

	const pyCode = generatePythonCode({
		sessionId,
		message,
		attachments,
		serverUrl,
	});

	const codeData = [
		{
			language: "typescript",
			filename: "example.ts",
			code: tsCode,
		},
		{
			language: "python",
			filename: "example.py",
			code: pyCode,
		},
	];

	return (
		<div className="my-2 w-full">
			{/* Get code button */}
			<Button
				type="button"
				onClick={() => setIsOpen(true)}
				variant="ghost"
				size="sm"
				className="gap-2 text-muted-foreground hover:text-foreground"
			>
				<CodeIcon className="size-4" />
				<span>Get code</span>
			</Button>

			{/* Modal dialog */}
			<Dialog open={isOpen} onOpenChange={setIsOpen}>
				<DialogContent className="!max-w-[70vw] w-[75vw] max-h-[80vh] overflow-y-auto">
					<DialogHeader>
						<DialogTitle>Get code</DialogTitle>
					</DialogHeader>

					<CodeBlock data={codeData} defaultValue="typescript">
						<CodeBlockHeader className="mb-4">
							<CodeBlockSelect>
								<CodeBlockSelectTrigger className="w-40">
									<CodeBlockSelectValue placeholder="Select language" />
								</CodeBlockSelectTrigger>
								<CodeBlockSelectContent>
									{(item) => (
										<CodeBlockSelectItem key={item.language} value={item.language}>
											{item.language === "typescript" ? "TypeScript" : "Python"}
										</CodeBlockSelectItem>
									)}
								</CodeBlockSelectContent>
							</CodeBlockSelect>
							<CodeBlockCopyButton />
						</CodeBlockHeader>
						<CodeBlockBody>
							{(item) => (
								<CodeBlockItem
									key={item.language}
									value={item.language}
									lineNumbers={true}
								>
									<CodeBlockContent
										language={
											item.language === "typescript" ? "typescript" : "python"
										}
									>
										{item.code}
									</CodeBlockContent>
								</CodeBlockItem>
							)}
						</CodeBlockBody>
					</CodeBlock>

					<DialogFooter className="mt-4">
						<Button onClick={() => setIsOpen(false)} variant="outline">
							Close
						</Button>
					</DialogFooter>
				</DialogContent>
			</Dialog>
		</div>
	);
}
