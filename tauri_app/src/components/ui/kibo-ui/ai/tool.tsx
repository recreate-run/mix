import { ChevronDownIcon, ClockIcon, XCircleIcon } from "lucide-react";
import type { ComponentProps, ReactNode } from "react";
import {
	Collapsible,
	CollapsibleContent,
	CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";

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
		{...props}
	/>
);

export type AIToolHeaderProps = ComponentProps<typeof CollapsibleTrigger> & {
	status?: AIToolStatus;
	name: string;
	description?: string;
};

export const AIToolHeader = ({
	className,
	status = "pending",
	name,
	description,
	...props
}: AIToolHeaderProps) => (
	<CollapsibleTrigger
		className={cn(
			"flex w-full items-center justify-between gap-4 hover:cursor-pointer",
			className,
		)}
		{...props}
	>
		<div className="flex items-center gap-2">
			<span className="font-medium text-xs">{name}</span>
		</div>
		<ChevronDownIcon className="size-4 text-muted-foreground transition-transform group-data-[state=open]:rotate-180" />
	</CollapsibleTrigger>
);

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
		className={cn("grid gap-4 overflow-x-auto border-t p-4 text-sm", className)}
		{...props}
	>
		{toolCall && (
			<>
				<AIToolParameters parameters={toolCall.parameters} />
				{(toolCall.result || toolCall.error) && (
					<AIToolResult error={toolCall.error} result={toolCall.result} />
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
	...props
}: AIToolParametersProps) => (
	<div className={cn("space-y-2", className)} {...props}>
		<div className="rounded-md">
			<pre className="overflow-x-scroll whitespace-pre text-muted-foreground text-xs">
				{JSON.stringify(parameters, null, 2)}
			</pre>
		</div>
	</div>
);

export type AIToolResultProps = ComponentProps<"div"> & {
	result?: ReactNode;
	error?: string;
};

export const AIToolResult = ({
	className,
	result,
	error,
	...props
}: AIToolResultProps) => {
	if (!(result || error)) {
		return null;
	}

	return (
		<div className={cn("space-y-2", className)} {...props}>
			<h4 className="font-medium text-muted-foreground text-xs uppercase tracking-wide">
				{error ? "Error" : "Result"}
			</h4>
			<div
				className={cn(
					"overflow-x-scroll whitespace-pre-wrap rounded-md p-3 text-xs",
					error
						? "bg-destructive/10 text-destructive"
						: "bg-muted/50 text-foreground",
				)}
			>
				{error ? <div>{error}</div> : <div>{result}</div>}
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
	<div className={cn("relative space-y-2 mb-2", className)} {...props}>
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
					{...props}
				>
					{children}
				</Collapsible>
			</div>
		</div>
	</div>
);
