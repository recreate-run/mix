import * as React from "react";
import { Plus, Trash2, Loader2, Settings } from "lucide-react";
import {
	Sidebar,
	SidebarContent,
	SidebarGroup,
	SidebarGroupContent,
	SidebarGroupLabel,
	SidebarRail,
} from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import {
	Select,
	SelectContent,
	SelectItem,
	SelectTrigger,
	SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Separator } from "@/components/ui/separator";
import {
	Popover,
	PopoverContent,
	PopoverTrigger,
} from "@/components/ui/popover";
import {
	useSessionCallbacks,
	useUpdateCallbacks,
} from "@/hooks/useSessionCallbacks";
import type { Callback } from "mix-typescript-sdk/models/callback.js";
import { CallbackType } from "mix-typescript-sdk/models/callback.js";

interface RightSidebarProps extends React.ComponentProps<typeof Sidebar> {
	sessionId?: string;
}

export function RightSidebar({ sessionId, ...props }: RightSidebarProps) {
	const [isAddingCallback, setIsAddingCallback] = React.useState(false);
	const { data, isLoading } = useSessionCallbacks(sessionId || "");
	const updateCallbacks = useUpdateCallbacks();

	const callbacks = data?.callbacks || [];
	const currentSessionId = data?.sessionId || sessionId || "";

	const handleDeleteCallback = (index: number) => {
		const newCallbacks = callbacks.filter((_, i) => i !== index);
		updateCallbacks.mutate({
			sessionId: currentSessionId,
			callbacks: newCallbacks,
		});
	};

	const handleAddCallback = (callback: Callback) => {
		const newCallbacks = [...callbacks, callback];
		updateCallbacks.mutate({
			sessionId: currentSessionId,
			callbacks: newCallbacks,
		});
		setIsAddingCallback(false);
	};

	if (!sessionId) {
		return (
			<Sidebar {...props}>
				<SidebarContent>
					<SidebarGroup>
						<SidebarGroupLabel>Session Callbacks</SidebarGroupLabel>
						<SidebarGroupContent>
							<p className="text-sm text-muted-foreground p-4">
								No session selected
							</p>
						</SidebarGroupContent>
					</SidebarGroup>
				</SidebarContent>
				<SidebarRail />
			</Sidebar>
		);
	}

	return (
		<Sidebar {...props}>
			<SidebarContent>
				<SidebarGroup>
					<SidebarGroupLabel>Session Callbacks</SidebarGroupLabel>
					<SidebarGroupContent className="space-y-3 p-4">
						{isLoading && (
							<div className="flex items-center justify-center py-8">
								<Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
							</div>
						)}

						{!isLoading && callbacks.length === 0 && !isAddingCallback && (
							<div className="text-center py-8">
								<p className="text-sm text-muted-foreground mb-4">
									No callbacks configured
								</p>
								<Button
									onClick={() => setIsAddingCallback(true)}
									size="sm"
									variant="outline"
								>
									<Plus className="h-4 w-4 mr-2" />
									Add Callback
								</Button>
							</div>
						)}

						{!isLoading && callbacks.length > 0 && (
							<>
								<div className="space-y-2">
									{callbacks.map((callback, index) => (
										<CallbackCard
											key={index}
											callback={callback}
											onDelete={() => handleDeleteCallback(index)}
										/>
									))}
								</div>

								{!isAddingCallback && (
									<Button
										onClick={() => setIsAddingCallback(true)}
										size="sm"
										variant="outline"
										className="w-full"
									>
										<Plus className="h-4 w-4 mr-2" />
										Add Callback
									</Button>
								)}
							</>
						)}

						{isAddingCallback && (
							<>
								<Separator />
								<CallbackForm
									onSubmit={handleAddCallback}
									onCancel={() => setIsAddingCallback(false)}
								/>
							</>
						)}
					</SidebarGroupContent>
				</SidebarGroup>
			</SidebarContent>
			<SidebarRail />
		</Sidebar>
	);
}

interface CallbackCardProps {
	callback: Callback;
	onDelete: () => void;
}

function CallbackCard({ callback, onDelete }: CallbackCardProps) {
	const isBashScript = callback.type === CallbackType.BashScript;

	return (
		<Card className="relative">
			<CardHeader className="p-3 pb-2">
				<div className="flex items-start justify-between gap-2">
					<div className="flex-1 min-w-0">
						<div className="font-medium text-sm mb-1 truncate">
							{callback.name}
						</div>
						<div className="flex flex-wrap gap-1">
							<Badge variant="secondary" className="text-xs">
								{callback.toolName || "*"}
							</Badge>
							<Badge variant="outline" className="text-xs">
								{callback.type}
							</Badge>
						</div>
					</div>
					<Button
						variant="ghost"
						size="icon"
						className="h-6 w-6 shrink-0"
						onClick={onDelete}
					>
						<Trash2 className="h-3 w-3" />
					</Button>
				</div>
			</CardHeader>
			<CardContent className="p-3 pt-0">
				{isBashScript ? (
					<>
						<p className="text-xs font-mono bg-muted p-2 rounded text-ellipsis overflow-hidden">
							{callback.bashCommand || "No command"}
						</p>
						{callback.bashTimeout && (
							<p className="text-xs text-muted-foreground mt-1">
								Timeout: {callback.bashTimeout}ms
							</p>
						)}
						{callback.nonBlocking && (
							<Badge variant="outline" className="text-xs mt-1">
								Non-blocking
							</Badge>
						)}
					</>
				) : (
					<>
						<p className="text-xs bg-muted p-2 rounded">
							{callback.subAgentPrompt || "No prompt"}
						</p>
						{callback.subAgentType && (
							<p className="text-xs text-muted-foreground mt-1">
								Type: {callback.subAgentType}
							</p>
						)}
					</>
				)}
			</CardContent>
		</Card>
	);
}

interface CallbackFormProps {
	onSubmit: (callback: Callback) => void;
	onCancel: () => void;
}

function CallbackForm({ onSubmit, onCancel }: CallbackFormProps) {
	const [name, setName] = React.useState("");
	const [toolName, setToolName] = React.useState("*");
	const [type, setType] = React.useState<CallbackType>(
		CallbackType.BashScript,
	);
	const [bashCommand, setBashCommand] = React.useState("");
	const [bashTimeout, setBashTimeout] = React.useState("120000");
	const [nonBlocking, setNonBlocking] = React.useState(false);
	const [subAgentPrompt, setSubAgentPrompt] = React.useState("");
	const [subAgentType, setSubAgentType] = React.useState("general-purpose");

	const isBashScript = type === CallbackType.BashScript;

	const handleSubmit = (e: React.FormEvent) => {
		e.preventDefault();

		// Generate default name if not provided
		const callbackName = name.trim() || `Callback #${Math.floor(Math.random() * 9000) + 1000}`;

		const callback: Callback = {
			name: callbackName,
			toolName,
			type,
			...(isBashScript
				? {
						bashCommand,
						bashTimeout: parseInt(bashTimeout, 10),
						nonBlocking,
					}
				: {
						subAgentPrompt,
						subAgentType,
					}),
		};

		onSubmit(callback);
	};

	return (
		<form onSubmit={handleSubmit} className="space-y-4">
			<div className="space-y-2">
				<Label htmlFor="name" className="text-xs">
					Callback Name
				</Label>
				<Input
					id="name"
					value={name}
					onChange={(e) => setName(e.target.value)}
					placeholder="Callback #1234 (auto-generated)"
					className="h-8 text-xs"
				/>
			</div>

			<div className="space-y-2">
				<Label htmlFor="toolName" className="text-xs">
					Tool Name
				</Label>
				<Select value={toolName} onValueChange={setToolName}>
					<SelectTrigger id="toolName" className="h-8 text-xs">
						<SelectValue />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value="*">* (All tools)</SelectItem>
						<SelectItem value="Bash">Bash</SelectItem>
						<SelectItem value="Edit">Edit</SelectItem>
						<SelectItem value="Write">Write</SelectItem>
						<SelectItem value="ReadText">ReadText</SelectItem>
						<SelectItem value="ReadMedia">ReadMedia</SelectItem>
						<SelectItem value="Glob">Glob</SelectItem>
						<SelectItem value="Grep">Grep</SelectItem>
						<SelectItem value="ShowMedia">ShowMedia</SelectItem>
						<SelectItem value="Task">Task</SelectItem>
						<SelectItem value="WebFetch">WebFetch</SelectItem>
						<SelectItem value="Search">Search</SelectItem>
						<SelectItem value="TodoWrite">TodoWrite</SelectItem>
						<SelectItem value="ExitPlanMode">ExitPlanMode</SelectItem>
					</SelectContent>
				</Select>
			</div>

			<div className="space-y-2">
				<Label htmlFor="type" className="text-xs">
					Callback Type
				</Label>
				<Select
					value={type}
					onValueChange={(v) => setType(v as CallbackType)}
				>
					<SelectTrigger id="type" className="h-8 text-xs">
						<SelectValue />
					</SelectTrigger>
					<SelectContent>
						<SelectItem value={CallbackType.BashScript}>Bash Script</SelectItem>
						<SelectItem value={CallbackType.SubAgent}>Sub Agent</SelectItem>
					</SelectContent>
				</Select>
			</div>

			{isBashScript ? (
				<>
					<div className="space-y-2">
						<div className="flex items-center justify-between">
							<Label htmlFor="bashCommand" className="text-xs">
								Bash Command
							</Label>
							<Popover>
								<PopoverTrigger asChild>
									<Button variant="ghost" size="icon" className="h-6 w-6">
										<Settings className="h-3 w-3" />
									</Button>
								</PopoverTrigger>
								<PopoverContent className="w-64" align="end">
									<div className="space-y-4">
										<div className="space-y-2">
											<Label
												htmlFor="bashTimeout"
												className="text-xs font-medium"
											>
												Timeout (ms)
											</Label>
											<Input
												id="bashTimeout"
												type="number"
												value={bashTimeout}
												onChange={(e) => setBashTimeout(e.target.value)}
												className="h-8 text-xs"
											/>
										</div>

										<div className="flex items-center space-x-2">
											<Switch
												id="nonBlocking"
												checked={nonBlocking}
												onCheckedChange={setNonBlocking}
											/>
											<Label htmlFor="nonBlocking" className="text-xs">
												Non-blocking (async)
											</Label>
										</div>
									</div>
								</PopoverContent>
							</Popover>
						</div>
						<Textarea
							id="bashCommand"
							value={bashCommand}
							onChange={(e) => setBashCommand(e.target.value)}
							placeholder="echo $CALLBACK_TOOL_RESULT"
							className="text-xs font-mono min-h-20"
							required
						/>
					</div>
				</>
			) : (
				<>
					<div className="space-y-2">
						<Label htmlFor="subAgentPrompt" className="text-xs">
							Sub-agent Prompt
						</Label>
						<Textarea
							id="subAgentPrompt"
							value={subAgentPrompt}
							onChange={(e) => setSubAgentPrompt(e.target.value)}
							placeholder="Analyze the tool output and suggest improvements"
							className="text-xs min-h-20"
							required
						/>
					</div>

					<div className="space-y-2">
						<Label htmlFor="subAgentType" className="text-xs">
							Sub-agent Type
						</Label>
						<Input
							id="subAgentType"
							value={subAgentType}
							onChange={(e) => setSubAgentType(e.target.value)}
							className="h-8 text-xs"
						/>
					</div>
				</>
			)}

			<div className="flex gap-2 pt-2">
				<Button type="submit" size="sm" className="flex-1">
					Add
				</Button>
				<Button type="button" size="sm" variant="outline" onClick={onCancel}>
					Cancel
				</Button>
			</div>
		</form>
	);
}
