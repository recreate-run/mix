import type { Metadata } from "next";
import Link from "next/link";

import { Footer } from "@/components/footer";
import {
	PageActions,
	PageHeader,
	PageHeaderDescription,
	PageHeaderHeading,
} from "@/components/page-header";
import { Button } from "@/components/ui/button";
import { DemoCards } from "@/components/demo-cards";
import { homepageDemos } from "@/lib/demos";
import { config } from "@/lib/config";

const title = "Multimodal Agent SDK";
const description =
	"SDK-first agent platform that analyzes videos, reads images, searches the web, and orchestrates complex workflows.";

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
			<div className="flex flex-col">
				<div className="flex items-center justify-center py-4 flex-none">
					<div className="container px-4 pb-8">
						<PageHeader className="mb-6">
							<PageHeaderHeading className="max-w-4xl text-3xl md:text-4xl">{title}</PageHeaderHeading>
							<PageHeaderDescription className="mb-4 text-sm">{description}</PageHeaderDescription>
							<PageActions>
								<Link href="/docs/mix/quickstart">
									<Button size="sm">Get Started</Button>
								</Link>
								<Link href="/docs/mix">
									<Button variant="outline" size="sm">View Docs</Button>
								</Link>
							</PageActions>
						</PageHeader>
					</div>
				</div>

				{homepageDemos.length > 0 && (
					<DemoCards demos={homepageDemos} />
				)}

				<Footer />
			</div>
		</>
	);
}