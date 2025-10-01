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
			<div className="flex flex-col">
				<div className="flex items-center justify-center py-4 flex-none min-h-screen">
					<div className="container px-4 pb-8">
						<PageHeader className="mb-6">
							<Announcement />
							<PageHeaderHeading className="max-w-4xl text-3xl md:text-4xl">{title}</PageHeaderHeading>
							<PageHeaderDescription className="mb-4 text-sm">{description}</PageHeaderDescription>
						</PageHeader>
					</div>
				</div>

				<Footer />
			</div>
		</>
	);
}