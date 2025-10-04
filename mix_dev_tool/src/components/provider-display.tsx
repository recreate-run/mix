import { CheckCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { handleProviderSelection } from "@/handlers/provider-command-handler";
import type { ProviderDisplayProps } from "@/types/provider";

export function ProviderDisplay({ data, onUpdate }: ProviderDisplayProps) {
	const handleSelect = async (providerId: string) => {
		// Only allow selecting authenticated providers
		const provider = data.providers.find((p) => p.id === providerId);
		if (!provider?.authenticated) {
			onUpdate({
				content: `❌ Cannot select provider ${provider?.displayName || providerId} because it is not authenticated. Please use the \`/login\` command first.`,
				from: "assistant",
				frontend_only: true,
			});
			return;
		}

		// Call the provider selection handler
		const result = await handleProviderSelection(providerId);
		onUpdate(result);
	};

	return (
		<Card>
			<CardContent className="p-4">
				<div className="space-y-4">
					<h3 className="mb-2 font-medium text-lg">Select Provider</h3>
					<div className="grid grid-cols-1 gap-2">
						{data.providers.map((provider) => (
							<Button
								className={`w-full justify-between ${provider.isPreferred ? "bg-primary text-primary-foreground hover:bg-primary/90" : provider.authenticated ? "" : "opacity-50"}`}
								disabled={!provider.authenticated}
								key={provider.id}
								onClick={() => handleSelect(provider.id)}
								variant={provider.isPreferred ? "default" : "outline"}
							>
								<div className="flex items-center gap-2">
									<span>{provider.displayName}</span>
									{provider.authenticated && (
										<span className="rounded-full bg-green-100 px-2 py-0.5 text-green-800 text-xs dark:bg-green-800/20 dark:text-green-400">
											{provider.authMethod === "oauth" ? "OAuth" : "API Key"}
										</span>
									)}
								</div>
								{provider.isPreferred && <CheckCircle className="h-4 w-4" />}
							</Button>
						))}
					</div>

					{/* Help text */}
					<p className="mt-4 text-muted-foreground text-sm">
						{data.providers.some((p) => !p.authenticated)
							? "Some providers are not authenticated. Use the `/login` command to authenticate with them."
							: "All available providers are authenticated."}
					</p>
				</div>
			</CardContent>
		</Card>
	);
}
