"use client";

import { useState } from "react";
import Link from "next/link";
import { Demo } from "@/lib/demos";
import { ExternalLink, BookOpen } from "lucide-react";
import { CodeBlock } from "@/components/code-block";

interface DemoCardsProps {
	demos: Demo[];
}

export function DemoCards({ demos }: DemoCardsProps) {
	const [selectedDemo, setSelectedDemo] = useState(0);
	const [copied, setCopied] = useState(false);

	const copyToClipboard = async () => {
		try {
			await navigator.clipboard.writeText(demos[selectedDemo].code);
			setCopied(true);
			setTimeout(() => setCopied(false), 2000);
		} catch (err) {
			console.error("Failed to copy:", err);
		}
	};

	if (demos.length === 0) return null;

	return (
		<div className="container px-4 pb-8">
			{/* Selected demo showcase */}
			<div className="max-w-7xl mx-auto">
				<div className="grid grid-cols-1 lg:grid-cols-5 gap-8 items-center">
					{/* Code section */}
					<div className="space-y-4 lg:col-span-2">
						<CodeBlock
							code={demos[selectedDemo].code}
							fileName={demos[selectedDemo].fileName}
							copied={copied}
							onCopy={copyToClipboard}
						/>
					</div>

					{/* Video section */}
					<div className="flex flex-col justify-center lg:col-span-3">
						<div className="rounded-xl overflow-hidden">
							<div className="rounded-xl overflow-hidden bg-black aspect-video">
								{demos[selectedDemo].youtubeId ? (
									<iframe
										key={selectedDemo}
										src={`https://www.youtube.com/embed/${demos[selectedDemo].youtubeId}?vq=hd1080&hd=1&rel=0&modestbranding=1`}
										className="w-full h-full"
										allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
										allowFullScreen
									/>
								) : (
									<video
										key={selectedDemo} // Force re-render when demo changes
										src={demos[selectedDemo].videoSrc}
										controls
										className="w-full h-full object-contain"
										poster={demos[selectedDemo].videoSrc}
									>
										Your browser does not support the video tag.
									</video>
								)}
							</div>
						</div>
						{demos[selectedDemo].videoCaption && (
							<p className="text-sm text-fd-muted-foreground mt-6 text-center font-medium">
								{demos[selectedDemo].videoCaption}
							</p>
						)}
					</div>
				</div>
			</div>

			{/* Demo-specific link or general CTA */}
			<div className="max-w-7xl mx-auto mt-12">
				<div className="flex items-center justify-center gap-4 flex-wrap">
					{demos[selectedDemo].githubUrl && (
						<Link
							href={demos[selectedDemo].githubUrl!}
							target="_blank"
							rel="noopener noreferrer"
							className="group inline-flex items-center gap-2 px-5 py-2.5 text-sm font-medium border-2 border-fd-primary/30 rounded-xl hover:bg-fd-primary/10 hover:border-fd-primary transition-all hover:shadow-lg hover:shadow-fd-primary/20 hover:scale-105"
						>
							<ExternalLink className="h-4 w-4 group-hover:rotate-12 transition-transform" />
							Try on GitHub
						</Link>
					)}
					<Link
						href="https://github.com/recreate-run/mix-cookbooks"
						target="_blank"
						rel="noopener noreferrer"
						className="group inline-flex items-center gap-2 px-5 py-2.5 text-sm font-medium border-2 border-fd-border/50 rounded-xl hover:bg-fd-muted hover:border-fd-border transition-all hover:shadow-lg hover:scale-105"
					>
						<BookOpen className="h-4 w-4 group-hover:scale-110 transition-transform" />
						Browse All Examples
					</Link>
				</div>
			</div>

			{/* Demo selector cards */}
			<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 mt-12 mb-36 max-w-6xl mx-auto">
				{demos.map((demo, index) => (
					<button
						type="button"
						key={demo.title}
						onClick={() => setSelectedDemo(index)}
						className={`group relative p-6 rounded-2xl text-left transition-all duration-300 ${
							selectedDemo === index
								? "bg-gradient-to-br from-fd-primary/10 via-fd-primary/5 to-transparent border border-fd-primary shadow-lg shadow-fd-primary/20 scale-[1.02]"
								: "bg-fd-card border border-fd-border/50 hover:border-fd-primary/30 hover:shadow-md hover:scale-[1.01]"
						}`}
					>
						{selectedDemo === index && (
							<div className="absolute inset-0 bg-gradient-to-br from-fd-primary/5 to-transparent rounded-2xl blur-xl -z-10" />
						)}
						<div>
							<h3
								className={`font-bold text-base mb-2 transition-colors ${
									selectedDemo === index
										? "text-fd-primary"
										: "text-fd-foreground"
								}`}
							>
								{demo.title}
							</h3>
							<p className="text-sm text-fd-muted-foreground leading-relaxed">
								{demo.description}
							</p>
						</div>
					</button>
				))}
			</div>
		</div>
	);
}
