/**
 * Core provider information
 */
export interface ProviderInfo {
  id: string;
  displayName: string;
  authenticated: boolean;
  authMethod?: "api_key" | "oauth";
  isPreferred?: boolean;
  authMethods: ("api_key" | "oauth")[];
}

/**
 * Model information
 */
export interface ModelInfo {
  id: string;
  displayName: string;
  isSelected?: boolean;
}

/**
 * Provider with associated models for hierarchical selection
 */
export interface ProviderWithModels extends ProviderInfo {
  models: ModelInfo[];
}

/**
 * Hierarchical model data structure for CMDK
 */
export interface HierarchicalModelData {
  providers: ProviderWithModels[];
  currentProvider?: string;
  currentModel?: string;
}

/**
 * Provider display component props
 */
export interface ProviderDisplayProps {
  data: {
    providers: ProviderInfo[];
    currentProvider?: string;
  };
  onUpdate: (message: any) => void;
}

/**
 * Model display component props
 */
export interface ModelDisplayProps {
  data: {
    models: ModelInfo[];
    currentModel?: string;
    provider: {
      id: string;
      displayName: string;
    };
  };
  onUpdate: (message: any) => void;
}