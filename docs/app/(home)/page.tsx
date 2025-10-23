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

const title = "The Production Ready Agents SDK";
const description =
	"Interated DevTools, one-command Supabase deployment, and the best model model for each tool.";

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
		<div className="flex flex-col">
			<div className="flex items-center justify-center flex-none -mt-10">
				<div className="container px-4">
					<PageHeader>
						<PageHeaderHeading className=" font-[family-name:var(--font-space-mono)]">
							{title}
						</PageHeaderHeading>
						<PageHeaderDescription>{description}</PageHeaderDescription>
						<PageActions>
							<Link href="/docs/mix/quickstart">
								<Button size="sm">Get Started</Button>
							</Link>
							<Link href="/docs/mix">
								<Button variant="outline" size="sm">
									View Docs
								</Button>
							</Link>
						</PageActions>
					</PageHeader>
				</div>
			</div>

			<DemoCards demos={homepageDemos} />

			<Footer />
		</div>
	);
}
