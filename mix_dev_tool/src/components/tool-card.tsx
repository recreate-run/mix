import { Loader2, Trash2 } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useDeleteCredentials, useStoreCredentials } from "@/hooks/useTools";

interface ToolInfo {
	provider: string;
	displayName: string;
	authenticated: boolean;
	apiKeyRequired: boolean;
}

interface ToolCardProps {
	categoryDisplayName: string;
	icon: React.ReactNode;
	tools: ToolInfo[];
}

export function ToolCard({ categoryDisplayName, icon, tools }: ToolCardProps) {
	const storeCredentials = useStoreCredentials();
	const deleteCredentials = useDeleteCredentials();

	const [apiKeys, setApiKeys] = useState<Record<string, string>>({});

	const handleApiKeyChange = (provider: string, value: string) => {
		setApiKeys((prev) => ({ ...prev, [provider]: value }));
	};

	const handleStoreApiKey = async (provider: string) => {
		const apiKey = apiKeys[provider]?.trim();
		if (!apiKey) return;

		await storeCredentials.mutateAsync({
			provider,
			api_key: apiKey,
		});

		// Clear the input after successful storage
		setApiKeys((prev) => ({ ...prev, [provider]: "" }));
	};

	const handleDeleteCredentials = async (provider: string) => {
		await deleteCredentials.mutateAsync({
			provider,
		});
	};

	return (
		<Card>
			<CardHeader className="pb-3">
				<div className="flex items-center gap-2">
					{icon}
					<CardTitle className="text-base">{categoryDisplayName}</CardTitle>
				</div>
			</CardHeader>
			<CardContent className="divide-y">
				{tools.length > 0 ? (
					tools.map((tool) => (
						<div className="py-3 first:pt-0" key={tool.provider}>
							<div className="flex items-start justify-between">
								<div>
									<div className="flex items-center gap-2">
										<p className="font-medium text-sm">{tool.displayName}</p>
										{tool.authenticated && (
											<span className="flex items-center gap-1 text-[11px] text-green-600">
												<span className="h-1.5 w-1.5 rounded-full bg-green-600" />
												Configured
											</span>
										)}
									</div>
								</div>

								{tool.authenticated && (
									<Button
										disabled={deleteCredentials.isPending}
										onClick={() => handleDeleteCredentials(tool.provider)}
										size="sm"
										variant="ghost"
										className="text-muted-foreground hover:text-foreground"
									>
										{deleteCredentials.isPending ? (
											<Loader2 className="h-4 w-4 animate-spin mr-1.5" />
										) : (
											<Trash2 className="h-4 w-4 mr-1.5" />
										)}
										Remove
									</Button>
								)}
							</div>

							{!tool.authenticated && tool.apiKeyRequired && (
								<div className="mt-2.5 flex gap-2">
									<Input
										className="flex-1"
										disabled={storeCredentials.isPending}
										id={`${tool.provider}-api-key`}
										onChange={(e) =>
											handleApiKeyChange(tool.provider, e.target.value)
										}
										placeholder={`Enter your ${tool.displayName} API key...`}
										type="password"
										value={apiKeys[tool.provider] || ""}
									/>
									<Button
										disabled={
											storeCredentials.isPending ||
											!apiKeys[tool.provider]?.trim()
										}
										onClick={() => handleStoreApiKey(tool.provider)}
										size="sm"
									>
										{storeCredentials.isPending ? (
											<Loader2 className="h-4 w-4 animate-spin mr-1.5" />
										) : (
											"Configure"
										)}
									</Button>
								</div>
							)}
						</div>
					))
				) : (
					<div className="py-8 text-center">
						<p className="text-muted-foreground text-sm">
							No tools available in this category.
						</p>
					</div>
				)}
			</CardContent>
		</Card>
	);
}
