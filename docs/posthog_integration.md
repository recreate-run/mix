# PostHog Analytics Integration

This document describes the analytics events tracked by the Mix application using PostHog.

## Configuration

Analytics tracking is configured via environment variables:

1. Copy `.env.example` to `.env` to enable analytics
2. Set the `POSTHOG_API_KEY` environment variable with your PostHog API key:

```bash
POSTHOG_API_KEY=your_posthog_api_key
```

If the API key is not provided, analytics tracking will be disabled.

## Events Tracked

The application tracks the following events:

### 1. User Messages (`user_message`)

Triggered when a user sends a message/prompt.

**Properties:**
- `session_id`: Unique identifier for the user's session
- `message_id`: Unique identifier for the specific message
- `content`: The text content of the user's message (truncated to 10,000 chars if longer)
- `content_length`: Original length of the content before truncation
- `is_truncated`: Boolean indicating if the content was truncated
- `model`: The AI model used for the conversation (e.g., "claude-4-sonnet")

### 2. Assistant Responses (`agent_response`)

Triggered when the assistant generates a response.

**Properties:**
- `session_id`: Unique identifier for the user's session
- `message_id`: Unique identifier for the specific message
- `content`: The text content of the assistant's response (truncated to 10,000 chars if longer)
- `content_length`: Original length of the content before truncation
- `is_truncated`: Boolean indicating if the content was truncated
- `model`: The AI model used to generate the response

### 3. Tool Calls (`tool_call`)

Triggered when the assistant uses a tool or when a tool call completes.

**Properties:**
- `session_id`: Unique identifier for the user's session
- `message_id`: Unique identifier for the message containing the tool call
- `tool_name`: Name of the tool being called (e.g., "Bash", "Read", "Write")
- `tool_input`: Input parameters for the tool call
- `tool_id`: Unique identifier for the specific tool call
- `success`: Boolean indicating whether the tool call succeeded
- `error`: Error message if the tool call failed (empty if successful)

## Implementation Details

The analytics integration is implemented through:

1. **Analytics Service**: A dedicated service that wraps the PostHog client and provides methods for tracking events.
2. **Message Service Wrapper**: A wrapper around the message service that adds tracking to message creation and updates.

The implementation follows these principles:
- Analytics are non-blocking and don't affect app performance
- Failures in analytics tracking are logged but don't impact the application
- No user identification data is collected other than anonymous session IDs
- Tool usage patterns are tracked for improving user experience

## Viewing Analytics Data

To view the analytics data:

1. Log in to your PostHog instance
2. Go to the Insights section
3. Create charts and dashboards based on the events described above

## Future Enhancements

Potential future enhancements for analytics:

- Track timing data for tool operations
- Track user session length and engagement metrics
- Add custom user properties with opt-in user demographic information
- Create specific event funnels to track user journeys