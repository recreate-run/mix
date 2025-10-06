"use client";

import { Check, Copy, Github } from "lucide-react";
import Link from "next/link";

interface CodeBlockProps {
	code: string;
	fileName: string;
	copied: boolean;
	onCopy: () => void;
	githubUrl?: string;
}

export function CodeBlock({ code, fileName, copied, onCopy, githubUrl }: CodeBlockProps) {
	return (
		<div className="rounded-2xl border-2 border-fd-border/50 bg-gradient-to-br from-fd-card to-fd-muted/20 overflow-hidden shadow-xl h-[600px] flex flex-col">
			<div className="bg-gradient-to-r from-fd-muted/80 to-fd-muted/60 backdrop-blur-sm px-5 py-0 border-b border-fd-border/50 flex items-center justify-between flex-none">
				<div className="flex items-center gap-3">
					<div className="flex gap-2">
						<div className="size-3 rounded-full bg-red-500/30 ring-1 ring-red-500/20"></div>
						<div className="size-3 rounded-full bg-yellow-500/30 ring-1 ring-yellow-500/20"></div>
						<div className="size-3 rounded-full bg-green-500/30 ring-1 ring-green-500/20"></div>
					</div>
					<div className="h-4 w-px bg-fd-border/50" />
					<span className="text-xs font-medium text-fd-muted-foreground">
						{fileName}
					</span>
				</div>
				<div className="flex items-center gap-2">
					{githubUrl && (
						<Link
							href={githubUrl}
							target="_blank"
							rel="noopener noreferrer"
							className="p-2 rounded-lg hover:bg-fd-muted/80 transition-all text-fd-muted-foreground hover:text-fd-foreground hover:scale-105 active:scale-95"
							title="View on GitHub"
						>
							<Github className="h-4 w-4" />
						</Link>
					)}
					<button
						type="button"
						onClick={onCopy}
						className="p-2 rounded-lg hover:bg-fd-muted/80 transition-all text-fd-muted-foreground hover:text-fd-foreground hover:scale-105 active:scale-95"
						title="Copy code"
					>
						{copied ? (
							<Check className="h-4 w-4 text-green-500" />
						) : (
							<Copy className="h-4 w-4" />
						)}
					</button>
				</div>
			</div>
			<pre className="p-6 text-sm leading-relaxed overflow-y-auto bg-gradient-to-b from-transparent to-fd-muted/10 flex-1 min-h-0 whitespace-pre-wrap break-words">
				<code className="language-python">{code}</code>
			</pre>
		</div>
	);
}
