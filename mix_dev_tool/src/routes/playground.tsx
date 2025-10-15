import { createFileRoute } from "@tanstack/react-router";
import "@/styles/App.css";
import { useEffect, useState } from "react";
import { toast } from "sonner";
import { ChatApp } from "@/components/chat-app";
import { PlaygroundWelcome } from "@/components/playground-welcome";
import { usePersistentSSE } from "@/hooks/usePersistentSSE";
import { useCreateSession } from "@/hooks/useSession";
import { useSessionMessages } from "@/hooks/useSessionMessages";
import { useBoundStore } from "@/stores";

export const Route = createFileRoute("/playground")({
	component: PlaygroundApp,
});

const PLAYGROUND_SESSION_KEY = "mix-playground-session-id";

function PlaygroundApp() {
	const [sessionId, setSessionId] = useState<string | null>(null);
	const [isReady, setIsReady] = useState(false);
	const createSession = useCreateSession();
	const sessionMessages = useSessionMessages(sessionId || "");
	const sseStream = usePersistentSSE(sessionId || "");

	// Get attachments from store
	const attachments = useBoundStore((state) => state.attachments);
	const referenceMap = useBoundStore((state) => state.referenceMap);

	const messages = sessionMessages.data || [];
	const hasMessages =
		messages.length > 0 ||
		sseStream.processing ||
		sseStream.pendingUserMessage !== null;

	// Show error toasts for streaming errors
	useEffect(() => {
		if (
			sseStream.error &&
			!sseStream.error.includes("cancelled") &&
			!sseStream.cancelled
		) {
			toast.error(`Failed to send message: ${sseStream.error}`);
		}
	}, [sseStream.error, sseStream.cancelled]);

	// biome-ignore lint/correctness/useExhaustiveDependencies: Run only once on mount to prevent multiple session creation
	useEffect(() => {
		// Prevent multiple session creations
		if (sessionId || isReady) return;

		const initSession = async () => {
			// Try to use existing playground session
			const existingSessionId = localStorage.getItem(PLAYGROUND_SESSION_KEY);
			if (existingSessionId) {
				// Validate session exists - if not, create new one
				try {
					// Just try to use it - ChatApp will handle if it doesn't exist
					setSessionId(existingSessionId);
					setIsReady(true);
					return;
				} catch (_error) {
					console.log("Playground session not found, creating new one");
					localStorage.removeItem(PLAYGROUND_SESSION_KEY);
				}
			}

			// Create new playground session
			try {
				const newSession = await createSession.mutateAsync({
					title: `Playground Session - ${new Date().toLocaleDateString()}`,
				});
				setSessionId(newSession.id);
				localStorage.setItem(PLAYGROUND_SESSION_KEY, newSession.id);
				setIsReady(true);
			} catch (error) {
				console.error("Failed to create playground session:", error);
			}
		};

		initSession();
	}, []); // Run only once on mount

	const handleSubmit = (text: string) => {
		if (sessionId && sseStream.connected) {
			sseStream.submitMessage({
				text,
				attachments,
				referenceMap,
				planMode: false,
			});
		}
	};

	const handleClear = async () => {
		try {
			// Clear streaming state first if needed
			if (sseStream.processing) {
				await sseStream.cancelMessage();
			}

			// Create new session
			const newSession = await createSession.mutateAsync({
				title: `Playground Session - ${new Date().toLocaleDateString()}`,
			});

			// Update localStorage with new session ID
			localStorage.setItem(PLAYGROUND_SESSION_KEY, newSession.id);

			// Clear any SSE streaming state
			sseStream.clearStreamingContent();

			// Update state to new session (this will trigger re-render and reconnect SSE)
			setSessionId(newSession.id);

			toast.success("Playground cleared - starting fresh!");
		} catch (error) {
			console.error("Failed to clear playground:", error);
			toast.error("Failed to clear playground");
		}
	};

	if (!isReady || !sessionId) {
		return (
			<div className="flex h-screen w-full items-center justify-center bg-background text-foreground">
				<div className="text-center">
					<div className="mx-auto mb-4 h-8 w-8 animate-spin rounded-full border-4 border-blue-500 border-t-transparent" />
					<p className="text-muted-foreground">Loading playground...</p>
				</div>
			</div>
		);
	}

	return (
		<div className="flex h-screen w-full flex-col bg-background text-foreground">
			{!hasMessages ? (
				<PlaygroundWelcome
					sessionId={sessionId}
					onSubmit={handleSubmit}
					onClear={handleClear}
				/>
			) : (
				<ChatApp sessionId={sessionId} onClear={handleClear} isPlayground />
			)}
		</div>
	);
}
