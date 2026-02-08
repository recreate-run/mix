# Browser Tool

Control a web browser for automated interactions. Provides session-isolated browser contexts with navigation, element interaction, and content extraction capabilities.

## IMPORTANT: Action Batching

The Browser tool supports executing multiple actions in a single call using the `action` action type with an `actions` array. This is more efficient than making separate tool calls.

## Action Types

### 1. navigate (action="open" | "go_back" | "go_forward")

Navigate to a URL, or go forward/back in browser history.

**IMPORTANT**: You should NEVER use this tool for scrolling to the top of the page. If you need to scroll, use `scroll` or `scroll_to` actions instead. Navigate WILL RESET DATA ON THE PAGE, so if you need to preserve data that's been loaded, use scrolling instead.

**IMPORTANT**: Navigating WILL ALSO CLEAR THE DOM. If you need to preserve the DOM, write data to files first.

**FILE PREVIEW**: You can use file:// URLs to preview files like HTML or PDFs. (Note that it will appear as sandbox:// in the browser, this is expected)

Parameters:
- `action`: "open" | "go_back" | "go_forward"
- `url`: (for open) URL to navigate to. Supports http://, https://, file:// schemes. File URLs must be within session storage directory.
- `tabId`: Tab ID to operate on (required)

Examples:
```json
{"action": "open", "url": "https://example.com", "tabId": "tab-1"}
{"action": "go_back", "tabId": "tab-1"}
{"action": "go_forward", "tabId": "tab-1"}
```

### 2. read_page (action="read_page")

Get an accessibility tree representation of elements CURRENTLY VISIBLE in the viewport (content in the DOM that's below/above the visible viewport is NOT included). Optionally filter for only interactive elements.

**Without filter**: Shows all visible elements in view - use this to understand current page content and structure.

**With 'interactive' filter**: Shows only buttons, links, and inputs with their coordinates - use this to find what you can click or interact with.

Coordinates returned are in screenshot coordinates (always within screenshot bounds since only viewport is shown).

Elements are referenced by their ref ID (e.g., f0_ref_1, f1_ref_2). The 'f' prefix indicates the frame:
- f0_ref_* = main frame
- f1_ref_*, f2_ref_* = iframes (numbered by depth-first traversal order)

Parameters:
- `action`: "read_page"
- `interactiveOnly`: (optional) boolean, default false
- `tabId`: Tab ID to operate on (required)

Example:
```json
{"action": "read_page", "interactiveOnly": true, "tabId": "tab-1"}
```

### 3. file_upload (action="upload")

Upload a file from the workspace to a browser file input (<input type='file'>).

Parameters:
- `action`: "upload"
- `index`: Element index of file input
- `filePath`: Path to file (absolute or session-relative)
- `tabId`: Tab ID to operate on (required)

Example:
```json
{"action": "upload", "index": 5, "filePath": "document.pdf", "tabId": "tab-1"}
```

### 4. action (action="action")

Execute a sequence of browser actions one by one.

Each action in the `actions` array can be one of the following:

**Click Actions** (use `index` parameter):
* `left_click`: Click the left mouse button. Optional: `duration` for click-and-hold, `repeat` for multiple clicks.
* `right_click`: Right-click to open context menus. Optional: `duration` for click-and-hold.
* `double_click`: Double-click the left mouse button.
* `triple_click`: Triple-click the left mouse button (select paragraph).

**Keyboard Actions**:
* `type`: Type text using keyboard. Requires `text` (string).
* `key`: Press keyboard key(s). Requires `key` with space-separated keys (e.g., "Enter", "Backspace Backspace", "cmd+a").

**Form Actions**:
* `form_input`: Set form element value directly (more reliable for React apps than typing). Requires `index` and `value`. For checkboxes use boolean, for selects use option text/value, for inputs use string/number.

**Scroll Actions**:
* `scroll`: Scroll at coordinates. Requires `direction` ("up"/"down"/"left"/"right"). Optional: `scroll_amount` (pixels, default 100).
* `scroll_to`: Scroll element into view. Requires `index`.

**Screenshot Action**:
* `screenshot`: Capture the current page state. Use this to inspect visual content mid-sequence (e.g., after scrolling to check what's visible before continuing). Optional: `file_path` to save to disk.

**Other Actions**:
* `wait`: Wait for seconds. Requires `duration` (milliseconds, 1-150000).
* `left_click_drag`: Drag from one point to another. Requires `fromIndex` and `toIndex`, or `fromX`, `fromY`, `toX`, `toY`.

Parameters:
- `action`: "action"
- `actions`: Array of sub-action objects
- `tabId`: Tab ID to operate on (required)

**Usage Guidelines**:
* IMPORTANT: Verify actions were executed correctly before performing irreversible actions (e.g. check form fields before submitting, review email content before sending).
* IMPORTANT: You can interleave `screenshot` actions between other actions when you need to see the page state mid-sequence (e.g., `[scroll, screenshot, scroll, screenshot]` to read content while scrolling through a lazy-loading page).
* If you include any `screenshot` actions, no automatic screenshot is added at the end.
* It's okay to chain multiple actions together if they are safe, there will automatically be a short delay between them while executing.

Example:
```json
{
  "action": "action",
  "tabId": "tab-1",
  "actions": [
    {"type": "left_click", "index": 5},
    {"type": "wait", "duration": 500},
    {"type": "type", "text": "search query"},
    {"type": "key", "key": "Enter"},
    {"type": "screenshot"}
  ]
}
```

### 5. find (action="find")

Find all elements in the DOM (not just the visible viewport) that match a natural language query.

**IMPORTANT LIMITATIONS**:
- ONLY searches elements currently loaded in the DOM (visible viewport + any lazy-loaded content so far). Scrolling to load more content will change the results.
- Does NOT find content that hasn't loaded yet (e.g., items beyond current total page height on infinite scroll/lazy-loading pages)
- Coordinates are relative to current viewport (0,0 = top-left of visible screen). Elements with negative or out-of-bounds coordinates are OFF-SCREEN and require scrolling to view before interaction

Usage:
- Search by purpose: "search bar", "login button", "add to cart". The more specific and descriptive the query is, the better the results will be.
- Search by text: "organic mango product", "cottage in name"
- Provide detailed context to help the tool understand the query more clearly

OUTPUT FORMAT:
- If <= 100 matching elements: Results are returned directly in the tool response
- If > 100 matching elements: Results are written to a file. The response will include the file path - use the read tool to view full results if needed

Parameters:
- `action`: "find"
- `query`: Natural language search query
- `tabId`: Tab ID to operate on (required)

Example:
```json
{"action": "find", "query": "button Submit", "tabId": "tab-1"}
```

### 6. get_page_text (action="get_text")

Extract all text content from the page in one call.

**When to use**: This tool is ideal for immediately grabbing all textual content on a page without needing to scroll and take multiple screenshots. Great for articles, blog posts, documentation, or any text-heavy page where you need to read the full content quickly.

**Limitations**: This tool only extracts content currently loaded in the DOM. For pages with lazy-loading or infinite scroll, content at the bottom may not be loaded yet. In those cases, you'll need to use action sequences with scroll + screenshot to load and view additional content.

Returns plain text without HTML formatting, prioritizing main content areas (article, main, content sections).

Parameters:
- `action`: "get_text"
- `strategy`: (optional) "auto" | "article" | "main" | "body", default "auto"
- `tabId`: Tab ID to operate on (required)

Example:
```json
{"action": "get_text", "strategy": "auto", "tabId": "tab-1"}
```

### 7. create_tab (action="tab_create")

Create a new browser tab and optionally navigate to a URL. Returns the new tab's ID (e.g., "tab-2") which you MUST use for subsequent operations on this tab.

Parameters:
- `action`: "tab_create"
- `url`: (optional) URL to navigate to in new tab

Example:
```json
{"action": "tab_create", "url": "https://example.com"}
```

### 8. list_tabs (action="tab_list")

List all open browser tabs with their URLs and titles.

Parameters:
- `action`: "tab_list"

Example:
```json
{"action": "tab_list"}
```

## Other Actions

The Browser tool also supports these individual actions for compatibility. **All actions except `tab_create`, `tab_list`, and `close` require the `tabId` parameter.**

Actions requiring `tabId`:
- `screenshot`: Capture page screenshot
- `click`: Click element by index
- `type`: Type text into element
- `right_click`, `double_click`, `triple_click`: Click variants
- `scroll`: Scroll page by direction and amount
- `key`: Press keyboard keys (e.g., "Enter", "cmd+a")
- `scroll_to`: Scroll element into view by index
- `drag`: Drag and drop operations
- `form_input`: Set form values directly
- `wait`: Pause execution
- `tab_switch`: Switch to different tab
- `tab_close`: Close a tab

Actions NOT requiring `tabId`:
- `tab_create`: Create new tab (returns new tab ID)
- `tab_list`: List all tabs
- `close`: Close entire browser

See individual action documentation in the full tool description for parameter details.

## Usage Patterns

### Navigate and Explore
1. Create a tab: `{"action": "tab_create", "url": "..."}` → Returns `{"id": "tab-1", ...}`
2. Open a URL: `{"action": "open", "url": "...", "tabId": "tab-1"}`
3. Use `read_page` to discover elements: `{"action": "read_page", "tabId": "tab-1"}`
4. Interact with elements using indices from `read_page`

### Form Filling with Action Sequences
```json
{
  "action": "action",
  "tabId": "tab-1",
  "actions": [
    {"type": "left_click", "index": 2},
    {"type": "type", "text": "user@example.com"},
    {"type": "left_click", "index": 3},
    {"type": "type", "text": "password123"},
    {"type": "key", "key": "Enter"}
  ]
}
```

### Multi-Tab Workflows
1. Create first tab: `{"action": "tab_create", "url": "https://example.com"}` → Returns `{"id": "tab-1", ...}`
2. Create second tab: `{"action": "tab_create", "url": "https://other.com"}` → Returns `{"id": "tab-2", ...}`
3. Navigate tab: `{"action": "open", "url": "...", "tabId": "tab-2"}`
4. Switch between tabs: `{"action": "tab_switch", "tabId": "tab-1"}`
5. All tab operations REQUIRE tabId parameter

## Important Notes

- Session Isolation: Each session gets its own browser context
- Element Indices: Element indices from `read_page` are stable until the page changes
- Auto-Reconnection: Connection failures are handled transparently
- Permissions: Only `open` action requires user permission
