import { ArrowRightIcon } from "lucide-react";

import { Badge } from "@/components/ui/badge";

export function Announcement({ content }: { content: string }) {
	return (
		<Badge variant="secondary" className="rounded-full">
			{content} <ArrowRightIcon />
		</Badge>
	);
}
