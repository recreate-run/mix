import { Folder } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card';
import { Label } from '@/components/ui/label';

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
      className="cursor-pointer border-none bg-neutral-200/60 dark:bg-neutral-900/20"
      onClick={() => onClick(project)}
    >
      <CardHeader>
        <CardTitle className=" flex items-center gap-3">
          <Folder className="mt-1 size-5" />
          {project.name}
        </CardTitle>
      </CardHeader>
      <CardContent>
        <p className="mb-2 truncate text-xs">{project.path}</p>
        <Label className="text-muted-foreground text-xs">
          Last opened: {new Date(project.lastUsed).toLocaleDateString()}
        </Label>
      </CardContent>
    </Card>
  );
}
