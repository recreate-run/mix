import { useState } from 'react';
import { Settings } from 'lucide-react';
import { CommandEmpty, CommandGroup, CommandItem } from '@/components/ui/command';
import type { AuthMethod, LoginProvider } from '@/types/command-slash';
import { BackButton } from './shared/BackButton';

interface AuthInputViewProps {
  selectedProvider: LoginProvider;
  selectedAuthMethod: AuthMethod | null;
  onBackToProviders: () => void;
  onBackToAuthMethods: () => void;
  onAuthMethodSelect: (method: AuthMethod) => void;
  onOAuthStart: (providerId: string) => Promise<void>;
  onApiKeySubmit: (providerId: string, apiKey: string) => Promise<void>;
  onOAuthCodeSubmit: (providerId: string, code: string) => Promise<void>;
}

export function AuthInputView({
  selectedProvider,
  selectedAuthMethod,
  onBackToProviders,
  onBackToAuthMethods,
  onAuthMethodSelect,
  onOAuthStart,
  onApiKeySubmit,
  onOAuthCodeSubmit,
}: AuthInputViewProps) {
  const [apiKey, setApiKey] = useState('');
  const [oauthCode, setOauthCode] = useState('');
  const [apiKeySubmitting, setApiKeySubmitting] = useState(false);
  const [oauthCodeSubmitting, setOauthCodeSubmitting] = useState(false);

  // If we have both provider and auth method selected, show the input form
  if (selectedProvider && selectedAuthMethod) {
    const handleApiKeySubmit = async () => {
      if (!apiKey || apiKeySubmitting) return;

      try {
        setApiKeySubmitting(true);
        await onApiKeySubmit(selectedProvider.id, apiKey);
      } catch (error) {
        console.error('API key submission failed:', error);
      } finally {
        setApiKeySubmitting(false);
      }
    };

    const handleOAuthCodeSubmit = async () => {
      if (!oauthCode || oauthCodeSubmitting) return;

      try {
        setOauthCodeSubmitting(true);
        await onOAuthCodeSubmit(selectedProvider.id, oauthCode.trim());
      } catch (error) {
        console.error('OAuth code submission failed:', error);
      } finally {
        setOauthCodeSubmitting(false);
      }
    };

    const handleOAuthStart = async () => {
      try {
        await onOAuthStart(selectedProvider.id);
        // Switch to code input mode after starting OAuth
        setTimeout(() => {
          onAuthMethodSelect('oauth_code');
        }, 500);
      } catch (error) {
        console.error('OAuth start failed:', error);
      }
    };

    return (
      <CommandGroup heading={`Enter ${selectedAuthMethod === 'api_key' ? 'API Key' : 'OAuth Code'}`}>
        <BackButton
          label="Back to Auth Methods"
          onSelect={onBackToAuthMethods}
          value="back-to-auth-methods"
        />

        <div>
          {selectedAuthMethod === 'api_key' && (
            <div className="p-2 border-b">
              <div className="font-medium text-sm mb-1">
                Enter API Key for {selectedProvider.displayName}:
              </div>
              <div className="text-muted-foreground text-xs mb-2">
                Format: {selectedProvider.apiKeyFormat || "API key"}
              </div>
              <div className="flex gap-2">
                <input
                  type="password"
                  value={apiKey}
                  onChange={(e) => setApiKey(e.target.value)}
                  placeholder={selectedProvider.apiKeyFormat || "Enter API key"}
                  disabled={apiKeySubmitting}
                  className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && apiKey) {
                      e.preventDefault();
                      handleApiKeySubmit();
                    }
                  }}
                />
                <button
                  onClick={handleApiKeySubmit}
                  disabled={!apiKey || apiKeySubmitting}
                  className={`rounded-md px-3 py-2 text-sm font-medium ${
                    !apiKey || apiKeySubmitting
                      ? 'bg-muted text-muted-foreground'
                      : 'bg-primary text-primary-foreground'
                  }`}
                >
                  {apiKeySubmitting ? 'Submitting...' : 'Submit'}
                </button>
              </div>
            </div>
          )}

          {selectedAuthMethod === 'oauth_code' && (
            <div className="p-3 border-b">
              <div className="font-medium text-sm mb-1">
                Enter OAuth code for {selectedProvider.displayName}:
              </div>
              <div className="text-muted-foreground text-xs mb-2">
                After authorizing in your browser, paste the code here
              </div>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={oauthCode}
                  onChange={(e) => setOauthCode(e.target.value)}
                  placeholder="Authorization code"
                  disabled={oauthCodeSubmitting}
                  className="flex-1 rounded-md border border-input bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary"
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && oauthCode) {
                      e.preventDefault();
                      e.stopPropagation();
                      handleOAuthCodeSubmit();
                    }
                  }}
                />
                <button
                  onClick={(e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    handleOAuthCodeSubmit();
                  }}
                  disabled={!oauthCode || oauthCodeSubmitting}
                  className={`rounded-md px-3 py-2 text-sm font-medium ${
                    !oauthCode || oauthCodeSubmitting
                      ? 'bg-muted text-muted-foreground'
                      : 'bg-primary text-primary-foreground'
                  }`}
                >
                  {oauthCodeSubmitting ? 'Submitting...' : 'Submit'}
                </button>
              </div>
            </div>
          )}

          {selectedAuthMethod === 'oauth' && (
            <div className="p-3 border-b">
              <div className="font-medium text-sm mb-1">
                Start OAuth flow for {selectedProvider.displayName}:
              </div>
              <div className="text-muted-foreground text-xs mb-2">
                You'll be redirected to authorize in your browser
              </div>
              <button
                onClick={(e) => {
                  e.stopPropagation();
                  e.preventDefault();
                  handleOAuthStart();
                }}
                className="w-full rounded-md px-3 py-2 text-sm font-medium bg-primary text-primary-foreground"
              >
                Start OAuth Authorization
              </button>
            </div>
          )}
        </div>
      </CommandGroup>
    );
  }

  // If we have provider but no auth method, show auth method selection
  if (selectedProvider) {
    if (!selectedProvider.authMethods.length) {
      return <CommandEmpty>No auth methods available</CommandEmpty>;
    }

    return (
      <CommandGroup heading={`Authentication Methods for ${selectedProvider.displayName}`}>
          <BackButton
            label="Back to Providers"
            onSelect={onBackToProviders}
            value="back-to-providers"
          />

          {selectedProvider.authMethods.map((method) => (
            <CommandItem
              key={method}
              onSelect={() => onAuthMethodSelect(method as AuthMethod)}
              value={method === 'api_key' ? 'API Key' : 'OAuth'}
            >
              <Settings className="size-4 text-muted-foreground" />
              <div className="flex-1">
                <div className="flex items-center gap-2 font-medium text-sm">
                  {method === 'api_key' ? 'API Key' : 'OAuth'}
                </div>
                <div className="text-muted-foreground text-xs">
                  {method === 'api_key'
                    ? 'Enter your API key directly'
                    : 'Connect through web authorization'}
                </div>
              </div>
            </CommandItem>
          ))}
      </CommandGroup>
    );
  }

  return <CommandEmpty>No provider selected</CommandEmpty>;
}