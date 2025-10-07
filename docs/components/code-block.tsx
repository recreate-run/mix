"use client";

import { Check, Copy, Github, ChevronDown } from "lucide-react";
import Link from "next/link";
import type { CodeSnippet } from "@/lib/demos";

interface CodeBlockProps {
	codeSnippets: CodeSnippet[];
	selectedLanguage: string;
	onLanguageChange: (lang: string) => void;
	copied: boolean;
	onCopy: () => void;
}

export function CodeBlock({
	codeSnippets,
	selectedLanguage,
	onLanguageChange,
	copied,
	onCopy,
}: CodeBlockProps) {
	const currentSnippet = codeSnippets.find((s) => s.language === selectedLanguage) || codeSnippets[0];

	return (
		<div className="rounded-2xl border-2 border-fd-border/50 bg-gradient-to-br from-fd-card to-fd-muted/20 overflow-hidden shadow-xl h-[520px] flex flex-col">
			<div className="bg-gradient-to-r from-fd-muted/80 to-fd-muted/60 backdrop-blur-sm px-5 py-0 border-b border-fd-border/50 flex items-center justify-between flex-none">
				<div className="flex items-center gap-3">
					<div className="flex gap-2">
						<div className="size-3 rounded-full bg-red-500/30 ring-1 ring-red-500/20"></div>
						<div className="size-3 rounded-full bg-yellow-500/30 ring-1 ring-yellow-500/20"></div>
						<div className="size-3 rounded-full bg-green-500/30 ring-1 ring-green-500/20"></div>
					</div>
					<div className="h-4 w-px bg-fd-border/50" />
					<span className="text-xs font-medium text-fd-muted-foreground">
						{currentSnippet.fileName}
					</span>
				</div>
				<div className="flex items-center gap-2">
					{/* Language selector dropdown */}
					<div className="relative">
						<select
							value={selectedLanguage}
							onChange={(e) => onLanguageChange(e.target.value)}
							className="appearance-none pl-3 pr-8 py-1.5 text-xs font-medium rounded-lg bg-fd-muted/50 text-fd-foreground border-0 hover:bg-fd-muted/70 transition-all cursor-pointer focus:outline-none focus:ring-0"
						>
							{codeSnippets.map((snippet) => (
								<option key={snippet.language} value={snippet.language}>
									{snippet.language.charAt(0).toUpperCase() + snippet.language.slice(1)}
								</option>
							))}
						</select>
						<ChevronDown className="absolute right-2 top-1/2 -translate-y-1/2 h-3 w-3 text-fd-muted-foreground pointer-events-none" />
					</div>
					{currentSnippet.githubUrl && (
						<Link
							href={currentSnippet.githubUrl}
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
			<pre className="p-8 text-[13px] leading-[1.8] overflow-y-auto bg-gradient-to-b from-transparent to-fd-muted/10 flex-1 min-h-0 font-mono">
				<code className={`language-${currentSnippet.language} block whitespace-pre font-mono`}>{currentSnippet.code}</code>
			</pre>
		</div>
	);
}
