import type { Metadata } from "next";
import Link from "next/link";

import { Announcement } from "@/components/announcement";
import { Footer } from "@/components/footer";
import {
	PageActions,
	PageHeader,
	PageHeaderDescription,
	PageHeaderHeading,
} from "@/components/page-header";
import { Button } from "@/components/ui/button";
import { config } from "@/lib/config";

const title = "Claude Code for Multimodal tasks";
const description =
	"Automate marketing video generation, analyze session recordings, and orchestrate complex workflows across Blender, Figma, Logic Pro, and more. Built for startups who need AI-powered creative automation.";

export const metadata: Metadata = {
	title,
	description,
	openGraph: {
		images: [
			{
				url: `/og?title=${encodeURIComponent(
					title,
				)}&description=${encodeURIComponent(description)}`,
			},
		],
	},
	twitter: {
		card: "summary_large_image",
		images: [
			{
				url: `/og?title=${encodeURIComponent(
					title,
				)}&description=${encodeURIComponent(description)}`,
			},
		],
	},
};

export default function HomePage() {
	return (
		<>
			<div className="flex flex-col min-h-[100vh]">
				<div className="flex items-center justify-center py-4 flex-none">
					<div className="container px-4 pb-8">
						<PageHeader className="mb-6">
							<Announcement />
							<PageHeaderHeading className="max-w-4xl text-3xl md:text-4xl">{title}</PageHeaderHeading>
							<PageHeaderDescription className="mb-4 text-sm">{description}</PageHeaderDescription>
						</PageHeader>
						
						<div className="max-w-6xl mx-auto">
							<p className="text-center text-muted-foreground mb-4 max-w-3xl mx-auto text-sm">
								Choose how you want to interact with Mix:
							</p>
							<div className="grid grid-cols-1 md:grid-cols-2 gap-6">
								{/* Terminal UI Card */}
								<div className="relative border rounded-lg p-4 shadow-md bg-card hover:shadow-lg transition-all overflow-hidden group">
									<div className="absolute inset-0 bg-gradient-to-r from-primary/5 to-transparent opacity-50 group-hover:opacity-70 transition-opacity"></div>
									<div className="relative z-10">
										<h3 className="text-lg font-semibold mb-1">Terminal UI</h3>
										<p className="mb-2 text-xs text-muted-foreground leading-tight">
											Interactive command-line interface with slash commands and a command palette.
											Perfect for rapid workflows and AI-powered task automation.
										</p>
										<div className="flex flex-wrap gap-1 mb-3">
											<span className="px-1.5 py-0.5 bg-primary/10 text-primary text-xs rounded-full">Command Palette</span>
											<span className="px-1.5 py-0.5 bg-primary/10 text-primary text-xs rounded-full">Slash Commands</span>
											<span className="px-1.5 py-0.5 bg-primary/10 text-primary text-xs rounded-full">App Integration</span>
										</div>
										<div className="flex flex-col sm:flex-row gap-2">
											<Button size="sm">
												<Link href="/docs/backend">
													Docs
												</Link>
											</Button>
											<Button variant="outline" size="sm">
												<Link 
													href={config.links.github}
													target="_blank"
													rel="noopener noreferrer"
												>
													GitHub
												</Link>
											</Button>
										</div>
									</div>
								</div>
								
								{/* TypeScript SDK Card */}
								<div className="relative border rounded-lg p-4 shadow-md bg-card hover:shadow-lg transition-all overflow-hidden group">
									<div className="absolute inset-0 bg-gradient-to-r from-secondary/5 to-transparent opacity-50 group-hover:opacity-70 transition-opacity"></div>
									<div className="relative z-10">
										<h3 className="text-lg font-semibold mb-1">TypeScript SDK</h3>
										<p className="mb-2 text-xs text-muted-foreground leading-tight">
											Integrate Mix into your applications with our type-safe JavaScript/TypeScript SDK.
											Build custom integrations and workflows programmatically.
										</p>
										<div className="flex flex-wrap gap-1 mb-3">
											<span className="px-1.5 py-0.5 bg-secondary/10 text-secondary text-xs rounded-full">Type-Safe</span>
											<span className="px-1.5 py-0.5 bg-secondary/10 text-secondary text-xs rounded-full">Open Source</span>
											<span className="px-1.5 py-0.5 bg-secondary/10 text-secondary text-xs rounded-full">Full API Access</span>
										</div>
										<div className="flex flex-col sm:flex-row gap-2">
											<Button size="sm">
												<Link href="/docs/sdk">
													Docs
												</Link>
											</Button>
											<Button variant="outline" size="sm">
												<Link 
													href={config.links.sdkGithub}
													target="_blank"
													rel="noopener noreferrer"
												>
													GitHub
												</Link>
											</Button>
										</div>
									</div>
								</div>
							</div>
						</div>
					</div>
				</div>

				<div className="mt-auto">
					<Footer />
				</div>
			</div>
		</>
	);
}