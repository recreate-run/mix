import { useState } from "react";
// Import shell to open URLs since opener plugin isn't properly exposed
import { open as shellOpen } from '@tauri-apps/plugin-shell';
import { CheckCircle, ExternalLink, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  authenticateWithApiKey,
  handleOAuthCallback,
  startOAuthFlow
} from "@/handlers/login-command-handler";

// Import provider types
import { ProviderInfo } from "@/types/provider";

// Extended provider info with API key format
interface LoginProviderInfo extends ProviderInfo {
  apiKeyFormat?: string;
}

// Login state from command handler
interface LoginState {
  providers: LoginProviderInfo[];
  selectedProvider?: string;
  step: "provider_select" | "auth_method" | "api_key" | "oauth_flow" | "oauth_code";
  authUrl?: string;
  state?: string; // Add state parameter for OAuth flow
  hasExistingPreferences?: boolean;
}

interface LoginUIProps {
  loginState: LoginState;
  onUpdate: (message: any) => void;
}

export function LoginUI({ loginState, onUpdate }: LoginUIProps) {
  // Validate required login state data
  if (!loginState.providers || loginState.providers.length === 0) {
    throw new Error("LoginUI: No providers available in login state");
  }

  const [selectedProvider, setSelectedProvider] = useState<string>(loginState.selectedProvider || "");
  const [apiKey, setApiKey] = useState("");
  const [authCode, setAuthCode] = useState("");
  const [isLoading, setIsLoading] = useState(false);
  const [step, setStep] = useState<LoginState["step"]>(loginState.step);
  const [oauthState, setOauthState] = useState<string>(""); // Store OAuth state
  
  // No automatic browser window opening on component mount
  // This will only happen when the user explicitly clicks the OAuth button
  
  // Handle provider selection
  const handleProviderSelect = (provider: string) => {
    setSelectedProvider(provider);
    
    const providerInfo = loginState.providers.find(p => p.id === provider);
    if (!providerInfo) {
      throw new Error(`Provider ${provider} not found in available providers`);
    }
    
    // If provider only has one auth method, skip method selection
    if (providerInfo.authMethods.length === 1) {
      setStep(providerInfo.authMethods[0] === "api_key" ? "api_key" : "oauth_flow");
    } else {
      setStep("auth_method");
    }
  };
  
  // Handle auth method selection
  const handleMethodSelect = (method: "api_key" | "oauth") => {
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
      
      if (!result.login) {
        throw new Error("OAuth flow initiation failed: no login data returned");
      }
      if (!result.login.authUrl) {
        throw new Error("OAuth flow initiation failed: no auth URL provided");
      }
      if (result.login.authUrl) {
        // Store the state parameter from the OAuth flow initiation
        if (result.login.state) {
          setOauthState(result.login.state);
        }
        
        // Automatically open the browser window when OAuth flow is started
        // Using Tauri's opener plugin instead of window.open for desktop apps
        try {
          // Use Tauri shell plugin to open URLs
          await shellOpen(result.login.authUrl);
        } catch (error) {
          console.warn("Failed to open with Tauri shell, falling back to window.open:", error);
          // Fallback to window.open
          try {
            window.open(result.login.authUrl, "_blank");
          } catch (windowError) {
            console.error("Both opener methods failed:", windowError);
          }
        }
        
        // Update UI step
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
      // Clean up the authorization code - remove any trailing whitespace
      let cleanCode = authCode.trim();
      
      // Extract just the code part if it contains a hash - the backend will handle this properly
      // Don't try to parse the code here - send it as is to the backend
      
      
      // Pass the stored state parameter to the callback handler
      const result = await handleOAuthCallback(selectedProvider, cleanCode, oauthState);
      onUpdate(result);
    } catch (error) {
      console.error("OAuth callback error:", error);
      onUpdate({
        content: `❌ Authentication failed: ${error instanceof Error ? error.message : "Unknown error. If this is an OAuth issue, try using an API key instead."}`,
        from: "assistant",
        frontend_only: true
      });
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
                className={`w-full justify-start ${loginState.selectedProvider === provider.id ? "border-2 border-primary" : ""}`}
              >
                {provider.displayName}
                {provider.authenticated && (
                  <CheckCircle className="h-4 w-4 ml-2 text-green-600" />
                )}
                {loginState.hasExistingPreferences && loginState.selectedProvider === provider.id && (
                  <span className="ml-auto text-xs text-muted-foreground">(Preferred)</span>
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
    if (!provider) {
      throw new Error(`Provider ${selectedProvider} not found for auth method selection`);
    }
    
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
    if (!provider) {
      throw new Error(`Provider ${selectedProvider} not found for auth method selection`);
    }
    
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
    // Use the selected provider
    const providerIdToUse = selectedProvider;
    const provider = loginState.providers.find(p => p.id === providerIdToUse);
    if (!provider) {
      throw new Error(`Provider ${providerIdToUse} not found for auth method selection`);
    }
    
    // Check if there's an auth URL in the login state
    const authUrl = loginState.authUrl;
    
    return (
      <Card>
        <CardContent className="p-4">
          <h3 className="font-medium mb-2">Connect with {provider.displayName}:</h3>
          <p className="text-sm text-muted-foreground mb-4">
            You will be redirected to {provider.displayName} to authorize this application.
          </p>
          <div className="space-y-4">
            {/* Always show auth URL for better UX */}
            {authUrl && (
              <div className="p-3 bg-amber-50 dark:bg-amber-900 border border-amber-200 dark:border-amber-800 rounded-md mb-3">
                <p className="text-sm mb-2 font-medium">Authentication URL:</p>
                <div className="flex flex-col gap-3">
                  <a 
                    href={authUrl} 
                    target="_blank" 
                    className="text-sm text-blue-600 hover:underline break-all"
                    rel="noreferrer"
                  >
                    {authUrl}
                  </a>
                  <div className="flex gap-2">
                    <Button 
                      onClick={() => window.open(authUrl, "_blank")}
                      size="sm"
                      className="flex-1"
                    >
                      <ExternalLink className="h-3 w-3 mr-1" />
                      Open in browser
                    </Button>
                    <Button 
                      onClick={() => {
                        navigator.clipboard.writeText(authUrl);
                        // Show brief copied feedback
                        const button = document.activeElement as HTMLButtonElement;
                        if (button) {
                          const originalText = button.innerText;
                          button.innerText = "Copied!";
                          setTimeout(() => {
                            button.innerText = originalText;
                          }, 1000);
                        }
                      }}
                      size="sm"
                      variant="outline"
                    >
                      Copy URL
                    </Button>
                  </div>
                </div>
              </div>
            )}
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
    if (!provider) {
      throw new Error(`Provider ${selectedProvider} not found for auth method selection`);
    }
    const authUrl = loginState.authUrl;
    
    return (
      <Card>
        <CardContent className="p-4">
          <h3 className="font-medium mb-2">Connect with {provider.displayName}:</h3>
          
          {/* Authentication URL section */}
          {authUrl && (
            <div className="p-3 bg-amber-50 dark:bg-amber-900 border border-amber-200 dark:border-amber-800 rounded-md mb-4">
              <p className="text-sm mb-2"><strong>Browser window should have opened automatically.</strong> If not, click the button below:</p>
              <Button
                onClick={async () => {
                  try {
                    await shellOpen(authUrl);
                  } catch (error) {
                    console.warn("Shell plugin failed, falling back to window.open", error);
                    window.open(authUrl, "_blank");
                  }
                }}
                className="w-full mb-2"
              >
                <ExternalLink className="h-4 w-4 mr-2" />
                Open Authorization Page
              </Button>
            </div>
          )}
          
          {/* Code input section */}
          <div className="mb-4">
            <p className="text-sm mb-2"><strong>Step 2:</strong> After authorizing, paste the code below:</p>
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
          </div>
          
          <Button
            onClick={() => setStep("provider_select")}
            variant="ghost"
          >
            Back
          </Button>
        </CardContent>
      </Card>
    );
  }
  
  return null;
}