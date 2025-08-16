import "./App.css";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useEffect } from "react";
import { ChatApp } from "@/components/chat-app";
import { ThemeProvider } from "@/components/ui/theme-provider";
import { fetchAppList } from "@/hooks/useOpenApps";

const queryClient = new QueryClient();

const App = () => {
	useEffect(() => {
		queryClient.prefetchQuery({
			queryKey: ["appList"],
			queryFn: fetchAppList,
		});
	}, []);

	return (
		<QueryClientProvider client={queryClient}>
			<ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
				<ChatApp sessionId="default" />
			</ThemeProvider>
		</QueryClientProvider>
	);
};
export default App;
