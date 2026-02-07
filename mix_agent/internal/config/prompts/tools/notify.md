Request user attention and input during workflow execution by displaying a notification that interrupts the user interface.

## Purpose

Use this tool when you need:
- User to solve a CAPTCHA or complete authentication challenge
- Credentials, API keys, or authentication tokens that must be provided securely
- Two-factor authentication (2FA) codes or one-time passwords
- Critical decision-making input from the user mid-workflow
- User confirmation before proceeding with destructive or irreversible actions
- Text input that cannot be obtained from files or environment

## When NOT to Use

- **Do not spam** notifications - use sparingly for truly interactive moments
- Do not use for routine status updates (use regular responses instead)
- Do not use when information is available through other tools
- Do not use for optional confirmations (proceed with reasonable defaults)

## Response Types

### `acknowledge`
Simple notification requiring only user dismissal. No input captured.

**Use for:**
- Important warnings or alerts
- Completion confirmations
- Status notifications requiring acknowledgment

**Example:**
```json
{
  "type": "warning",
  "title": "Process Completed",
  "message": "Database migration completed successfully. Please verify the results before continuing.",
  "responseType": "acknowledge"
}
```

### `text`
Free-form text input from user.

**Use for:**
- CAPTCHA codes
- API keys or tokens
- Passwords or credentials
- 2FA codes
- Custom text input

**Example:**
```json
{
  "type": "question",
  "title": "CAPTCHA Required",
  "message": "Please enter the 6-character CAPTCHA code displayed in the browser screenshot.",
  "responseType": "text"
}
```

### `choice`
Multiple-choice selection. Requires `choices` array with at least 2 options.

**Use for:**
- Decision points with predefined options
- Configuration selections
- Approval/rejection workflows
- Multiple strategy choices

**Example:**
```json
{
  "type": "question",
  "title": "Deployment Environment",
  "message": "Which environment should I deploy to?",
  "responseType": "choice",
  "choices": ["development", "staging", "production"]
}
```

## Notification Types

- `info`: Informational notifications (blue)
- `warning`: Caution or important notices (yellow)
- `error`: Error conditions requiring attention (red)
- `question`: Questions requiring user input (purple)

## Timeout Behavior

- **Default:** 60 seconds
- **Range:** 1-300 seconds
- **On timeout:** Tool execution fails with error
- User has countdown timer showing remaining time
- Choose timeout based on expected response complexity:
  - CAPTCHA: 30-60 seconds
  - Reading and decision-making: 60-120 seconds
  - Complex input: 120-180 seconds

## Best Practices

1. **Clear, specific titles** - Max 100 characters, descriptive
2. **Detailed messages** - Explain exactly what you need and why
3. **Appropriate timeouts** - Give users enough time without blocking forever
4. **Fallback handling** - Handle timeout errors gracefully in your workflow
5. **Context provision** - Reference screenshots, files, or URLs when helpful

## Response Handling

The tool returns:
- **Acknowledge:** Confirmation that user saw the notification
- **Text:** User's text input in the response
- **Choice:** The option the user selected

## Security Considerations

- Sensitive input (passwords, API keys) is transmitted over authenticated session only
- Do not log or store sensitive user input in plain text responses
- Use appropriate notification types to signal sensitivity (use `question` type for auth)

## Examples

### CAPTCHA Solving
```json
{
  "type": "question",
  "title": "CAPTCHA Required",
  "message": "A CAPTCHA verification is required to proceed. Please view the screenshot at [URL] and enter the code you see.",
  "responseType": "text",
  "timeout": 60
}
```

### 2FA Code Request
```json
{
  "type": "question",
  "title": "Two-Factor Authentication",
  "message": "Please enter the 6-digit code from your authenticator app to continue login.",
  "responseType": "text",
  "timeout": 90
}
```

### Critical Action Confirmation
```json
{
  "type": "warning",
  "title": "Confirm Database Deletion",
  "message": "You are about to delete the 'production_backup' database. This action is irreversible. Do you want to proceed?",
  "responseType": "choice",
  "choices": ["Cancel", "Proceed with deletion"],
  "timeout": 120
}
```

### API Key Request
```json
{
  "type": "question",
  "title": "API Key Required",
  "message": "Please provide your OpenAI API key to enable GPT-4 integration. This will be used for the current session only.",
  "responseType": "text",
  "timeout": 90
}
```

### Deployment Strategy Selection
```json
{
  "type": "info",
  "title": "Choose Deployment Strategy",
  "message": "The application can be deployed using blue-green deployment or rolling update. Blue-green provides instant rollback but requires double resources. Rolling update uses existing resources but has gradual rollback.",
  "responseType": "choice",
  "choices": ["Blue-green deployment", "Rolling update", "Cancel deployment"],
  "timeout": 120
}
```

## Error Handling

If the notification times out:
```
Notification request timed out - user did not respond in time
```

Handle this gracefully in your workflow:
- Retry with extended timeout if appropriate
- Proceed with safe defaults if possible
- Report timeout to user and explain next steps
- Do not retry repeatedly (avoid spam)
