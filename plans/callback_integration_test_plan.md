# Post-Tool Callback Integration Test Plan

## Overview
This test plan validates the post-tool execution callback system. The callbacks allow running automated actions (bash scripts or sub-agents) after specific tool executions complete.

## Test Objective
Verify that the callback system:
1. Initializes correctly with the agent
2. Detects tools that implement the CallbackTool interface
3. Executes bash script callbacks successfully
4. Logs all callback execution stages
5. Runs callbacks non-blocking (async) without affecting main agent flow

## Test Setup

### Prerequisites
- Mix agent running with dev server (`make dev`)
- Access to UI or CLI
- A CSV file with portfolio data to analyze

### Test Callback Configuration
The `show_media` tool has been configured with a test callback that:
- Logs execution details to `callback_test.log` in the session directory
- Captures tool name, session ID, tool ID, and result
- Runs asynchronously (non-blocking)
- Has a 5-second timeout

**File**: `mix_agent/internal/llm/tools/show_media.go:188-204`

## Test Case: Portfolio Analysis with Show Media

### Test Prompt
```
Look at my portfolio in the data in @{file_info.url} and find the top winners and losers in Q4. Show the three most relevant plots.
```

### Expected Flow
1. **Agent receives prompt** with CSV file attachment
2. **Agent analyzes data** using Python/data analysis tools
3. **Agent generates plots** (images)
4. **Agent calls `show_media` tool** to display the plots
5. **Callback triggers** after `show_media` execution
6. **Callback logs** execution details to `callback_test.log`

### Step-by-Step Test Instructions

#### Step 1: Start the Dev Server
```bash
cd /Users/vaibhavagarwal/Documents/recreate/mix
make dev
```

#### Step 2: Monitor Logs in Real-Time
Open a new terminal and run:
```bash
make tail-log
```

Look for these log patterns (all prefixed with `[CALLBACKS]`):

**1. Callback Executor Initialization:**
```
[CALLBACKS] Callback executor initialized for agent agentName=main
```

**2. Tool Callback Detection:**
```
[CALLBACKS] Tool has callbacks registered tool=show_media callbackCount=1 sessionID=<session-id>
```

**3. Callback Execution Start:**
```
[CALLBACKS] Executing callback tool=show_media callbackIndex=0 callbackType=bash_script nonBlocking=true
[CALLBACKS] Starting async callback execution tool=show_media callbackIndex=0
```

**4. Bash Execution Logs:**
```
[CALLBACKS] Executing bash callback tool=show_media sessionID=<session-id> command=<bash-command>
[CALLBACKS] Environment variables set for callback CALLBACK_TOOL_NAME=show_media
[CALLBACKS] Executing bash command command=<bash-command>
[CALLBACKS] Bash callback execution completed exitCode=0 interrupted=false
```

**5. Callback Completion:**
```
[CALLBACKS] Async callback completed successfully tool=show_media callbackIndex=0 outputLength=<length>
```

#### Step 3: Execute Test in UI

1. **Upload your portfolio CSV** using the attachment feature
2. **Send the test prompt**:
   ```
   Look at my portfolio in the data in @{file_info.url} and find the top winners and losers in Q4. Show the three most relevant plots.
   ```
3. **Wait for agent to complete** the analysis and show plots

#### Step 4: Verify Callback Execution

**Option A: Check the log file** (Real-time)
```bash
# Find your session directory (look in logs for sessionID)
# Session directories are typically in:
cd ~/.mix_storage/sessions/<session-id>

# View the callback log
cat callback_test.log
```

**Expected content in `callback_test.log`:**
```
=== SHOW_MEDIA CALLBACK EXECUTED ===
Timestamp: Sat Oct 11 09:25:30 PDT 2025
Tool: show_media
Session: <session-id>
Tool ID: <tool-call-id>
Result (first 200 chars): Successfully showcasing 3 media output(s): Top Winners Q4, Top Losers Q4, Q4 Performance Distribution
----------------------------------------
```

**Option B: Search logs for callback markers**
```bash
# From the project root
grep -r "\[CALLBACKS\]" dev.log | tail -50
```

#### Step 5: Verify No Blocking
- Check that the agent continues processing after `show_media` is called
- The callback should run asynchronously in the background
- Agent response should not be delayed by the callback execution

## Success Criteria

✅ **PASS if all of the following are true:**

1. **Initialization**: Callback executor logs show it was initialized
2. **Detection**: Logs show `show_media` tool has callbacks registered
3. **Execution**: Logs show callback execution started and completed
4. **File Creation**: `callback_test.log` exists in session directory
5. **Content Verification**: Log file contains expected callback execution details
6. **Non-Blocking**: Agent completes response without waiting for callback
7. **No Errors**: No callback-related errors in dev.log

❌ **FAIL if any of:**
- No `[CALLBACKS]` logs appear
- Callback executor not initialized
- `callback_test.log` file not created
- Callback execution errors in logs
- Agent hangs waiting for callback to complete

## Debugging Tips

### If callback doesn't execute:

1. **Check callback executor initialization:**
   ```bash
   grep "Callback executor initialized" dev.log
   ```

2. **Verify show_media was called:**
   ```bash
   grep "show_media" dev.log
   ```

3. **Check for callback registration:**
   ```bash
   grep "Tool has callbacks registered" dev.log
   ```

4. **Look for errors:**
   ```bash
   grep -i "error.*callback" dev.log
   ```

### If callback file not found:

1. **Find session storage directory:**
   ```bash
   # Look for this in logs:
   grep "SessionStorageDir" dev.log

   # Or check default location:
   ls -la ~/.mix_storage/sessions/
   ```

2. **Check file permissions:**
   ```bash
   ls -la ~/.mix_storage/sessions/<session-id>/
   ```

## Advanced Testing

### Test Blocking Callback
Modify `show_media.go:202` to set `NonBlocking: false` and verify:
- Agent waits for callback to complete
- Callback completes before agent sends response
- Logs show synchronous execution

### Test Callback Failure
Modify the bash command to cause an error:
```go
BashCommand: "exit 1",
```
Verify:
- Error is logged
- Agent continues normally
- Error doesn't crash the system

### Test Multiple Callbacks
Add multiple callbacks to the return array:
```go
return []interfaces.CallbackConfig{
    {Type: interfaces.CallbackTypeBashScript, BashCommand: "echo 'First callback'", ...},
    {Type: interfaces.CallbackTypeBashScript, BashCommand: "echo 'Second callback'", ...},
}
```
Verify:
- Both callbacks execute
- Both are logged separately with callbackIndex 0 and 1

## Cleanup

After testing, you can disable the test callback by modifying `show_media.go:188`:
```go
func (t *mediaShowcaseTool) GetCallbacks() []interfaces.CallbackConfig {
    return []interfaces.CallbackConfig{}  // Disabled
}
```

Or keep it enabled for production use with your custom callback logic.

## Environment Variables Reference

These environment variables are available to bash callbacks:

| Variable | Description | Example |
|----------|-------------|---------|
| `CALLBACK_TOOL_RESULT` | Full tool output/result | "Successfully showcasing 3 media output(s): ..." |
| `CALLBACK_TOOL_NAME` | Name of the tool that executed | "show_media" |
| `CALLBACK_TOOL_ID` | Unique ID for this tool call | "toolu_01ABC..." |
| `CALLBACK_SESSION_ID` | Current session ID | "550e8400-e29b-41d4-a716-446655440000" |

## Next Steps

After successful testing:
1. Customize the callback command for your use case
2. Add callbacks to other tools as needed
3. Consider implementing sub-agent callbacks (requires architecture refactoring)
4. Add callback configuration to database/files for dynamic behavior

## Questions or Issues?

If the test fails or you encounter issues:
1. Check dev.log for detailed error messages
2. Verify session storage directory exists and is writable
3. Ensure the dev server restarted after code changes
4. Look for `[CALLBACKS]` prefixed logs for the execution flow
