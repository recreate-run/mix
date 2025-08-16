import { Outlet, createRootRoute } from "@tanstack/react-router";
import { ThemeProvider } from "@/components/ui/theme-provider";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { TooltipProvider } from "@/components/ui/tooltip";
import { Link } from "@tanstack/react-router";
import { useState } from "react";

export const Route = createRootRoute({
	component: RootComponent,
	notFoundComponent: defaultNotFoundComponent,
	errorComponent: defaultErrorComponent,
});

const queryClient = new QueryClient();

function RootComponent() {
	return (
		<QueryClientProvider client={queryClient}>
			<ThemeProvider defaultTheme="dark" storageKey="vite-ui-theme">
				<TooltipProvider>
					<Outlet />
				</TooltipProvider>
			</ThemeProvider>
		</QueryClientProvider>
	);
}

function defaultNotFoundComponent() {
	return (
		<div>
			<p>Not found!</p>
			<Link to="/">Go home</Link>
		</div>
	);
}

interface ErrorComponentProps {
	error: Error;
	info?: { componentStack: string };
	reset: () => void;
}

function defaultErrorComponent({ error, info, reset }: ErrorComponentProps) {
	const [showError, setShowError] = useState(
		process.env.NODE_ENV === "development",
	);

	return (
		<div className="p-6 max-w-2xl mx-auto">
			<h2 className="text-2xl font-bold text-red-600 mb-4">
				Something went wrong!
			</h2>

			<div className="mb-4">
				<button
					onClick={() => setShowError(!showError)}
					className="px-4 py-2 bg-gray-200 hover:bg-gray-300 rounded mr-2"
				>
					{showError ? "Hide Error" : "Show Error"}
				</button>

				<button
					onClick={reset}
					className="px-4 py-2 bg-blue-500 hover:bg-blue-600 text-white rounded mr-2"
				>
					Try Again
				</button>

				<Link
					to="/"
					className="px-4 py-2 bg-gray-500 hover:bg-gray-600 text-white rounded inline-block"
				>
					Go Home
				</Link>
			</div>

			{showError && (
				<div className="bg-red-50 border border-red-200 rounded p-4">
					<h3 className="font-semibold text-red-800 mb-2">Error Details:</h3>
					<p className="text-red-700 mb-2">{error.message}</p>

					{info?.componentStack && (
						<details className="mt-2">
							<summary className="cursor-pointer text-red-600 font-medium">
								Component Stack Trace
							</summary>
							<pre className="mt-2 text-xs text-red-600 whitespace-pre-wrap overflow-x-auto">
								{info.componentStack}
							</pre>
						</details>
					)}
				</div>
			)}
		</div>
	);
}
