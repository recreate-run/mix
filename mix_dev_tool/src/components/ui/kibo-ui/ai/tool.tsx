import {
	CheckIcon,
	ChevronDownIcon,
	ClockIcon,
	CopyIcon,
	XCircleIcon,
} from "lucide-react";
import type { ComponentProps, ReactNode } from "react";
import { Button } from "@/components/ui/button";
import {
	Collapsible,
	CollapsibleContent,
	CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { useCopyToClipboard } from "@/hooks/use-copy-to-clipboard";
import { cn } from "@/lib/utils";

const TOOL_CONTENT_TRUNCATE_LIMIT = 80;

export type AIToolStatus = "pending" | "running" | "completed" | "error";

export type AIToolProps = ComponentProps<typeof Collapsible> & {
	status?: AIToolStatus;
};

export const AITool = ({
	className,
	status = "pending",
	...props
}: AIToolProps) => (
	<Collapsible
		className={cn("not-prose mb-4 w-full rounded-md border", className)}
		defaultOpen={true}
		{...props}
	/>
);

// Helper function to safely stringify any value (handles objects, arrays, primitives)
const safeStringify = (value: unknown): string => {
	if (value === null) return "null";
	if (value === undefined) return "undefined";
	if (typeof value === "string") return value;
	if (typeof value === "number" || typeof value === "boolean")
		return String(value);
	if (typeof value === "object") {
		try {
			return JSON.stringify(value);
		} catch {
			return String(value);
		}
	}
	return String(value);
};

// Unified function to format tool content - used by both header and popover
const formatToolContent = (
	content: string | Record<string, unknown>,
	toolName?: string,
	options: { truncate?: boolean; limit?: number } = {},
): string => {
	const { truncate = false, limit = TOOL_CONTENT_TRUNCATE_LIMIT } = options;

	// Only apply special key-value formatting for "Bash" tool
	const shouldApplyKeyValueFormatting = toolName?.toLowerCase() === "bash";

	// For Task tool, only show the prompt parameter
	const isTaskTool = toolName?.toLowerCase() === "task";

	let processedContent = "";

	if (isTaskTool) {
		// Handle Task tool - extract only the "prompt" parameter
		if (typeof content === "object" && content !== null) {
			const promptValue = content.prompt;
			processedContent = promptValue ? safeStringify(promptValue) : "";
		} else {
			// Try to parse as JSON if it's a string
			const stringContent = String(content);
			const trimmed = stringContent.trim();
			if (trimmed.startsWith("{") && trimmed.endsWith("}")) {
				try {
					const parsed = JSON.parse(trimmed);
					const promptValue = parsed.prompt;
					processedContent = promptValue
						? safeStringify(promptValue)
						: stringContent;
				} catch {
					processedContent = stringContent;
				}
			} else {
				processedContent = stringContent;
			}
		}
	} else if (shouldApplyKeyValueFormatting) {
		// Handle object input for barch tool
		if (typeof content === "object" && content !== null) {
			const entries = Object.entries(content);
			if (entries.length === 0) return "{}";

			// Extract only values, safely stringified
			const values = entries.map(([, value]) => safeStringify(value));
			processedContent = values.length === 1 ? values[0] : values.join(", ");
		} else {
			// Handle string input for barch tool
			const stringContent = String(content);
			const trimmed = stringContent.trim();

			// Check for JSON object format
			if (trimmed.startsWith("{") && trimmed.endsWith("}")) {
				try {
					const parsed = JSON.parse(trimmed);
					if (
						typeof parsed === "object" &&
						parsed !== null &&
						!Array.isArray(parsed)
					) {
						const values = Object.values(parsed).map(safeStringify);
						processedContent =
							values.length === 1 ? values[0] : values.join(", ");
					} else {
						processedContent = stringContent;
					}
				} catch {
					processedContent = stringContent;
				}
			} else {
				// Check for simple key:value or key=value patterns
				const keyValueMatches = trimmed.match(/[:=]\s*([^,:=]+)/g);
				if (keyValueMatches && keyValueMatches.length > 0) {
					const values = keyValueMatches.map((match) =>
						match.replace(/^[:=]\s*/, "").trim(),
					);
					processedContent =
						values.length === 1 ? values[0] : values.join(", ");
				} else {
					processedContent = stringContent;
				}
			}
		}

		// Remove outer brackets if they exist (only for barch tool)
		const trimmed = processedContent.trim();
		const withoutBrackets = (() => {
			if (trimmed.startsWith("(") && trimmed.endsWith(")")) {
				return trimmed.slice(1, -1);
			}
			if (trimmed.startsWith("{") && trimmed.endsWith("}")) {
				return trimmed.slice(1, -1);
			}
			if (trimmed.startsWith("[") && trimmed.endsWith("]")) {
				return trimmed.slice(1, -1);
			}
			return trimmed;
		})();

		processedContent = withoutBrackets;
	} else {
		// For all other tools, show content as-is
		if (typeof content === "object" && content !== null) {
			processedContent = JSON.stringify(content, null, 2);
		} else {
			processedContent = String(content);
		}
	}

	// Apply truncation if requested
	if (truncate && processedContent.length > limit) {
		return `${processedContent.substring(0, limit)}...`;
	}

	return processedContent;
};

// Helper function to extract and format tool content for header display (truncated)
const extractToolContent = (toolCall?: {
	name: string;
	parameters: Record<string, unknown>;
	result?: string;
	error?: string;
}): string => {
	if (!toolCall) return "";

	// Priority: result > parameters > error
	let content: string | Record<string, unknown> = "";
	if (toolCall.result) {
		content = toolCall.result;
	} else if (
		toolCall.parameters &&
		Object.keys(toolCall.parameters).length > 0
	) {
		content = toolCall.parameters;
	} else if (toolCall.error) {
		content = toolCall.error;
	}

	if (!content) return "";

	return formatToolContent(content, toolCall.name, { truncate: true });
};

export type AIToolHeaderProps = ComponentProps<typeof CollapsibleTrigger> & {
	status?: AIToolStatus;
	name: string;
	description?: string;
	toolCall?: {
		name: string;
		parameters: Record<string, unknown>;
		result?: string;
		error?: string;
	};
};

export const AIToolHeader = ({
	className,
	status = "pending",
	name,
	description,
	toolCall,
	...props
}: AIToolHeaderProps) => {
	const toolContent = extractToolContent(toolCall);

	return (
		<CollapsibleTrigger
			className={cn(
				"flex w-full items-center justify-between gap-4 hover:cursor-pointer",
				className,
			)}
			{...props}
		>
			<div className="flex items-center gap-2">
				<span className="font-medium text-xs">{name}</span>
				{toolContent && (
					<span className="text-muted-foreground text-xs">{toolContent}</span>
				)}
				{status === "running" && description && (
					<span className="animate-pulse text-muted-foreground text-xs">
						{description}
					</span>
				)}
			</div>
			<ChevronDownIcon className="size-4 text-muted-foreground transition-transform group-data-[state=open]:rotate-180" />
		</CollapsibleTrigger>
	);
};

export type AIToolContentProps = ComponentProps<typeof CollapsibleContent> & {
	toolCall?: {
		name: string;
		parameters: Record<string, unknown>;
		result?: string;
		error?: string;
	};
};

export const AIToolContent = ({
	className,
	toolCall,
	children,
	...props
}: AIToolContentProps) => (
	<CollapsibleContent
		className={cn("grid gap-4 overflow-x-auto p-4 text-sm", className)}
		{...props}
	>
		{toolCall && (
			<>
				<AIToolParameters
					parameters={toolCall.parameters}
					toolName={toolCall.name}
				/>
				{(toolCall.result || toolCall.error) && (
					<AIToolResult
						error={toolCall.error}
						result={toolCall.result}
						toolName={toolCall.name}
					/>
				)}
			</>
		)}
		{children}
	</CollapsibleContent>
);

export type AIToolParametersProps = ComponentProps<"div"> & {
	parameters: Record<string, unknown>;
};

export const AIToolParameters = ({
	className,
	parameters,
	toolName,
	...props
}: AIToolParametersProps & { toolName?: string }) => {
	const formattedContent = formatToolContent(parameters, toolName);
	const { isCopied, copyToClipboard } = useCopyToClipboard();

	const handleCopy = () => {
		copyToClipboard(formattedContent);
	};

	return (
		<div className={cn("flex items-center gap-2", className)} {...props}>
			<Button
				onClick={handleCopy}
				size="icon"
				title={isCopied ? "Copied!" : "Copy parameters"}
				variant="ghost"
			>
				{isCopied ? <CheckIcon /> : <CopyIcon />}
			</Button>
			<pre className="overflow-x-scroll whitespace-pre text-muted-foreground text-xs">
				{formattedContent}
			</pre>
		</div>
	);
};

export type AIToolResultProps = ComponentProps<"div"> & {
	result?: ReactNode;
	error?: string;
};

export const AIToolResult = ({
	className,
	result,
	error,
	toolName,
	...props
}: AIToolResultProps & { toolName?: string }) => {
	const { isCopied, copyToClipboard } = useCopyToClipboard();

	if (!(result || error)) {
		return null;
	}

	const displayContent = error || result;
	const formattedContent =
		typeof displayContent === "string"
			? formatToolContent(displayContent, toolName)
			: displayContent;

	const handleCopy = () => {
		const contentToCopy =
			typeof formattedContent === "string"
				? formattedContent
				: String(formattedContent);
		copyToClipboard(contentToCopy);
	};

	return (
		<div className={cn("space-y-2", className)} {...props}>
			<h4 className="font-medium text-muted-foreground text-xs uppercase tracking-wide">
				{error ? "Error" : "Result"}
			</h4>
			<div
				className={cn(
					"relative overflow-x-scroll whitespace-pre-wrap rounded-md p-3 text-xs",
					error
						? "bg-destructive/10 text-destructive"
						: "bg-muted/50 text-foreground",
				)}
			>
				<Button
					className="absolute top-1 left-1 z-10 size-6"
					onClick={handleCopy}
					size="icon"
					title={isCopied ? "Copied!" : "Copy result"}
					variant="ghost"
				>
					{isCopied ? (
						<CheckIcon className="size-3 text-green-600" />
					) : (
						<CopyIcon className="size-3" />
					)}
				</Button>
				<div className="pl-6">{formattedContent}</div>
			</div>
		</div>
	);
};

// Ladder View Components
export type AIToolLadderProps = ComponentProps<"div">;

export const AIToolLadder = ({
	className,
	children,
	...props
}: AIToolLadderProps) => (
	<div className={cn("relative mb-4 space-y-2", className)} {...props}>
		{children}
	</div>
);

export type AIToolStepProps = ComponentProps<typeof Collapsible> & {
	status?: AIToolStatus;
	stepNumber: number;
	isLast?: boolean;
};

export const AIToolStep = ({
	className,
	status = "pending",
	stepNumber,
	isLast = false,
	children,
	...props
}: AIToolStepProps) => (
	<div className="relative">
		<div className="flex items-center gap-2">
			{/* Step indicator */}

			<div
				className={cn(
					"flex size-4 items-center justify-center rounded-full font-medium text-xs",
					status === "completed" && "text-green-700",
					status === "running" && "animate-pulse text-blue-700",
					status === "error" && " text-red-700",
					status === "pending" && " text-muted-foreground",
				)}
			>
				{status === "completed"}
				{status === "error" && <XCircleIcon className="" />}
				{status === "running" && <ClockIcon className="" />}
				{status === "pending" && stepNumber}
			</div>

			{/* Tool content */}
			<div className="min-w-0 flex-1">
				<Collapsible
					className={cn("not-prose w-full rounded-md ", className)}
					defaultOpen={false}
					{...props}
				>
					{children}
				</Collapsible>
			</div>
		</div>
	</div>
);
