import { AlertCircle, CheckCircle, ExternalLink, Loader2 } from 'lucide-react';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Card, CardContent } from '@/components/ui/card';
import { Input } from '@/components/ui/input';
import { rpcCall } from '@/lib/rpc';

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
  const [isLoading, setIsLoading] = useState(false);
  const [showSuccess, setShowSuccess] = useState(false);

  const handleSubmit = async () => {
    const input = authMode === 'code' ? authCode.trim() : apiKey.trim();
    if (!input) return;

    setIsLoading(true);
    try {
      const result = await rpcCall<{ status: string; step?: string }>(
        'auth.login',
        authMode === 'code'
          ? { authCode: input, manual: input.startsWith('sk-ant-') }
          : { apiKey: input }
      );

      if (result.status === 'success') {
        setShowSuccess(true);
        setTimeout(() => window.location.reload(), 2000);
      } else if (result.step === 'manual_fallback') {
        setAuthMode('apikey');
      }
    } catch (error) {
      const errorMsg =
        error instanceof Error ? error.message : 'Authentication failed';
      if (
        errorMsg.includes('Cloudflare') ||
        errorMsg.includes('manual token')
      ) {
        setAuthMode('apikey');
      }
    } finally {
      setIsLoading(false);
      setAuthCode('');
      setApiKey('');
    }
  };

  if (data.type === 'error') {
    const errorData = data as ErrorResponse;
    return (
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2 text-red-600">
            <AlertCircle className="h-4 w-4" />
            <span>{errorData.error}</span>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (data.type === 'message') {
    const messageData = data as MessageResponse;
    return (
      <Card>
        <CardContent className="p-4">
          <p>{messageData.message}</p>
        </CardContent>
      </Card>
    );
  }

  if (data.type === 'auth_status') {
    const statusData = data as AuthStatusResponse;
    const isAuthenticated = statusData.status === 'authenticated';

    return (
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2">
            {isAuthenticated ? (
              <CheckCircle className="h-4 w-4 text-green-600" />
            ) : (
              <AlertCircle className="h-4 w-4 text-yellow-600" />
            )}
            <span>{statusData.message}</span>
          </div>
        </CardContent>
      </Card>
    );
  }

  // Show success state for RPC authentication
  if (showSuccess) {
    return (
      <Card>
        <CardContent className="p-4">
          <div className="flex items-center gap-2 text-green-600">
            <CheckCircle className="h-4 w-4" />
            <span>✅ Authentication successful! Reloading...</span>
          </div>
        </CardContent>
      </Card>
    );
  }

  if (data.type === 'auth_login') {
    const loginData = data as AuthLoginResponse;

    // Handle success state for slash command
    if (loginData.status === 'success') {
      return (
        <Card>
          <CardContent className="p-4">
            <div className="flex items-center gap-2 text-green-600">
              <CheckCircle className="h-4 w-4" />
              <span>✅ Authentication successful! Reloading...</span>
            </div>
          </CardContent>
        </Card>
      );
    }

    return (
      <Card>
        <CardContent className="space-y-4 p-4">
          <p>{loginData.message}</p>

          {loginData.authUrl && loginData.status === 'pending' && (
            <>
              <div className="flex gap-2">
                <Button
                  onClick={() => setAuthMode('code')}
                  size="sm"
                  variant={authMode === 'code' ? 'default' : 'outline'}
                >
                  OAuth
                </Button>
                <Button
                  onClick={() => setAuthMode('apikey')}
                  size="sm"
                  variant={authMode === 'apikey' ? 'default' : 'outline'}
                >
                  API Key
                </Button>
              </div>

              {authMode === 'code' ? (
                <div className="space-y-3">
                  <Button
                    className="w-full"
                    onClick={() => window.open(loginData.authUrl, '_blank')}
                    variant="outline"
                  >
                    <ExternalLink className="mr-2 h-4 w-4" />
                    Authenticate
                  </Button>
                  <div className="flex gap-2">
                    <Input
                      disabled={isLoading}
                      onChange={(e) => setAuthCode(e.target.value)}
                      onKeyDown={(e) => e.key === 'Enter' && handleSubmit()}
                      placeholder="Paste authorization code..."
                      value={authCode}
                    />
                    <Button
                      disabled={!authCode.trim() || isLoading}
                      onClick={handleSubmit}
                      variant="outline"
                    >
                      {isLoading ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        'Submit'
                      )}
                    </Button>
                  </div>
                </div>
              ) : (
                <div className="flex gap-2">
                  <Input
                    disabled={isLoading}
                    onChange={(e) => setApiKey(e.target.value)}
                    onKeyDown={(e) => e.key === 'Enter' && handleSubmit()}
                    placeholder="sk-ant-..."
                    type="password"
                    value={apiKey}
                  />
                  <Button
                    disabled={!apiKey.trim() || isLoading}
                    onClick={handleSubmit}
                    variant="outline"
                  >
                    {isLoading ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      'Submit'
                    )}
                  </Button>
                </div>
              )}
            </>
          )}
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardContent className="p-4">
        <pre className="text-sm">{JSON.stringify(data, null, 2)}</pre>
      </CardContent>
    </Card>
  );
}
