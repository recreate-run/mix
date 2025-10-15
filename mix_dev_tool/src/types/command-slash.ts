import type { QueryKey } from "@tanstack/react-query";
import type { HierarchicalModelData } from "@/types";

export type { HierarchicalModelData };

export interface CommandSlashProps {
	onClose: () => void;
	sessionId: string;
	onNewSession?: () => void;
	onQueryClientInvalidate?: (keys: QueryKey) => void;
	onSubmitMessage?: (message: string) => void;
}

export interface Provider {
	id: string;
	displayName: string;
	authenticated: boolean;
	authMethod?: "api_key" | "oauth";
	isPreferred?: boolean;
}

export interface LoginProvider extends Provider {
	authMethods: ("api_key" | "oauth")[];
	apiKeyFormat?: string;
}

export interface StatusData {
	providers: Provider[];
}

interface HelpMenuItem {
	id: string;
	name: string;
	description: string;
	action: string;
	url?: string;
}

export interface HelpData {
	menuItems: HelpMenuItem[];
}

export type AuthMethod = "api_key" | "oauth" | "oauth_code";

export type ViewState =
	| "commands"
	| "permissions"
	| "sessions"
	| "mcp"
	| "mcp-tools"
	| "hierarchical-model"
	| "hierarchical-models"
	| "status"
	| "help";
