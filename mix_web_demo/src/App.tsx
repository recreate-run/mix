import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ThemeProvider } from "@/components/ui/theme-provider";
import { TooltipProvider } from "@/components/ui/tooltip";
import { Toaster } from "sonner";
import { ChatInterface } from "@/components/chat-interface";
import { DemoBanner } from "@/components/demo-banner";
import { useDemoSession } from "@/hooks/useDemoSession";
import { LoadingScreen } from "@/components/loading/LoadingScreen";

const queryClient = new QueryClient();

function AppContent() {
	const { sessionId, isReady } = useDemoSession();

	if (!isReady || !sessionId) {
		return <LoadingScreen />;
	}

	return (
		<div className="flex h-screen flex-col overflow-hidden">
			<DemoBanner />
			<div className="flex-1 overflow-hidden">
				<ChatInterface sessionId={sessionId} />
			</div>
		</div>
	);
}

function App() {
	return (
		<QueryClientProvider client={queryClient}>
			<ThemeProvider defaultTheme="dark" storageKey="mix-web-demo-theme">
				<TooltipProvider>
					<AppContent />
					<Toaster position="top-right" />
				</TooltipProvider>
			</ThemeProvider>
		</QueryClientProvider>
	);
}

export default App;
