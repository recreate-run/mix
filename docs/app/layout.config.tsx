import type { BaseLayoutProps } from "fumadocs-ui/layouts/shared";
import { Github } from "lucide-react";
import { config } from "@/lib/config";

/**
 * Shared layout configurations
 *
 * you can customise layouts individually from:
 * Home Layout: app/(home)/layout.tsx
 * Docs Layout: app/docs/layout.tsx
 */
export const baseOptions: BaseLayoutProps = {
	nav: {
		title: (
			<>
				<img
					src="/favicon.ico"
					width="24"
					height="24"
					alt="Mix Logo"
					className="inline-block"
				/>
				Mix
			</>
		),
		children: (
			<a
				href={config.links.github}
				target="_blank"
				rel="noopener noreferrer"
				className="ml-4 inline-flex items-center justify-center rounded-md p-2 text-sm font-medium text-fd-foreground hover:bg-fd-accent hover:text-fd-accent-foreground transition-colors"
				aria-label="View source on GitHub"
			>
				<Github className="h-4 w-4" />
			</a>
		),
	},
};
