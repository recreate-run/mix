import { AlertCircle, CheckCircle, ExternalLink, Info, Loader2, LogIn } from 'lucide-react';
import { useState } from 'react';
import { rpcCall } from '@/lib/rpc';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Input } from '@/components/ui/input';

interface AuthStatusResponse {
  type: string;
  status: string; // "authenticated" | "not_authenticated"
  provider: string; // "anthropic"
  expiresIn?: number; // minutes until expiry
  message: string;
}

interface AuthLoginResponse {
  type: string;
  status: string; // "success" | "pending" | "error"
  message: string;
  authUrl?: string; // for OAuth flow
  step?: string; // current step in flow
}

interface ErrorResponse {
  type: string;
  error: string;
  command?: string;
}

interface MessageResponse {
  type: string;
  message: string;
  command?: string;
}

type AuthDisplayProps = 
  | { data: AuthStatusResponse }
  | { data: AuthLoginResponse }
  | { data: ErrorResponse }
  | { data: MessageResponse };

export function AuthDisplay({ data }: AuthDisplayProps) {
  const [authCode, setAuthCode] = useState('');
  const [apiKey, setApiKey] = useState('');
  const [authMode, setAuthMode] = useState<'code' | 'apikey'>('code');
  const [isExchanging, setIsExchanging] = useState(false);

  const handleAuthCodeSubmit = async () => {
    if (!authCode.trim()) return;
    
    // Ensure the code has the correct format: code#state
    const trimmedCode = authCode.trim();
    const isAPIKey = trimmedCode.startsWith('sk-ant-');
    const isValidFormat = isAPIKey || trimmedCode.includes('#');
    
    if (!isValidFormat) {
      alert('Invalid format. Enter either:\n- Full authorization code (format: code#state)\n- Or Anthropic API key (starts with sk-ant-)');
      return;
    }
    
    // Log the code format details for debugging
    if (!isAPIKey) {
      const parts = trimmedCode.split('#');
      console.log('Auth code format details:', {
        fullLength: trimmedCode.length,
        codePartLength: parts[0].length,
        statePartLength: parts[1] ? parts[1].length : 0,
        hasHashSymbol: trimmedCode.includes('#'),
        partsCount: parts.length
      });
    } else {
      console.log('Submitting API key (redacted):', '*'.repeat(10));
    }
    
    setIsExchanging(true);
    try {
      // Use the direct auth.login RPC method
      // If it's an API key, we'll mark it as manual to treat it differently
      const result = await rpcCall('auth.login', {
        authCode: trimmedCode,
        manual: isAPIKey
      });
      
      console.log('Authentication successful:', result);
      
      // Show a success message
      if (result.status === 'success') {
        alert(`Authentication successful! You can now use the application. Token expires in ${result.expiresIn} minutes.`);
      } else {
        // For manual fallback or other non-error states
        // If we got a manual_fallback response, switch to API key mode
        if (result.step === 'manual_fallback') {
          setAuthMode('apikey');
          alert('Due to Cloudflare protection, please use API key authentication instead. You can get an API key from https://console.anthropic.com/settings/keys');
        } else {
          alert(result.message || 'Authentication processed but may require additional steps. Please check the logs for more information.');
        }
      }
      
      // Only reload if successful
      if (result.status === 'success') {
        // Reload the application after a short delay
        setTimeout(() => {
          window.location.reload();
        }, 1000);
      }
      
    } catch (error) {
      console.error('Auth code exchange failed:', error);
      
      // Check if it's a Cloudflare protection error
      const errorMsg = error instanceof Error ? error.message : 'Unknown error';
      if (errorMsg.includes('Cloudflare') || errorMsg.includes('manual token') || 
          errorMsg.includes('invalid_grant')) {
        // Switch to API key mode
        setAuthMode('apikey');
        alert('Authentication requires API key setup due to Cloudflare protection. Please use the API key option below.');
      } else {
        alert('Authentication failed: ' + errorMsg);
      }
    } finally {
      setIsExchanging(false);
      setAuthCode(''); // Clear the input field
    }
  };
  
  const handleApiKeySubmit = async () => {
    if (!apiKey.trim()) return;
    
    // Validate API key format
    if (!apiKey.trim().startsWith('sk-ant-')) {
      alert('Invalid API key format. Anthropic API keys must start with "sk-ant-"');
      return;
    }
    
    setIsExchanging(true);
    try {
      console.log('Setting API key...');
      
      // Use the auth.login endpoint with apiKey parameter for unified handling
      const result = await rpcCall('auth.login', {
        apiKey: apiKey.trim()
      });
      
      console.log('API key set successfully:', result);
      
      // Show a success message
      alert('API key set successfully! You can now use the application.');
      
      // Reload the application after a short delay
      setTimeout(() => {
        window.location.reload();
      }, 1000);
      
    } catch (error) {
      console.error('API key setting failed:', error);
      alert('Failed to set API key: ' + (error instanceof Error ? error.message : 'Unknown error'));
    } finally {
      setIsExchanging(false);
      setApiKey(''); // Clear the input field
    }
  };

  const openAuthUrl = (url: string) => {
    window.open(url, '_blank');
  };

  // Handle error responses
  if (data.type === 'error') {
    const errorData = data as ErrorResponse;
    return (
      <Card className="border-red-200 bg-red-50 dark:border-red-800 dark:bg-red-950">
        <CardHeader className="pb-3">
          <div className="flex items-center gap-2">
            <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400" />
            <CardTitle className="text-red-800 dark:text-red-200">
              {errorData.command ? `Command Error: /${errorData.command}` : 'Error'}
            </CardTitle>
          </div>
        </CardHeader>
        <CardContent>
          <p className="text-red-700 dark:text-red-300">{errorData.error}</p>
        </CardContent>
      </Card>
    );
  }

  // Handle message responses
  if (data.type === 'message') {
    const messageData = data as MessageResponse;
    return (
      <Card className="border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-950">
        <CardHeader className="pb-3">
          <div className="flex items-center gap-2">
            <Info className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            <CardTitle className="text-blue-800 dark:text-blue-200">
              {messageData.command ? `/${messageData.command}` : 'Information'}
            </CardTitle>
          </div>
        </CardHeader>
        <CardContent>
          <p className="text-blue-700 dark:text-blue-300">{messageData.message}</p>
        </CardContent>
      </Card>
    );
  }

  // Handle auth status responses
  if (data.type === 'auth_status') {
    const statusData = data as AuthStatusResponse;
    const isAuthenticated = statusData.status === 'authenticated';
    
    return (
      <Card className={isAuthenticated 
        ? "border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-950"
        : "border-orange-200 bg-orange-50 dark:border-orange-800 dark:bg-orange-950"
      }>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              {isAuthenticated ? (
                <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400" />
              ) : (
                <AlertCircle className="h-5 w-5 text-orange-600 dark:text-orange-400" />
              )}
              <CardTitle className={isAuthenticated 
                ? "text-green-800 dark:text-green-200"
                : "text-orange-800 dark:text-orange-200"
              }>
                Authentication Status
              </CardTitle>
            </div>
            <Badge variant={isAuthenticated ? "default" : "secondary"}>
              {statusData.provider}
            </Badge>
          </div>
          {statusData.expiresIn && statusData.expiresIn > 0 && (
            <CardDescription className="text-sm">
              Expires in {statusData.expiresIn} minutes
            </CardDescription>
          )}
        </CardHeader>
        <CardContent>
          <p className={isAuthenticated 
            ? "text-green-700 dark:text-green-300"
            : "text-orange-700 dark:text-orange-300"
          }>
            {statusData.message}
          </p>
        </CardContent>
      </Card>
    );
  }

  // Handle auth login responses
  if (data.type === 'auth_login') {
    const loginData = data as AuthLoginResponse;
    
    return (
      <Card className="border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-950">
        <CardHeader className="pb-3">
          <div className="flex items-center gap-2">
            {loginData.status === 'success' ? (
              <CheckCircle className="h-5 w-5 text-green-600 dark:text-green-400" />
            ) : loginData.status === 'pending' ? (
              <LogIn className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            ) : (
              <AlertCircle className="h-5 w-5 text-red-600 dark:text-red-400" />
            )}
            <CardTitle className="text-blue-800 dark:text-blue-200">
              Claude Code Authentication
            </CardTitle>
          </div>
          {loginData.step && (
            <CardDescription className="text-sm">
              Step: {loginData.step}
            </CardDescription>
          )}
        </CardHeader>
        <CardContent className="space-y-4">
          <p className="text-blue-700 dark:text-blue-300">{loginData.message}</p>
          
          {loginData.authUrl && loginData.status === 'pending' && (
            <div className="space-y-3">
              <div className="flex mb-4 border-b">
                <button
                  className={`px-4 py-2 ${authMode === 'code' ? 'text-blue-600 border-b-2 border-blue-600 font-medium' : 'text-gray-500'}`}
                  onClick={() => setAuthMode('code')}
                >
                  OAuth Code
                </button>
                <button
                  className={`px-4 py-2 ${authMode === 'apikey' ? 'text-blue-600 border-b-2 border-blue-600 font-medium' : 'text-gray-500'}`}
                  onClick={() => setAuthMode('apikey')}
                >
                  API Key
                </button>
              </div>

              {authMode === 'code' ? (
                <>
                  <Button 
                    onClick={() => openAuthUrl(loginData.authUrl!)}
                    className="w-full"
                    variant="default"
                  >
                    <ExternalLink className="mr-2 h-4 w-4" />
                    Open Authentication Page
                  </Button>
                  
                  <div className="space-y-2">
                    <label className="text-sm font-medium">
                      After completing authentication, paste either:
                      <br/>- FULL authorization code (format: code#state)
                      <br/>- OR your Anthropic API key (starts with sk-ant-)
                    </label>
                    <div className="flex gap-2">
                      <Input
                        placeholder="Authorization code from callback URL..."
                        value={authCode}
                        onChange={(e) => setAuthCode(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' && !isExchanging && authCode.trim()) {
                            e.preventDefault();
                            handleAuthCodeSubmit();
                          }
                        }}
                        className="flex-1"
                        disabled={isExchanging}
                      />
                      <Button 
                        onClick={handleAuthCodeSubmit}
                        disabled={!authCode.trim() || isExchanging}
                        variant="outline"
                      >
                        {isExchanging ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          'Submit Code'
                        )}
                      </Button>
                    </div>
                  </div>
                  
                  <div className="text-xs bg-yellow-50 dark:bg-yellow-950 border border-yellow-200 dark:border-yellow-800 p-3 rounded">
                    <p className="font-medium text-yellow-800 dark:text-yellow-200 mb-1">Authentication Instructions:</p>
                    <p className="text-yellow-700 dark:text-yellow-300">1. Click "Open Authentication Page" above</p>
                    <p className="text-yellow-700 dark:text-yellow-300">2. Make sure you're already logged into claude.ai</p>
                    <p className="text-yellow-700 dark:text-yellow-300">3. After authentication, you'll be redirected to a URL containing a code</p>
                    <p className="text-yellow-700 dark:text-yellow-300">4. <strong>Important:</strong> Copy the FULL code from the URL</p>
                    <p className="text-yellow-700 dark:text-yellow-300">5. Format must be: code#state (include everything after "code=" up to "&" + "#" + everything after "state=")</p>
                    <p className="text-yellow-700 dark:text-yellow-300">6. Paste the entire code above and click "Submit Code"</p>
                    <p className="text-yellow-700 dark:text-yellow-300">7. If you get a state mismatch error, make sure you're copying the complete code and state</p>
                  </div>
                  
                  <details className="text-xs text-muted-foreground">
                    <summary className="cursor-pointer hover:text-foreground">
                      How to get the authorization code
                    </summary>
                    <div className="mt-2 space-y-1">
                      <p>1. Click "Open Authentication Page" above</p>
                      <p>2. Complete the OAuth flow on claude.ai</p>
                      <p>3. After authorization, you'll be redirected to a callback URL</p>
                      <p>4. Copy the authorization code from the URL and paste it above</p>
                      <p>5. The code will be in the format: code#state (copy everything)</p>
                    </div>
                  </details>
                </>
              ) : (
                <>
                  <div className="space-y-2">
                    <label className="text-sm font-medium">
                      Enter your Anthropic API Key:
                    </label>
                    <div className="flex gap-2">
                      <Input
                        placeholder="sk-ant-..."
                        value={apiKey}
                        onChange={(e) => setApiKey(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' && !isExchanging && apiKey.trim()) {
                            e.preventDefault();
                            handleApiKeySubmit();
                          }
                        }}
                        className="flex-1"
                        disabled={isExchanging}
                        type="password"
                      />
                      <Button 
                        onClick={handleApiKeySubmit}
                        disabled={!apiKey.trim() || isExchanging}
                        variant="outline"
                      >
                        {isExchanging ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          'Set API Key'
                        )}
                      </Button>
                    </div>
                  </div>
                  
                  <div className="text-xs bg-yellow-50 dark:bg-yellow-950 border border-yellow-200 dark:border-yellow-800 p-3 rounded">
                    <p className="font-medium text-yellow-800 dark:text-yellow-200 mb-1">API Key Instructions:</p>
                    <p className="text-yellow-700 dark:text-yellow-300">1. Go to <a href="https://console.anthropic.com/settings/keys" target="_blank" rel="noopener noreferrer" className="underline">Anthropic Console</a></p>
                    <p className="text-yellow-700 dark:text-yellow-300">2. Create a new API key</p>
                    <p className="text-yellow-700 dark:text-yellow-300">3. Copy the API key</p>
                    <p className="text-yellow-700 dark:text-yellow-300">4. Paste it above and click "Set API Key"</p>
                  </div>
                </>
              )}
            </div>
          )}
        </CardContent>
      </Card>
    );
  }

  // Fallback for unknown auth response types
  return (
    <Card>
      <CardContent className="pt-6">
        <pre className="text-sm">{JSON.stringify(data, null, 2)}</pre>
      </CardContent>
    </Card>
  );
}