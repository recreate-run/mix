# Enhanced Login Command Implementation Plan

## Overview

This document outlines the plan to enhance the `/login` command in the Mix application. The goal is to create a more intuitive authentication flow that allows users to select a provider and authenticate using either API keys or OAuth, depending on provider support.

## Current Implementation

The current `/login` command implementation has several limitations:

1. It doesn't allow users to select a specific provider to authenticate with
2. The authentication UI doesn't clearly distinguish between API key and OAuth flows
3. There's no feedback about which providers are already authenticated
4. The implementation doesn't fully utilize the capabilities of the Mix TypeScript SDK

## Enhanced Implementation Plan

### 1. Create Login Command Handler

Create a dedicated handler for the `/login` command that uses the SDK to:

- Fetch current authentication status
- Display provider selection interface
- Handle authentication based on provider type

### 2. Provider Selection UI

Create a provider selection interface that:

- Shows all available providers (Anthropic, OpenAI, OpenRouter)
- Indicates which providers are already authenticated
- Shows which authentication methods are supported for each provider
- Allows the user to select a provider for authentication

### 3. Authentication Method Selection

After provider selection:

- For providers with multiple auth methods (e.g., Anthropic with both OAuth and API key), show a method selection UI
- For providers with only API key authentication, go directly to API key input
- Clearly indicate the expected API key format for each provider

### 4. Authentication Flow Implementation

#### API Key Authentication

- Input field for API key with format guidance
- Validation of API key format before submission
- Clear success/error feedback
- Storing the API key via the SDK

#### OAuth Authentication (for Anthropic)

- "Connect with Anthropic" button that opens the OAuth URL
- Field for entering the authorization code
- Handling the OAuth callback
- Clear success/error feedback

### 5. Feedback and Status Updates

- Show loading states during authentication
- Clear success/error messages
- Update authentication status after successful authentication
- Option to retry on failure

## Technical Implementation Details

### 1. New Files to Create

#### `src/handlers/login-command-handler.ts`

```typescript
import { mix } from "@/lib/mix-sdk";
import { UIMessage } from "@/types/message";

// Provider type definition
interface ProviderInfo {
  id: string;
  displayName: string;
  authMethods: ("api_key" | "oauth")[];
  authenticated: boolean;
  apiKeyFormat?: string;
}

/**
 * Handles the login command, returns a message with login UI or success/error message
 */
export async function handleLoginCommand(provider?: string): Promise<UIMessage> {
  try {
    // Get current authentication status
    const status = await mix.authentication.getAuthStatus();
    
    // Map available providers
    const providers: ProviderInfo[] = [
      {
        id: "anthropic",
        displayName: "Anthropic",
        authMethods: ["api_key", "oauth"],
        authenticated: status.providers?.anthropic?.authenticated || false,
        apiKeyFormat: "sk-ant-..."
      },
      {
        id: "openai",
        displayName: "OpenAI (GPT)",
        authMethods: ["api_key"],
        authenticated: status.providers?.openai?.authenticated || false,
        apiKeyFormat: "sk-..."
      },
      {
        id: "openrouter",
        displayName: "OpenRouter",
        authMethods: ["api_key"],
        authenticated: status.providers?.openrouter?.authenticated || false,
        apiKeyFormat: "sk-..."
      }
    ];
    
    // Return message with login UI elements and provider info
    return {
      content: "",
      from: "assistant",
      frontend_only: true,
      login: {
        providers,
        selectedProvider: provider,
        step: provider ? "auth_method" : "provider_select"
      }
    };
  } catch (error) {
    return {
      content: `Failed to initialize login: ${error instanceof Error ? error.message : "Unknown error"}`,
      from: "assistant",
      frontend_only: true
    };
  }
}

/**
 * Authenticates with a provider using an API key
 */
export async function authenticateWithApiKey(
  provider: string,
  apiKey: string
): Promise<UIMessage> {
  try {
    // Store API key using the SDK
    const result = await mix.authentication.storeApiKey({
      provider,
      apiKey
    });
    
    return {
      content: `✅ Successfully authenticated with ${provider} using API key`,
      from: "assistant",
      frontend_only: true
    };
  } catch (error) {
    return {
      content: `❌ Failed to authenticate: ${error instanceof Error ? error.message : "Unknown error"}`,
      from: "assistant",
      frontend_only: true
    };
  }
}

/**
 * Starts OAuth flow for a provider (currently only Anthropic)
 */
export async function startOAuthFlow(provider: string): Promise<UIMessage> {
  try {
    // Only Anthropic supports OAuth
    if (provider !== "anthropic") {
      return {
        content: `❌ OAuth is not supported for ${provider}`,
        from: "assistant",
        frontend_only: true
      };
    }
    
    // Start OAuth flow using the SDK
    const result = await mix.authentication.startOAuthFlow({
      provider
    });
    
    // Return success with auth URL
    return {
      content: "",
      from: "assistant",
      frontend_only: true,
      login: {
        step: "oauth_flow",
        authUrl: result.authUrl,
        provider
      }
    };
  } catch (error) {
    return {
      content: `❌ Failed to start OAuth flow: ${error instanceof Error ? error.message : "Unknown error"}`,
      from: "assistant",
      frontend_only: true
    };
  }
}

/**
 * Handles OAuth callback with authorization code
 */
export async function handleOAuthCallback(
  provider: string,
  code: string
): Promise<UIMessage> {
  try {
    // Handle OAuth callback using the SDK
    await mix.authentication.handleOAuthCallback({
      provider,
      code
    });
    
    return {
      content: `✅ Successfully authenticated with ${provider} using OAuth`,
      from: "assistant",
      frontend_only: true
    };
  } catch (error) {
    return {
      content: `❌ Failed to complete OAuth: ${error instanceof Error ? error.message : "Unknown error"}`,
      from: "assistant",
      frontend_only: true
    };
  }
}
```

#### `src/components/login-ui.tsx`

```typescript
import React, { useState } from "react";
import { AlertCircle, CheckCircle, ExternalLink, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  authenticateWithApiKey,
  handleOAuthCallback,
  startOAuthFlow
} from "@/handlers/login-command-handler";

// Provider info structure 
interface ProviderInfo {
  id: string;
  displayName: string;
  authMethods: ("api_key" | "oauth")[];
  authenticated: boolean;
  apiKeyFormat?: string;
}

// Login state from command handler
interface LoginState {
  providers: ProviderInfo[];
  selectedProvider?: string;
  step: "provider_select" | "auth_method" | "api_key" | "oauth_flow" | "oauth_code";
  authUrl?: string;
}

interface LoginUIProps {
  loginState: LoginState;
  onUpdate: (message: any) => void;
}

export function LoginUI({ loginState, onUpdate }: LoginUIProps) {
  const [selectedProvider, setSelectedProvider] = useState<string>(loginState.selectedProvider || "");
  const [selectedMethod, setSelectedMethod] = useState<"api_key" | "oauth">("api_key");
  const [apiKey, setApiKey] = useState("");
  const [authCode, setAuthCode] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [step, setStep] = useState<LoginState["step"]>(loginState.step);
  
  // Handle provider selection
  const handleProviderSelect = (provider: string) => {
    setSelectedProvider(provider);
    
    const providerInfo = loginState.providers.find(p => p.id === provider);
    
    // If provider only has one auth method, skip method selection
    if (providerInfo && providerInfo.authMethods.length === 1) {
      setSelectedMethod(providerInfo.authMethods[0]);
      setStep(providerInfo.authMethods[0] === "api_key" ? "api_key" : "oauth_flow");
    } else {
      setStep("auth_method");
    }
  };
  
  // Handle auth method selection
  const handleMethodSelect = (method: "api_key" | "oauth") => {
    setSelectedMethod(method);
    setStep(method === "api_key" ? "api_key" : "oauth_flow");
  };
  
  // Handle API key submission
  const handleApiKeySubmit = async () => {
    if (!selectedProvider || !apiKey) return;
    
    setIsLoading(true);
    try {
      const result = await authenticateWithApiKey(selectedProvider, apiKey);
      onUpdate(result);
    } finally {
      setIsLoading(false);
    }
  };
  
  // Start OAuth flow
  const handleStartOAuth = async () => {
    if (!selectedProvider) return;
    
    setIsLoading(true);
    try {
      const result = await startOAuthFlow(selectedProvider);
      
      if (result.login?.authUrl) {
        // Open auth URL in browser
        window.open(result.login.authUrl, "_blank");
        setStep("oauth_code");
      } else {
        onUpdate(result);
      }
    } finally {
      setIsLoading(false);
    }
  };
  
  // Handle OAuth code submission
  const handleOAuthCodeSubmit = async () => {
    if (!selectedProvider || !authCode) return;
    
    setIsLoading(true);
    try {
      const result = await handleOAuthCallback(selectedProvider, authCode);
      onUpdate(result);
    } finally {
      setIsLoading(false);
    }
  };
  
  // Provider selection screen
  if (step === "provider_select") {
    return (
      <Card>
        <CardContent className="p-4">
          <h3 className="font-medium mb-2">Select a provider to authenticate with:</h3>
          <div className="space-y-2">
            {loginState.providers.map(provider => (
              <Button
                key={provider.id}
                onClick={() => handleProviderSelect(provider.id)}
                variant={provider.authenticated ? "outline" : "default"}
                className="w-full justify-start"
              >
                {provider.displayName}
                {provider.authenticated && (
                  <CheckCircle className="h-4 w-4 ml-2 text-green-600" />
                )}
              </Button>
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }
  
  // Auth method selection screen
  if (step === "auth_method") {
    const provider = loginState.providers.find(p => p.id === selectedProvider);
    if (!provider) return null;
    
    return (
      <Card>
        <CardContent className="p-4">
          <h3 className="font-medium mb-2">Select authentication method for {provider.displayName}:</h3>
          <div className="space-y-2">
            {provider.authMethods.includes("api_key") && (
              <Button
                onClick={() => handleMethodSelect("api_key")}
                variant="default"
                className="w-full justify-start"
              >
                API Key
              </Button>
            )}
            {provider.authMethods.includes("oauth") && (
              <Button
                onClick={() => handleMethodSelect("oauth")}
                variant="default"
                className="w-full justify-start"
              >
                OAuth
              </Button>
            )}
          </div>
          <Button
            onClick={() => setStep("provider_select")}
            variant="ghost"
            className="mt-4"
          >
            Back
          </Button>
        </CardContent>
      </Card>
    );
  }
  
  // API key input screen
  if (step === "api_key") {
    const provider = loginState.providers.find(p => p.id === selectedProvider);
    if (!provider) return null;
    
    return (
      <Card>
        <CardContent className="p-4">
          <h3 className="font-medium mb-2">Enter API key for {provider.displayName}:</h3>
          <p className="text-sm text-muted-foreground mb-4">
            Format: {provider.apiKeyFormat || "API key"}
          </p>
          <div className="space-y-4">
            <div className="flex gap-2">
              <Input
                type="password"
                value={apiKey}
                onChange={(e) => setApiKey(e.target.value)}
                placeholder={provider.apiKeyFormat || "Enter API key"}
                disabled={isLoading}
                onKeyDown={(e) => e.key === "Enter" && handleApiKeySubmit()}
              />
              <Button
                onClick={handleApiKeySubmit}
                disabled={!apiKey.trim() || isLoading}
              >
                {isLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : "Submit"}
              </Button>
            </div>
            <Button
              onClick={() => provider.authMethods.length > 1 ? setStep("auth_method") : setStep("provider_select")}
              variant="ghost"
            >
              Back
            </Button>
          </div>
        </CardContent>
      </Card>
    );
  }
  
  // OAuth flow screen
  if (step === "oauth_flow") {
    const provider = loginState.providers.find(p => p.id === selectedProvider);
    if (!provider) return null;
    
    return (
      <Card>
        <CardContent className="p-4">
          <h3 className="font-medium mb-2">Connect with {provider.displayName}:</h3>
          <p className="text-sm text-muted-foreground mb-4">
            You will be redirected to {provider.displayName} to authorize this application.
          </p>
          <div className="space-y-4">
            <Button
              onClick={handleStartOAuth}
              disabled={isLoading}
              className="w-full"
            >
              {isLoading ? (
                <Loader2 className="h-4 w-4 animate-spin mr-2" />
              ) : (
                <ExternalLink className="h-4 w-4 mr-2" />
              )}
              Connect with {provider.displayName}
            </Button>
            <Button
              onClick={() => provider.authMethods.length > 1 ? setStep("auth_method") : setStep("provider_select")}
              variant="ghost"
            >
              Back
            </Button>
          </div>
        </CardContent>
      </Card>
    );
  }
  
  // OAuth code input screen
  if (step === "oauth_code") {
    const provider = loginState.providers.find(p => p.id === selectedProvider);
    if (!provider) return null;
    
    return (
      <Card>
        <CardContent className="p-4">
          <h3 className="font-medium mb-2">Enter authorization code from {provider.displayName}:</h3>
          <p className="text-sm text-muted-foreground mb-4">
            After authorizing, copy the code from {provider.displayName} and paste it below.
          </p>
          <div className="space-y-4">
            <div className="flex gap-2">
              <Input
                value={authCode}
                onChange={(e) => setAuthCode(e.target.value)}
                placeholder="Authorization code"
                disabled={isLoading}
                onKeyDown={(e) => e.key === "Enter" && handleOAuthCodeSubmit()}
              />
              <Button
                onClick={handleOAuthCodeSubmit}
                disabled={!authCode.trim() || isLoading}
              >
                {isLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : "Submit"}
              </Button>
            </div>
            <Button
              onClick={() => setStep("oauth_flow")}
              variant="ghost"
            >
              Back
            </Button>
          </div>
        </CardContent>
      </Card>
    );
  }
  
  return null;
}
```

### 2. Updates to Existing Files

#### Update `src/types/message.ts`

```typescript
export interface UIMessage {
  content: string;
  from: 'user' | 'assistant';
  frontend_only?: boolean;
  toolCalls?: ToolCall[];
  attachments?: Attachment[];
  timeline?: TimelineEntry[];
  mediaOutputs?: MediaOutput[];
  reasoning?: string;
  reasoningDuration?: number;
  login?: {
    providers?: any[];
    selectedProvider?: string;
    step?: string;
    authUrl?: string;
  };
}
```

#### Update `src/components/conversation-display.tsx`

```typescript
// Add import for LoginUI component
import { LoginUI } from './login-ui';

// In the message rendering section:
{message.login && (
  <LoginUI 
    loginState={message.login} 
    onUpdate={(updatedMessage) => {
      // Handle login flow updates
      setMessages((prev) => [
        ...prev.slice(0, -1), // Remove current message
        updatedMessage // Add updated message
      ]);
    }}
  />
)}
```

#### Update `src/components/chat-app.tsx`

```typescript
// Add import for login command handler
import { handleLoginCommand } from '@/handlers/login-command-handler';

// Update command handler to handle /login command
if (command === 'login') {
  handleLoginCommand().then(result => {
    // Add user message to show that command was executed
    setMessages((prev) => [
      ...prev,
      {
        content: "/login",
        from: "user",
      },
    ]);
    
    // Add response with login UI
    setMessages((prev) => [
      ...prev,
      result
    ]);
  }).catch(error => {
    console.error("Login command failed:", error);
    setMessages((prev) => [
      ...prev,
      {
        content: `/login command failed: ${error}`,
        from: "assistant",
        frontend_only: true,
      },
    ]);
  });
  return;
}
```

## Implementation Steps

1. Create the new files:
   - `src/handlers/login-command-handler.ts`
   - `src/components/login-ui.tsx`

2. Update existing files:
   - Add `login` field to UIMessage interface in `src/types/message.ts`
   - Update `src/components/conversation-display.tsx` to render the LoginUI component
   - Update `src/components/chat-app.tsx` to handle the `/login` command

3. Test the enhanced login command:
   - Test provider selection
   - Test API key authentication for all providers
   - Test OAuth authentication for Anthropic
   - Verify error handling and feedback

4. Document the enhanced login flow for users

## Benefits of Enhanced Implementation

1. **Improved UX**: Clearer authentication flow with visual guidance
2. **Provider Selection**: Users can choose which provider to authenticate with
3. **Authentication Method Clarity**: Clear distinction between API key and OAuth flows
4. **Better Feedback**: Improved success/error messages and loading states
5. **Consistency**: UI aligned with the rest of the application
6. **SDK Integration**: Proper utilization of Mix TypeScript SDK

## Future Enhancements

1. **Remember Last Provider**: Store the last used provider for quicker authentication
2. **Provider Icons**: Add provider logos for visual recognition
3. **API Key Validation**: Client-side validation of API key format before submission
4. **Auto-Detection**: Auto-detect API key format and provider from pasted keys
5. **Session Management**: Allow viewing and managing authenticated sessions

## Conclusion

This enhanced login command implementation will significantly improve the authentication experience in the Mix application. By providing a clear, step-by-step flow with proper feedback and provider selection, users will be able to authenticate more easily and with less confusion.

The implementation leverages the Mix TypeScript SDK properly and follows best practices for React component design, state management, and error handling.
