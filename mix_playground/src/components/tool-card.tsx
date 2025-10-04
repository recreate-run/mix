import { Loader2, Trash2 } from "lucide-react";
import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
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
			<CardHeader>
				<div className="flex items-center gap-2">
					{icon}
					<CardTitle>{categoryDisplayName}</CardTitle>
				</div>
			</CardHeader>
			<CardContent className="space-y-4">
				{tools.length > 0 ? (
					tools.map((tool) => (
						<div className="rounded-lg border p-4" key={tool.provider}>
							<div className="flex items-start justify-between">
								<div>
									<p className="font-medium">{tool.displayName}</p>
									<div className="mt-2 flex items-center gap-2">
										{tool.authenticated ? (
											<Badge
												className="border-green-600 text-green-600 text-xs"
												variant="outline"
											>
												✓ Configured
											</Badge>
										) : (
											<Badge
												className="text-muted-foreground text-xs"
												variant="outline"
											>
												Requires setup
											</Badge>
										)}
									</div>
								</div>

								{tool.authenticated && (
									<Button
										className="flex items-center gap-2"
										disabled={deleteCredentials.isPending}
										onClick={() => handleDeleteCredentials(tool.provider)}
										size="sm"
										variant="outline"
									>
										{deleteCredentials.isPending ? (
											<Loader2 className="h-4 w-4 animate-spin" />
										) : (
											<Trash2 className="h-4 w-4" />
										)}
										Remove
									</Button>
								)}
							</div>

							{!tool.authenticated && tool.apiKeyRequired && (
								<div className="mt-4 space-y-3">
									<div className="space-y-2">
										<Label
											className="font-medium text-sm"
											htmlFor={`${tool.provider}-api-key`}
										>
											API Key
										</Label>
										<div className="flex gap-2">
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
												className="flex items-center gap-2"
												disabled={
													storeCredentials.isPending ||
													!apiKeys[tool.provider]?.trim()
												}
												onClick={() => handleStoreApiKey(tool.provider)}
												size="sm"
											>
												{storeCredentials.isPending ? (
													<Loader2 className="h-4 w-4 animate-spin" />
												) : (
													"Configure"
												)}
											</Button>
										</div>
									</div>
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
