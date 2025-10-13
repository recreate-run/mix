import { useEffect, useState } from "react";
import { useCreateSession } from "./useSession";
import { mix } from "@/lib/mix-sdk";

const DEMO_SESSION_KEY = "mix-demo-session-id";

export function useDemoSession() {
	const [sessionId, setSessionId] = useState<string | null>(null);
	const [isReady, setIsReady] = useState(false);
	const createSession = useCreateSession();

	useEffect(() => {
		const initSession = async () => {
			// Try to restore from localStorage
			const existingSessionId = localStorage.getItem(DEMO_SESSION_KEY);

			if (existingSessionId) {
				try {
					// Validate session still exists
					const session = await mix.sessions.get({ id: existingSessionId });
					if (session) {
						setSessionId(existingSessionId);
						setIsReady(true);
						return;
					}
				} catch (error) {
					// Session doesn't exist, create new one
					console.log("Previous session not found, creating new one");
					localStorage.removeItem(DEMO_SESSION_KEY);
				}
			}

			// Create new session
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

	return { sessionId, isReady };
}
