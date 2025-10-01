import { blog } from "@/lib/source";
import type { Metadata } from "next";
import { notFound } from "next/navigation";
import Link from "next/link";
import { ChevronLeft } from "lucide-react";
import {
	PageRoot,
	PageArticle,
	PageTOC,
	PageTOCItems,
	PageTOCPopover,
	PageTOCPopoverContent,
	PageTOCPopoverItems,
	PageTOCPopoverTrigger,
	PageTOCTitle,
} from "fumadocs-ui/layouts/docs/page";
import { getMDXComponents } from "@/mdx-components";

interface Param {
	slug: string[];
}

interface Props {
	params: Promise<Param>;
}

export default async function BlogPostPage(props: Props) {
	const params = await props.params;
	const page = blog.getPage(params.slug);

	if (!page) notFound();

	const MDX = page.data.body;
	const toc = page.data.toc;

	return (
		<PageRoot
			toc={{
				toc,
				single: false,
			}}
		>
			{toc.length > 0 && (
				<PageTOCPopover>
					<PageTOCPopoverTrigger />
					<PageTOCPopoverContent>
						<PageTOCPopoverItems />
					</PageTOCPopoverContent>
				</PageTOCPopover>
			)}
			<PageArticle>
				<Link
					href="/blog"
					className="inline-flex items-center text-sm text-fd-muted-foreground hover:text-fd-foreground mb-8 transition-colors"
				>
					<ChevronLeft className="h-4 w-4 mr-1" />
					Back to Blog
				</Link>

				<article>
					<header className="mb-8 pb-8 border-b border-fd-border">
						<h1 className="text-3xl font-semibold">{page.data.title}</h1>
						<p className="text-lg text-fd-muted-foreground">
							{page.data.description}
						</p>
						<div className="flex items-center justify-between text-sm text-fd-muted-foreground mt-4">
							<span className="font-medium">{page.data.author as string}</span>
							<time dateTime={page.data.date as string}>
								{new Date(page.data.date as string).toLocaleDateString("en-US", {
									year: "numeric",
									month: "long",
									day: "numeric",
								})}
							</time>
						</div>
						{page.data.tags && Array.isArray(page.data.tags) && (
							<div className="flex flex-wrap gap-2 mt-4">
								{(page.data.tags as string[]).map((tag) => (
									<span
										key={tag}
										className="text-xs px-2 py-1 rounded-full bg-fd-accent text-fd-accent-foreground"
									>
										{tag}
									</span>
								))}
							</div>
						)}
					</header>

					<div className="prose flex-1 text-fd-foreground/80">
						<MDX components={getMDXComponents()} />
					</div>
				</article>
			</PageArticle>
			{toc.length > 0 && (
				<PageTOC>
					<PageTOCTitle />
					<PageTOCItems />
				</PageTOC>
			)}
		</PageRoot>
	);
}

export async function generateStaticParams(): Promise<Param[]> {
	return blog.getPages().map((page) => ({
		slug: page.slugs,
	}));
}

export async function generateMetadata(props: Props): Promise<Metadata> {
	const params = await props.params;
	const page = blog.getPage(params.slug);

	if (!page) notFound();

	return {
		title: page.data.title,
		description: page.data.description,
	};
}
