import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { open } from "@tauri-apps/plugin-dialog";
import { Plus } from "lucide-react";
import { useEffect, useState } from "react";
import {
	IntroHeader,
	IntroHeaderDescription,
	IntroHeaderHeading,
} from "@/components/intro-header";
import { ProjectCard } from "@/components/project-card";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Card, CardContent } from "@/components/ui/card";
import { useCreateSession } from "@/hooks/useSession";
import { type SessionData, useSessionsList } from "@/hooks/useSessionsList";
import { normalizePath } from "@/utils/pathUtils";

export const Route = createFileRoute("/")({
	component: ProjectSelector,
});

interface Project {
	name: string;
	path: string;
	lastUsed: string;
}

function ProjectSelector() {
	const navigate = useNavigate();
	const createSession = useCreateSession();
	const { data: sessions = [] } = useSessionsList();
	const [projects, setProjects] = useState<Project[]>([]);
	const [error, setError] = useState<string | null>(null);

	// Helper function to find existing session by working directory
	const findExistingSession = async (
		workingDirectory: string,
	): Promise<SessionData | null> => {
		const normalizedInputPath = await normalizePath(workingDirectory);

		for (const session of sessions) {
			if (session.workingDirectory) {
				const normalizedSessionPath = await normalizePath(
					session.workingDirectory,
				);
				if (normalizedSessionPath === normalizedInputPath) {
					return session;
				}
			}
		}

		return null;
	};

	// Load projects from localStorage
	useEffect(() => {
		const stored = localStorage.getItem("mix-projects");
		if (stored) {
			setProjects(JSON.parse(stored));
		}
	}, []);

	const addProject = async (project: Project) => {
		const normalizedProjectPath = await normalizePath(project.path);

		// Filter out any existing projects with the same normalized path
		const filteredProjects = [];
		for (const p of projects) {
			const normalizedExistingPath = await normalizePath(p.path);
			if (normalizedExistingPath !== normalizedProjectPath) {
				filteredProjects.push(p);
			}
		}

		const updated = [project, ...filteredProjects].slice(0, 6);
		setProjects(updated);
		localStorage.setItem("mix-projects", JSON.stringify(updated));
	};

	// Unified function to open any project by path and name
	const openProject = async (projectPath: string, projectName: string) => {
		setError(null);

		try {
			const existingSession = await findExistingSession(projectPath);

			if (existingSession) {
				navigate({
					to: "/$sessionId",
					params: { sessionId: existingSession.id },
					replace: true,
				});
			} else {
				const normalizedProjectPath = await normalizePath(projectPath);
				const newSession = await createSession.mutateAsync({
					title: projectName,
					workingDirectory: normalizedProjectPath,
				});

				navigate({
					to: "/$sessionId",
					params: { sessionId: newSession.id },
					replace: true,
				});
			}
		} catch (error) {
			setError(
				`Failed to open project "${projectName}": ${error instanceof Error ? error.message : "Unknown error"}`,
			);
		}
	};

	const handleNewProject = async () => {
		const folder = await open({
			directory: true,
			multiple: false,
		});
		if (!folder || typeof folder !== "string") return;

		const projectName = folder.split("/").pop() || "Untitled Project";
		const project: Project = {
			name: projectName,
			path: folder,
			lastUsed: new Date().toISOString(),
		};

		await addProject(project);
		await openProject(folder, projectName);
	};

	const handleOpenProject = async (project: Project) => {
		const updatedProject = { ...project, lastUsed: new Date().toISOString() };
		await addProject(updatedProject);
		await openProject(project.path, project.name);
	};

	return (
		<section className="flex min-h-screen flex-col items-center justify-center p-8 ">
			<IntroHeader className="mb-16 max-w-4xl ">
				<IntroHeaderHeading className="max-w-4xl">Mix</IntroHeaderHeading>
				<IntroHeaderDescription>
					Select a project to get started
				</IntroHeaderDescription>
			</IntroHeader>

			{error && (
				<Alert className="mx-auto mb-6 max-w-4xl" variant="destructive">
					<AlertDescription>{error}</AlertDescription>
				</Alert>
			)}

			<div className="mx-auto grid max-w-4xl grid-flow-col justify-items-center gap-4">
				{/* New Project Card */}
				<Card
					className="cursor-pointer border-2 bg-transparent transition-shadow hover:shadow-md"
					onClick={handleNewProject}
				>
					<CardContent className="min-h-[120px] grid  place-items-center p-6">
						<Plus className="mb-2 h-8 w-8 text-muted-foreground" />
						<p className="font-medium text-xl">Select project folder</p>
					</CardContent>
				</Card>

				{/* Recent Projects */}
				{projects.map((project) => (
					<ProjectCard
						key={project.path}
						onClick={handleOpenProject}
						project={project}
					/>
				))}
			</div>
		</section>
	);
}
