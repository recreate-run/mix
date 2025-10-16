import { AlertCircle, CheckCircle, Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import type { CallbackResultData } from "mix-typescript-sdk/models";

interface CallbackResultDisplayProps {
	result: CallbackResultData;
	className?: string;
}

export function CallbackResultDisplay({
	result,
	className,
}: CallbackResultDisplayProps) {
	// Determine state for unified styling
	const getState = () => {
		if (result.callbackType === "sub_agent") {
			const isComplete =
				result.subagentResult !== undefined || result.error !== undefined;
			if (!isComplete) return "running";
		}
		return result.success ? "success" : "error";
	};

	const state = getState();

	// Unified styling based on state
	const stateStyles = {
		success: "border-green-500",
		error: "border-red-500",
		running: "border-blue-500",
	};

	const iconStyles = {
		success: "text-green-600 dark:text-green-400",
		error: "text-red-600 dark:text-red-400",
		running: "text-blue-600 dark:text-blue-400",
	};

	// Unified icon component
	const StateIcon = () => {
		if (state === "success") {
			return;
		}
		if (state === "error") {
			return <AlertCircle className={cn("size-5", iconStyles.error)} />;
		}
		return (
			<Loader2 className={cn("size-5 animate-spin", iconStyles.running)} />
		);
	};

	// Render bash script callback
	if (result.callbackType === "bash_script") {
		return (
			<div className={cn("rounded-lg p-4 flex gap-3", className)}>
				<StateIcon />
				<div className="flex-1 min-w-0 space-y-2">
					<div className="flex items-baseline gap-2 flex-wrap">
						<span className="font-semibold">{result.callbackName}</span>
						<span className="text-xs text-muted-foreground/70">
							{result.toolName}
						</span>
						{result.nonBlocking && (
							<span className="text-[10px] px-1.5 py-0.5 border border-blue-300 dark:border-blue-700 rounded text-blue-600 dark:text-blue-400">
								async
							</span>
						)}
					</div>

					{result.stdout && (
						<pre className="bg-background/50 rounded p-2 text-xs overflow-x-auto whitespace-pre-wrap break-words">
							<code>{result.stdout}</code>
						</pre>
					)}

					{result.stderr && (
						<pre className="bg-background/50 rounded p-2 text-xs overflow-x-auto whitespace-pre-wrap break-words text-red-600 dark:text-red-400">
							<code>{result.stderr}</code>
						</pre>
					)}

					{result.exitCode !== undefined && result.exitCode !== 0 && (
						<div className="text-xs text-muted-foreground">
							Exit code: {result.exitCode}
						</div>
					)}

					{result.error && !result.stderr && (
						<div className="text-xs text-red-600 dark:text-red-400">
							{result.error}
						</div>
					)}
				</div>
			</div>
		);
	}

	// Render subagent callback
	if (result.callbackType === "sub_agent") {
		return (
			<div className={cn("p-4 flex gap-3", className)}>
				<StateIcon />
				<div className="flex-1 min-w-0 space-y-2">
					<div className="flex items-baseline gap-2 flex-wrap">
						<span className="font-semibold">{result.callbackName}</span>
						<span className="text-xs text-muted-foreground/70">
							{result.toolName}
						</span>
					</div>

					{result.subagentResult && (
						<div className="bg-background/50 rounded p-2 text-xs">
							{result.subagentResult}
						</div>
					)}

					{!result.success && result.error && (
						<div className="text-xs text-red-600 dark:text-red-400">
							{result.error}
						</div>
					)}
				</div>
			</div>
		);
	}

	// Fallback for unknown callback types
	return null;
}
