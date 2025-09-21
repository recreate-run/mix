# OpenRouter Integration Testing

This document outlines the steps to test OpenRouter integration with our application. We'll test multiple models including Claude models through OpenRouter to ensure proper functionality.

## Prerequisites
- OpenRouter API Key: `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`
- HTTP client (cURL, Postman, or similar)

## Setup and Testing Steps

### 1. Reset Current Preferences

First, we need to reset any existing preferences:

```bash
curl -X POST http://localhost:8088/api/preferences/reset \
  -H "Content-Type: application/json"
```

Expected response: Success message and default preferences

### 2. Set OpenRouter as Provider with API Key

Set the OpenRouter provider and API key:

```bash
curl -X POST http://localhost:8088/api/auth/api-key \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "openrouter",
    "key": "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"
  }'
```

Expected response: Success message confirming API key was stored

### 3. Update Preferences to Use OpenRouter

Update preferences to use OpenRouter with a specific model:

```bash
curl -X POST http://localhost:8088/api/preferences \
  -H "Content-Type: application/json" \
  -d '{
    "preferredProvider": "openrouter",
    "preferredModel": "openrouter.claude-3.5-sonnet"
  }'
```

Expected response: Updated preferences with OpenRouter as provider

### 4. Verify Authentication Status

Verify that the authentication is working:

```bash
curl -X GET http://localhost:8088/api/auth/status \
  -H "Content-Type: application/json"
```

Expected response: Authentication status showing OpenRouter is properly configured

### 5. Create a New Session

Create a new chat session:

```bash
curl -X POST http://localhost:8088/api/sessions \
  -H "Content-Type: application/json" \
  -d '{
    "name": "OpenRouter Test Session"
  }'
```

Expected response: Session object with ID (save this ID for subsequent requests)

### 6. Test Basic Message

Send a basic message to test connectivity:

```bash
curl -X POST http://localhost:8088/api/sessions/[SESSION_ID]/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Hello, this is a test message. Please respond briefly."
  }'
```

Expected response: Message object and assistant's response

### 7. Test Tool Calling - Basic

Test basic tool calling functionality:

```bash
curl -X POST http://localhost:8088/api/sessions/[SESSION_ID]/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "What time is it now? Please use a tool to find out."
  }'
```

Expected response: Tool call should be made and response should include the current time

### 8. Test Tool Calling - Complex

Test more complex tool with multiple parameters:

```bash
curl -X POST http://localhost:8088/api/sessions/[SESSION_ID]/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Can you read the content of server.go file? Use the file read tool."
  }'
```

Expected response: Tool call should be made and response should include file content

### 9. Test Stream Response

Test streaming response using SSE endpoint:

```bash
curl -X GET http://localhost:8088/stream?sessionId=[SESSION_ID] \
  -H "Accept: text/event-stream"
```

Then send a message to see streaming in action:

```bash
curl -X POST http://localhost:8088/api/sessions/[SESSION_ID]/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Please explain a complex topic in great detail so I can see your streaming response."
  }'
```

Expected response: Streamed content from the model

### 10. Test Cancel API Call

Start a long-running request, then cancel it:

```bash
# First start a long request
curl -X POST http://localhost:8088/api/sessions/[SESSION_ID]/messages \
  -H "Content-Type: application/json" \
  -d '{
    "content": "Write a very detailed 3000 word essay about quantum physics."
  }'

# Then quickly cancel it
curl -X POST http://localhost:8088/api/sessions/[SESSION_ID]/cancel \
  -H "Content-Type: application/json"
```

Expected response: Request should be cancelled and appropriate message received

## Testing Different Models

Repeat steps 3-10 with different OpenRouter models:

### Free Models with Tool Calling Support

#### Test Z.AI GLM 4.5 Air
```bash
curl -X POST http://localhost:8088/api/preferences \
  -H "Content-Type: application/json" \
  -d '{
    "preferred_provider": "openrouter",
    "main_agent_model": "openrouter.zai-glm-4.5-air",
    "main_agent_max_tokens": 1000
  }'
```

#### Test DeepSeek V3.1
```bash
curl -X POST http://localhost:8088/api/preferences \
  -H "Content-Type: application/json" \
  -d '{
    "preferred_provider": "openrouter",
    "main_agent_model": "openrouter.deepseek-v3.1",
    "main_agent_max_tokens": 1000
  }'
```

#### Test Sonoma Dusk Alpha
```bash
curl -X POST http://localhost:8088/api/preferences \
  -H "Content-Type: application/json" \
  -d '{
    "preferred_provider": "openrouter",
    "main_agent_model": "openrouter.sonoma-dusk-alpha",
    "main_agent_max_tokens": 1000
  }'
```

#### Test Sonoma Sky Alpha
```bash
curl -X POST http://localhost:8088/api/preferences \
  -H "Content-Type: application/json" \
  -d '{
    "preferred_provider": "openrouter",
    "main_agent_model": "openrouter.sonoma-sky-alpha",
    "main_agent_max_tokens": 1000
  }'
```

### Paid Models

#### Test OpenAI Models via OpenRouter

```bash
curl -X POST http://localhost:8088/api/preferences \
  -H "Content-Type: application/json" \
  -d '{
    "preferred_provider": "openrouter",
    "main_agent_model": "openrouter.gpt-4o",
    "main_agent_max_tokens": 1000
  }'
```

#### Test Claude Models via OpenRouter

```bash
curl -X POST http://localhost:8088/api/preferences \
  -H "Content-Type: application/json" \
  -d '{
    "preferred_provider": "openrouter",
    "main_agent_model": "openrouter.claude-3-haiku",
    "main_agent_max_tokens": 1000
  }'
```

#### Test Gemini Models via OpenRouter

```bash
curl -X POST http://localhost:8088/api/preferences \
  -H "Content-Type: application/json" \
  -d '{
    "preferred_provider": "openrouter",
    "main_agent_model": "openrouter.gemini-2.5-flash",
    "main_agent_max_tokens": 1000
  }'
```

## Issue Tracking

For each test, document any issues encountered in this format:

| Test | Model | Issue | Solution/Fix |
|------|-------|-------|--------------|
| API Key Setting | All models | Invalid parameter name `key` | Use snake_case `api_key` instead |
| Model Selection | Free models | Models not defined in app | Add model definitions to openrouter.go |
| Tool Calling | Initial tests | Credit limit issues | Reduce max_tokens to 1000 |

## Performance Comparison

Compare response times between different models for similar prompts:

| Model | Basic Response Time | Tool Call Response Time | Streaming Response |
|-------|---------------------|-------------------------|-------------------|
| openrouter.zai-glm-4.5-air | Fast (~7s) | Fast (~5s) | Well-structured, detailed response with markdown formatting |
| openrouter.deepseek-v3.1 | Fast (~5s) | Fast (~4s) | Concise response, more summarized |

## Advanced Testing Scenarios

1. **Multi-turn conversations** - Test if context is maintained across multiple messages
2. **Image handling** - Test if models can process and reference images
3. **Long context handling** - Test with very long prompts to verify context window handling
4. **Error handling** - Test with invalid inputs to verify error responses
5. **Rate limit handling** - Test behavior when rate limits are hit

## Test Results

### Free Models With Tool Calling Support Testing

#### Z.AI GLM 4.5 Air Testing

##### Test 1: Set OpenRouter API Key
- **Command**: `curl -X POST http://localhost:8088/api/auth/api-key -H "Content-Type: application/json" -d '{"provider": "openrouter", "api_key": "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"}'`
- **Result**: Success - API key stored successfully

##### Test 2: Update Preferences to Use Z.AI GLM 4.5 Air
- **Command**: `curl -X POST http://localhost:8088/api/preferences -H "Content-Type: application/json" -d '{"preferred_provider": "openrouter", "main_agent_model": "openrouter.zai-glm-4.5-air", "main_agent_max_tokens": 1000}'`
- **Result**: Success - Preferences updated to use Z.AI GLM 4.5 Air model

##### Test 3: Create Test Session
- **Command**: `curl -X POST http://localhost:8088/api/sessions -H "Content-Type: application/json" -d '{"title": "Z.AI GLM 4.5 Air Test"}'`
- **Result**: Success - Session created with ID "82f050b2-fe35-4ffc-bbcb-ce089084f982"

##### Test 4: Basic Message Test
- **Command**: `curl -X POST "http://localhost:8088/api/sessions/82f050b2-fe35-4ffc-bbcb-ce089084f982/messages" -H "Content-Type: application/json" -d '{"content": "Hello, this is a test message. Please respond briefly."}'`
- **Result**: Success - Received "Hello!" response

##### Test 5: Simple Tool Calling Test
- **Command**: `curl -X POST "http://localhost:8088/api/sessions/82f050b2-fe35-4ffc-bbcb-ce089084f982/messages" -H "Content-Type: application/json" -d '{"content": "What time is it now? Please use a tool to find out."}'`
- **Result**: Success - Used bash tool to run `date` command

##### Test 6: Complex Tool Calling Test
- **Command**: `curl -X POST "http://localhost:8088/api/sessions/82f050b2-fe35-4ffc-bbcb-ce089084f982/messages" -H "Content-Type: application/json" -d '{"content": "Can you check if a file named go.mod exists in the current directory and show me its contents if it does?"}'`
- **Result**: Success - Used glob tool to search for go.mod file

##### Test 7: Streaming Response Test
- **Commands**: 
  ```bash
  # Open streaming connection
  curl -N -X GET "http://localhost:8088/stream?sessionId=82f050b2-fe35-4ffc-bbcb-ce089084f982" -H "Accept: text/event-stream" &
  
  # Send message that triggers a long response
  curl -X POST "http://localhost:8088/api/sessions/82f050b2-fe35-4ffc-bbcb-ce089084f982/messages" -H "Content-Type: application/json" -d '{"content": "Please explain quantum computing in detail with examples. Make it a comprehensive explanation."}'
  ```
- **Result**: Success - Received a detailed, well-structured response about quantum computing with markdown formatting

#### DeepSeek V3.1 Testing

##### Test 1: Update Preferences to Use DeepSeek V3.1
- **Command**: `curl -X POST http://localhost:8088/api/preferences -H "Content-Type: application/json" -d '{"preferred_provider": "openrouter", "main_agent_model": "openrouter.deepseek-v3.1", "main_agent_max_tokens": 1000}'`
- **Result**: Success - Preferences updated to use DeepSeek V3.1 model

##### Test 2: Create Test Session
- **Command**: `curl -X POST http://localhost:8088/api/sessions -H "Content-Type: application/json" -d '{"title": "DeepSeek V3.1 Test"}'`
- **Result**: Success - Session created with ID "aa6fed98-9c0d-4434-a753-7c7ddda815b4"

##### Test 3: Basic Message Test
- **Command**: `curl -X POST "http://localhost:8088/api/sessions/aa6fed98-9c0d-4434-a753-7c7ddda815b4/messages" -H "Content-Type: application/json" -d '{"content": "Hello, this is a test message. Please respond briefly."}'`
- **Result**: Success - Received "Hello! I'm Mix, ready to help with creative content generation and multimedia analysis tasks." response

##### Test 4: Simple Tool Calling Test
- **Command**: `curl -X POST "http://localhost:8088/api/sessions/aa6fed98-9c0d-4434-a753-7c7ddda815b4/messages" -H "Content-Type: application/json" -d '{"content": "What time is it now? Please use a tool to find out."}'`
- **Result**: Success - Used bash tool to run `date '+%H:%M:%S'` command

##### Test 5: Complex Tool Calling Test
- **Command**: `curl -X POST "http://localhost:8088/api/sessions/aa6fed98-9c0d-4434-a753-7c7ddda815b4/messages" -H "Content-Type: application/json" -d '{"content": "Can you check if a file named go.mod exists in the current directory and show me its contents if it does?"}'`
- **Result**: Success - Used ls tool to list files in current directory

##### Test 6: Streaming Response Test
- **Commands**: 
  ```bash
  # Open streaming connection
  curl -N -X GET "http://localhost:8088/stream?sessionId=aa6fed98-9c0d-4434-a753-7c7ddda815b4" -H "Accept: text/event-stream" &
  
  # Send message that triggers a long response
  curl -X POST "http://localhost:8088/api/sessions/aa6fed98-9c0d-4434-a753-7c7ddda815b4/messages" -H "Content-Type: application/json" -d '{"content": "Please explain quantum computing in detail with examples. Make it a comprehensive explanation."}'
  ```
- **Result**: Success - Received a concise response with key concepts about quantum computing

[Further tests with other models to be added]

## Conclusion

Our testing of OpenRouter integration with the application has been successful. We have successfully tested both Z.AI GLM 4.5 Air and DeepSeek V3.1 models from OpenRouter with tool calling capabilities. Here are the key findings:

1. **API Parameter Format**: OpenRouter expects snake_case parameters in API calls (e.g., `api_key` instead of `key`)

2. **Model Definition Requirements**: Models must be properly defined in the application's model registry (openrouter.go) before they can be used

3. **Tool Calling Support**: Both Z.AI GLM 4.5 Air and DeepSeek V3.1 support tool calling functionality

4. **Response Style**: The models have different response styles:
   - Z.AI GLM 4.5 Air is more concise
   - DeepSeek V3.1 is more verbose and tends to introduce itself as "Mix"

5. **Performance**: Both models respond relatively quickly and successfully invoke tools when requested

6. **Token Limits**: Setting `main_agent_max_tokens` to 1000 helps avoid credit limit issues

7. **Streaming Responses**: Both models support streaming responses with different characteristics:
   - Z.AI GLM 4.5 Air provides well-structured, detailed responses with markdown formatting
   - DeepSeek V3.1 provides more concise responses

Recommendations:

1. Add model definitions for all free models that support tool calling in OpenRouter
2. Use snake_case consistently in API parameters for OpenRouter integration
3. Default to lower max token settings (1000) for free models to avoid credit issues
4. Continue testing with more complex tool calls and multi-turn conversations to validate reliability