import type { Metadata } from "next";
import Link from "next/link";

import { Announcement } from "@/components/announcement";
import {
	PageActions,
	PageHeader,
	PageHeaderDescription,
	PageHeaderHeading,
} from "@/components/page-header";
import { Button } from "@/components/ui/button";
import { config } from "@/lib/config";

const title = "Claude Code for Complex Multimodal Workflows";
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
		<div className="grid min-h-[80vh] place-content-center place-items-center">
			<PageHeader>
				<Announcement />
				<PageHeaderHeading className="max-w-4xl">{title}</PageHeaderHeading>
				<PageHeaderDescription>{description}</PageHeaderDescription>
				<PageActions>
					<Button size="sm">
						<Link href={config.links.github}>Get Started</Link>
					</Button>
					<Button size="sm" variant="ghost">
						<Link href="/docs/backend">View Documentation</Link>
					</Button>
				</PageActions>
			</PageHeader>
		</div>
	);
}
