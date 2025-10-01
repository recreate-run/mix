import { blog } from "@/lib/source";
import type { Metadata } from "next";
import { notFound } from "next/navigation";
import Link from "next/link";
import { ChevronLeft } from "lucide-react";

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

	return (
		<main className="container mx-auto px-4 py-12 max-w-4xl">
			<Link
				href="/blog"
				className="inline-flex items-center text-sm text-fd-muted-foreground hover:text-fd-foreground mb-8 transition-colors"
			>
				<ChevronLeft className="h-4 w-4 mr-1" />
				Back to Blog
			</Link>

			<article>
				<header className="mb-8 pb-8 border-b border-fd-border">
					<h1 className="text-4xl font-bold mb-4">{page.data.title}</h1>
					<p className="text-lg text-fd-muted-foreground mb-4">
						{page.data.description}
					</p>
					<div className="flex items-center justify-between text-sm text-fd-muted-foreground">
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

				<div className="prose prose-fd max-w-none">
					<MDX />
				</div>
			</article>
		</main>
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
