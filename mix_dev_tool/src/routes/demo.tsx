import { createFileRoute } from "@tanstack/react-router";
import "@/styles/App.css";
import { useEffect, useState } from "react";
import { ChatApp } from "@/components/chat-app";
import { DemoBanner } from "@/components/demo-banner";
import { useCreateSession } from "@/hooks/useSession";

export const Route = createFileRoute("/demo")({
	component: DemoApp,
});

const DEMO_SESSION_KEY = "mix-demo-session-id";

function DemoApp() {
	const [sessionId, setSessionId] = useState<string | null>(null);
	const [isReady, setIsReady] = useState(false);
	const createSession = useCreateSession();

	useEffect(() => {
		const initSession = async () => {
			// Try to use existing demo session
			const existingSessionId = localStorage.getItem(DEMO_SESSION_KEY);
			if (existingSessionId) {
				// Validate session exists - if not, create new one
				try {
					// Just try to use it - ChatApp will handle if it doesn't exist
					setSessionId(existingSessionId);
					setIsReady(true);
					return;
				} catch (_error) {
					console.log("Demo session not found, creating new one");
					localStorage.removeItem(DEMO_SESSION_KEY);
				}
			}

			// Create new demo session
			try {
				const newSession = await createSession.mutateAsync({
					title: `Demo Session - ${new Date().toLocaleDateString()}`,
				});
				setSessionId(newSession.id);
				localStorage.setItem(DEMO_SESSION_KEY, newSession.id);
				setIsReady(true);
			} catch (error) {
				console.error("Failed to create demo session:", error);
			}
		};

		initSession();
	}, [createSession]);

	if (!isReady || !sessionId) {
		return (
			<div className="flex h-screen w-full items-center justify-center bg-background text-foreground">
				<div className="text-center">
					<div className="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-4 border-blue-500 border-t-transparent" />
					<p className="text-muted-foreground">Loading demo...</p>
				</div>
			</div>
		);
	}

	return (
		<div className="flex h-screen w-full flex-col bg-background text-foreground">
			<DemoBanner />
			<div className="relative flex min-h-0 flex-1">
				<ChatApp sessionId={sessionId} />
			</div>
		</div>
	);
}
