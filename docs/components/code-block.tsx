"use client";

import { Check, Copy, Github, ChevronDown } from "lucide-react";
import Link from "next/link";
import { useState, useRef, useEffect } from "react";
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
	const [isOpen, setIsOpen] = useState(false);
	const dropdownRef = useRef<HTMLDivElement>(null);
	const currentSnippet = codeSnippets.find((s) => s.language === selectedLanguage) || codeSnippets[0];

	useEffect(() => {
		const handleClickOutside = (event: MouseEvent) => {
			if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
				setIsOpen(false);
			}
		};

		document.addEventListener("mousedown", handleClickOutside);
		return () => document.removeEventListener("mousedown", handleClickOutside);
	}, []);

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
					<div className="relative" ref={dropdownRef}>
						<button
							type="button"
							onClick={() => setIsOpen(!isOpen)}
							className="flex items-center gap-2 pl-3 pr-2 py-1.5 text-xs font-medium rounded-lg bg-fd-muted/50 text-fd-foreground hover:bg-fd-muted/70 transition-all"
						>
							<span>{currentSnippet.language.charAt(0).toUpperCase() + currentSnippet.language.slice(1)}</span>
							<ChevronDown className="h-3 w-3 text-fd-muted-foreground" />
						</button>
						{isOpen && (
							<div className="absolute top-full left-0 mt-1 min-w-[120px] rounded-lg border border-fd-border bg-fd-popover shadow-lg z-50 overflow-hidden">
								{codeSnippets.map((snippet) => (
									<button
										key={snippet.language}
										type="button"
										onClick={() => {
											onLanguageChange(snippet.language);
											setIsOpen(false);
										}}
										className={`w-full text-left px-3 py-2 text-xs font-medium transition-colors ${
											selectedLanguage === snippet.language
												? "bg-fd-accent text-fd-accent-foreground"
												: "hover:bg-fd-accent/50 text-fd-foreground"
										}`}
									>
										{snippet.language.charAt(0).toUpperCase() + snippet.language.slice(1)}
									</button>
								))}
							</div>
						)}
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
