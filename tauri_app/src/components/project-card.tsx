import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Label } from "@/components/ui/label";
import { Folder } from "lucide-react";

interface Project {
	name: string;
	path: string;
	lastUsed: string;
}

interface ProjectCardProps {
	project: Project;
	onClick: (project: Project) => void;
}

export function ProjectCard({ project, onClick }: ProjectCardProps) {
	return (
		<Card
			className="cursor-pointer bg-neutral-200/60 dark:bg-neutral-700/60 border-none"
			onClick={() => onClick(project)}
		>
			<CardHeader>
				<CardTitle className=" flex items-center gap-3">
					<Folder className="size-5 mt-1" />
					{project.name}
				</CardTitle>
			</CardHeader>
			<CardContent>
				<p className="text-xs truncate mb-2">{project.path}</p>
				<Label className="text-xs text-muted-foreground">
					Last opened: {new Date(project.lastUsed).toLocaleDateString()}
				</Label>
			</CardContent>
		</Card>
	);
}
