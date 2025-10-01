import Link from "next/link";
import { blog } from "@/lib/source";

export default function BlogPage() {
	const posts = [...blog.getPages()].sort((a, b) => {
		const dateA = new Date(a.data.date as string);
		const dateB = new Date(b.data.date as string);
		return dateB.getTime() - dateA.getTime();
	});

	return (
		<main className="container mx-auto px-4 py-12 max-w-6xl">
			<div className="mb-12">
				<h1 className="text-4xl font-bold mb-4">Blog</h1>
				<p className="text-lg text-fd-muted-foreground">
					Latest updates, tutorials, and insights from the Mix team
				</p>
			</div>

			<div className="grid gap-8 md:grid-cols-2 lg:grid-cols-3">
				{posts.map((post) => (
					<Link
						key={post.url}
						href={post.url}
						className="group block rounded-lg border border-fd-border bg-fd-card p-6 transition-all hover:shadow-lg hover:border-fd-primary"
					>
						<div className="mb-4">
							<h2 className="text-xl font-semibold mb-2 group-hover:text-fd-primary transition-colors">
								{post.data.title}
							</h2>
							<p className="text-sm text-fd-muted-foreground mb-3">
								{post.data.description}
							</p>
						</div>

						<div className="flex items-center justify-between text-sm text-fd-muted-foreground">
							<span>{post.data.author as string}</span>
							<time dateTime={post.data.date as string}>
								{new Date(post.data.date as string).toLocaleDateString(
									"en-US",
									{
										year: "numeric",
										month: "long",
										day: "numeric",
									},
								)}
							</time>
						</div>

						{post.data.tags && Array.isArray(post.data.tags) && (
							<div className="flex flex-wrap gap-2 mt-4">
								{(post.data.tags as string[]).map((tag) => (
									<span
										key={tag}
										className="text-xs px-2 py-1 rounded-full bg-fd-accent text-fd-accent-foreground"
									>
										{tag}
									</span>
								))}
							</div>
						)}
					</Link>
				))}
			</div>
		</main>
	);
}
