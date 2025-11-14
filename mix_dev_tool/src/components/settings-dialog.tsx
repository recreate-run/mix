import { IconLogin, IconLogout } from "@tabler/icons-react";
import { useQueryClient } from "@tanstack/react-query";
import { Eye, Loader2, Search, Settings } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { toast } from "sonner";
import { OAuthCodeDialog } from "@/components/oauth-code-dialog";
import { ProvidersLoadingSkeleton } from "@/components/provider-skeleton";
import { ToolCard } from "@/components/tool-card";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
	Dialog,
	DialogContent,
	DialogDescription,
	DialogHeader,
	DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import {
	authenticateWithApiKey,
	startOAuthFlow,
} from "@/handlers/login-command-handler";
import { logoutProvider } from "@/handlers/logout-command-handler";
import { useProviders } from "@/hooks/useProviders";
import { useToolsStatus } from "@/hooks/useTools";

interface SettingsDialogProps {
	open: boolean;
	onOpenChange: (open: boolean) => void;
}

export function SettingsDialog({ open, onOpenChange }: SettingsDialogProps) {
	const queryClient = useQueryClient();

	// Fetch providers with TanStack Query
	const {
		data: allProviders = [],
		isLoading: loadingProviders,
		refetch,
	} = useProviders();

	// Fetch tools status
	const { data: toolsStatus, isLoading: loadingTools } = useToolsStatus();

	// Login/logout state
	const [loggingOutProvider, setLoggingOutProvider] = useState<string | null>(
		null,
	);
	const [loginInProgress, setLoginInProgress] = useState<
		Record<string, boolean>
	>({});
	const [apiKeys, setApiKeys] = useState<Record<string, string>>({});
	const [selectedAuthMethod, setSelectedAuthMethod] = useState<
		Record<string, "api_key" | "oauth">
	>({});

	// OAuth code dialog state
	const [oauthCodeDialog, setOauthCodeDialog] = useState<{
		open: boolean;
		provider: string;
		oauthState: string;
	}>({
		open: false,
		provider: "",
		oauthState: "",
	});

	// Initialize auth method selection for unauthenticated providers
	const initializeAuthMethods = useCallback(() => {
		const newSelectedMethods: Record<string, "api_key" | "oauth"> = {};
		allProviders.forEach((provider) => {
			if (!provider.authenticated && provider.authMethods.length > 0) {
				newSelectedMethods[provider.id] = provider.authMethods[0];
			}
		});
		setSelectedAuthMethod((prev) => ({ ...prev, ...newSelectedMethods }));
	}, [allProviders]);

	// Initialize auth methods when providers change
	useEffect(() => {
		if (allProviders.length > 0) {
			initializeAuthMethods();
		}
	}, [allProviders, initializeAuthMethods]);

	const handleProviderLogout = async (providerId: string) => {
		if (loggingOutProvider) return;

		setLoggingOutProvider(providerId);

		try {
			const result = await logoutProvider(providerId);

			if (result.suppressChatMessage) {
				toast.success(
					`Successfully logged out from ${allProviders.find((p) => p.id === providerId)?.displayName || providerId}`,
				);
				// Invalidate and refetch provider data
				queryClient.invalidateQueries({
					queryKey: ["preferences", "providers"],
				});
				await refetch();
			} else {
				toast.error(result.content || "Logout failed");
			}
		} catch (error) {
			console.error("Logout failed:", error);
			toast.error("Logout failed. Please try again.");
		} finally {
			setLoggingOutProvider(null);
		}
	};

	const handleLogin = async (providerId: string) => {
		const provider = allProviders.find((p) => p.id === providerId);
		const authMethod =
			selectedAuthMethod[providerId] || provider?.authMethods[0];
		if (!authMethod) return;

		setLoginInProgress((prev) => ({ ...prev, [providerId]: true }));

		try {
			if (authMethod === "api_key") {
				const apiKey = apiKeys[providerId];
				if (!apiKey || apiKey.trim() === "") {
					toast.error("Please enter an API key");
					return;
				}

				const result = await authenticateWithApiKey(providerId, apiKey.trim());
				if (result.suppressChatMessage) {
					// Clear the API key after successful authentication
					setApiKeys((prev) => ({ ...prev, [providerId]: "" }));
					// Invalidate and refetch provider data
					queryClient.invalidateQueries({
						queryKey: ["preferences", "providers"],
					});
					await refetch();
				} else {
					toast.error(result.content || "Authentication failed");
				}
			} else if (authMethod === "oauth") {
				const result = await startOAuthFlow(providerId);
				if (result.loginData?.oauthState) {
					// OAuth flow started successfully - open the auth URL
					const authUrl = result.content.match(/https?:\/\/[^\s]+/)?.[0];
					if (authUrl) {
						try {
							window.open(authUrl, "_blank", "width=600,height=800");
							toast.info(
								"OAuth window opened. Please complete authentication.",
							);

							// Show OAuth code dialog
							setOauthCodeDialog({
								open: true,
								provider: providerId,
								oauthState: result.loginData.oauthState,
							});
						} catch (windowError) {
							console.error("Failed to open OAuth browser:", windowError);
							toast.error(
								"Failed to open OAuth browser. Please copy this URL manually: " +
									authUrl,
							);
						}
					}
				} else {
					toast.error(result.content || "Failed to start OAuth flow");
				}
			}
		} catch (error) {
			console.error("Login failed:", error);
			toast.error("Login failed. Please try again.");
		} finally {
			setLoginInProgress((prev) => ({ ...prev, [providerId]: false }));
		}
	};

	const handleAuthMethodChange = (
		providerId: string,
		method: "api_key" | "oauth",
	) => {
		setSelectedAuthMethod((prev) => ({ ...prev, [providerId]: method }));
		// Clear API key when switching away from API key method
		if (method !== "api_key") {
			setApiKeys((prev) => ({ ...prev, [providerId]: "" }));
		}
	};

	const handleApiKeyChange = (providerId: string, value: string) => {
		setApiKeys((prev) => ({ ...prev, [providerId]: value }));
	};

	return (
		<>
			<Dialog onOpenChange={onOpenChange} open={open}>
				<DialogContent className="max-h-[80vh] max-w-2xl overflow-hidden">
					<DialogHeader>
						<DialogTitle className="flex items-center gap-2">
							<Settings className="h-5 w-5" />
							Settings
						</DialogTitle>
						<DialogDescription className="text-xs">
							Manage your providers, tools, and authentication settings.
						</DialogDescription>
					</DialogHeader>

					<div className="max-h-[60vh] min-h-[500px] space-y-5 overflow-y-auto">
						{/* Providers Section */}
						<Card>
							<CardHeader className="pb-3">
								<CardTitle className="text-base">AI Providers</CardTitle>
							</CardHeader>
							<CardContent>
								{loadingProviders ? (
									<ProvidersLoadingSkeleton />
								) : allProviders.length === 0 ? (
									<div className="py-8 text-center">
										<p className="text-muted-foreground text-sm">
											No providers available.
										</p>
									</div>
								) : (
									<div className="divide-y">
										{allProviders.map((provider) => (
											<div className="py-3 first:pt-0" key={provider.id}>
												<div className="flex items-center justify-between">
													<div>
														<div className="flex items-center gap-2">
															<p className="font-medium text-sm">
																{provider.displayName}
															</p>
															{provider.authenticated && (
																<>
																	<span className="flex items-center gap-1 text-[11px] text-green-600">
																		<span className="h-1.5 w-1.5 rounded-full bg-green-600" />
																		Authenticated
																	</span>
																	{provider.isPreferred && (
																		<span className="text-[11px] text-muted-foreground">
																			· Preferred
																		</span>
																	)}
																</>
															)}
														</div>
													</div>

													{provider.authenticated ? (
														<Button
															disabled={!!loggingOutProvider}
															onClick={() => handleProviderLogout(provider.id)}
															size="sm"
															variant="ghost"
															className="text-muted-foreground hover:text-foreground"
														>
															{loggingOutProvider === provider.id ? (
																<Loader2 className="h-4 w-4 animate-spin mr-1.5" />
															) : (
																<IconLogout className="h-4 w-4 mr-1.5" />
															)}
															Logout
														</Button>
													) : null}
												</div>

												{!provider.authenticated && (
													<div className="mt-2.5 space-y-2.5">
														{/* Authentication method selection */}
														{provider.authMethods.length > 1 && (
															<RadioGroup
																className="flex gap-6"
																onValueChange={(value: string) =>
																	handleAuthMethodChange(
																		provider.id,
																		value as "api_key" | "oauth",
																	)
																}
																value={
																	selectedAuthMethod[provider.id] ||
																	provider.authMethods[0]
																}
															>
																{provider.authMethods.includes("api_key") && (
																	<div className="flex items-center space-x-2">
																		<RadioGroupItem
																			id={`${provider.id}-api-key`}
																			value="api_key"
																		/>
																		<Label
																			className="text-sm"
																			htmlFor={`${provider.id}-api-key`}
																		>
																			API Key
																		</Label>
																	</div>
																)}
																{provider.authMethods.includes("oauth") && (
																	<div className="flex items-center space-x-2">
																		<RadioGroupItem
																			id={`${provider.id}-oauth`}
																			value="oauth"
																		/>
																		<Label
																			className="text-sm"
																			htmlFor={`${provider.id}-oauth`}
																		>
																			OAuth
																		</Label>
																	</div>
																)}
															</RadioGroup>
														)}

														{/* API Key input or OAuth button */}
														<div>
															{(selectedAuthMethod[provider.id] ||
																provider.authMethods[0]) === "api_key" ? (
																<div className="flex gap-2">
																	<Input
																		className="flex-1"
																		disabled={loginInProgress[provider.id]}
																		id={`${provider.id}-key`}
																		onChange={(e) =>
																			handleApiKeyChange(
																				provider.id,
																				e.target.value,
																			)
																		}
																		placeholder={
																			provider.apiKeyFormat ||
																			"Enter your API key..."
																		}
																		type="password"
																		value={apiKeys[provider.id] || ""}
																	/>
																	<Button
																		disabled={
																			loginInProgress[provider.id] ||
																			!apiKeys[provider.id]?.trim()
																		}
																		onClick={() => handleLogin(provider.id)}
																		size="sm"
																	>
																		{loginInProgress[provider.id] ? (
																			<Loader2 className="h-4 w-4 animate-spin mr-1.5" />
																		) : (
																			<IconLogin className="h-4 w-4 mr-1.5" />
																		)}
																		Login
																	</Button>
																</div>
															) : (
																<Button
																	disabled={loginInProgress[provider.id]}
																	onClick={() => handleLogin(provider.id)}
																	size="sm"
																>
																	{loginInProgress[provider.id] ? (
																		<Loader2 className="h-4 w-4 animate-spin mr-1.5" />
																	) : (
																		<IconLogin className="h-4 w-4 mr-1.5" />
																	)}
																	Connect with OAuth
																</Button>
															)}
														</div>
													</div>
												)}
											</div>
										))}
									</div>
								)}
							</CardContent>
						</Card>

						{/* Tools & Agents Section */}
						<div className="space-y-3">
							<h3 className="font-semibold text-base">Tools & Subagents</h3>

							{!loadingTools &&
								toolsStatus?.categories &&
								Object.entries(toolsStatus.categories).map(
									([categoryId, category]) => {
										// Map category IDs to icons
										const categoryIcons: Record<string, React.ReactNode> = {
											web_search: <Search className="h-5 w-5" />,
											multimodal_analyzer: <Eye className="h-5 w-5" />,
										};

										return (
											<ToolCard
												categoryDisplayName={category.displayName}
												icon={
													categoryIcons[categoryId] || (
														<Settings className="h-5 w-5" />
													)
												}
												key={categoryId}
												tools={category.tools}
											/>
										);
									},
								)}
						</div>
					</div>
				</DialogContent>
			</Dialog>

			{/* OAuth Code Dialog */}
			<OAuthCodeDialog
				oauthState={oauthCodeDialog.oauthState}
				onOpenChange={(open) =>
					setOauthCodeDialog((prev) => ({ ...prev, open }))
				}
				onSuccess={() => {
					// Refresh providers data after successful authentication
					refetch();
					// Clear login in progress state
					setLoginInProgress((prev) => ({
						...prev,
						[oauthCodeDialog.provider]: false,
					}));
				}}
				open={oauthCodeDialog.open}
				provider={oauthCodeDialog.provider}
			/>
		</>
	);
}
