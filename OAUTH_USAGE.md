# Using the OAuth API

This document explains how to use the OAuth authentication API endpoints.

## Authentication Flow

1. **Start OAuth Flow**

```http
POST /api/auth/oauth/anthropic
```

Response:
```json
{
  "auth_url": "https://claude.ai/oauth/authorize?client_id=...",
  "state": "...",
  "message": "Open the auth_url in your browser to complete OAuth authentication"
}
```

2. **Complete OAuth Flow**

After authorizing in your browser, you'll receive an authorization code.
Use this code with our callback endpoint:

```http
POST /api/auth/oauth-callback
Content-Type: application/json

{
  "provider": "anthropic",
  "code": "YOUR_AUTHORIZATION_CODE",
  "state": "STATE_FROM_FIRST_RESPONSE"
}
```

Response:
```json
{
  "status": "success",
  "provider": "anthropic",
  "message": "OAuth authentication successful",
  "expires_in": 3600
}
```

## Example JavaScript Implementation

```javascript
// Step 1: Start OAuth flow
async function startOAuthFlow() {
  const response = await fetch('http://localhost:8088/api/auth/oauth/anthropic', {
    method: 'POST',
  });
  
  const data = await response.json();
  
  // Store the state for later use
  localStorage.setItem('oauth_state', data.state);
  
  // Open authorization URL in a new window
  window.open(data.auth_url, '_blank');
}

// Step 2: Complete OAuth flow with the code
async function completeOAuthFlow(authorizationCode) {
  const state = localStorage.getItem('oauth_state');
  
  const response = await fetch('http://localhost:8088/api/auth/oauth-callback', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      provider: 'anthropic',
      code: authorizationCode,
      state: state
    })
  });
  
  const result = await response.json();
  
  if (result.status === 'success') {
    console.log('OAuth authentication successful!');
    // Update UI to show authenticated state
  }
}

// Usage in UI
document.getElementById('oauth-button').addEventListener('click', startOAuthFlow);

// Add a form to enter the authorization code
document.getElementById('code-form').addEventListener('submit', (e) => {
  e.preventDefault();
  const code = document.getElementById('auth-code').value;
  completeOAuthFlow(code);
});
```

## Flow Diagram

```
Client              API Server          Anthropic OAuth
  |                     |                     |
  | Start OAuth Flow    |                     |
  |-------------------->|                     |
  |                     |                     |
  | Returns auth_url    |                     |
  |<--------------------|                     |
  |                     |                     |
  | User visits auth_url|                     |
  |-------------------------------------------->|
  |                     |                     |
  | Authorization       |                     |
  |<--------------------------------------------|
  |                     |                     |
  | Submit code via     |                     |
  | oauth-callback      |                     |
  |-------------------->|                     |
  |                     | Exchange for tokens |
  |                     |-------------------->|
  |                     |                     |
  |                     | Returns tokens      |
  |                     |<--------------------|
  |                     |                     |
  | Success response    |                     |
  |<--------------------|                     |
  |                     |                     |
```

## Notes

- The `state` parameter is crucial for security - it prevents CSRF attacks
- The authorization code expires quickly, so submit it immediately
- OAuth tokens are automatically refreshed by the backend when needed
- You can check authentication status using `/api/auth/status` endpoint