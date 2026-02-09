# Browser Tool

Control a web browser for automated interactions. Provides session-isolated browser contexts with navigation, element interaction, and content extraction capabilities.

## IMPORTANT: Action Batching

The Browser tool supports executing multiple actions in a single call using the `action` action type with an `actions` array. This is more efficient than making separate tool calls.

## Action Types

### 1. navigate (action="open" | "go_back" | "go_forward")

Navigate to a URL, or go forward/back in browser history.

CRITICAL WARNINGS - DATA LOSS:
- You should NEVER use this tool for scrolling to the top of the page
- Navigate WILL RESET DATA ON THE PAGE - any form data, JavaScript state, or loaded content will be lost
- Navigating WILL ALSO CLEAR THE DOM OF WHATEVER PAGE YOU WERE PREVIOUSLY ON
- Workaround: If you need to preserve data that's been loaded on the page, use `scroll` within the action tool instead
- Before navigating: If you need to preserve data, write it to files first

go_back / go_forward Behavior:
- Navigates through browser history (like clicking browser back/forward buttons)
- Returns the URL navigated to
- WARNING: May trigger page reload and clear current DOM state (same data loss as navigate)
- Clears element cache on navigation

FILE PREVIEW: You can use file:// URLs to preview files like HTML or PDFs. (Note that it will appear as sandbox:// in the browser, this is expected)

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

Use Cases:
- Understand page structure and find interactive elements
- Discover what elements are available for interaction
- Get element references (refs) for precise targeting

Two Modes:
- Without filter: Shows all visible elements in view - use this to understand current page content and structure
- With 'interactive' filter: Shows only buttons, links, and inputs with their coordinates - use this to find what you can click or interact with

IMPORTANT Limitations:
- Only searches elements currently loaded in the DOM (visible viewport + any lazy-loaded content so far)
- Does NOT include content that hasn't been loaded yet (below fold, lazy-loaded sections)
- For pages with lazy-loading, you must scroll to load more content before it appears in read_page results

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

Click Actions (use `index` parameter):
* `left_click`: Click the left mouse button. Optional: `duration` for click-and-hold, `repeat` for multiple clicks.
* `right_click`: Right-click to open context menus. Optional: `duration` for click-and-hold.
* `double_click`: Double-click the left mouse button.
* `triple_click`: Triple-click the left mouse button (select paragraph).

Click Action Guidance:
- PRIMARY preference: Use screenshot coordinates for clicking when possible
- Use element refs only as a fallback when coordinates are difficult to determine
- When clicking an element, consult the screenshot and aim for the center of the element, not edges
- If a click doesn't register, adjust coordinates so the cursor tip is clearly on the target element
- Staleness warning: Page UI changes can make older screenshot coordinates stale — take a fresh screenshot if needed

Keyboard Actions:
* `type`: Type text using keyboard. Requires `text` (string). Best for simple text input where you need to simulate actual typing.
* `key`: Press keyboard key(s). Requires `key` with space-separated keys (e.g., "Enter", "Backspace Backspace", "cmd+a"). Use for navigation (Tab, Enter), shortcuts (Cmd+A, Ctrl+C), special keys (Escape, Backspace).

Form Actions:
* `form_input`: Set form element value directly (more reliable for React apps than typing). Requires `index` and `value`. For checkboxes use boolean, for selects use option text/value, for inputs use string/number. Preferred over `type` for React apps because it directly sets the value in the DOM, bypassing synthetic event handling issues.

Scroll Actions:
* `scroll`: Scroll page in a direction. Requires `direction` ("up"/"down"/"left"/"right"). Optional: `scroll_amount` (pixels, default 100). Use for general page navigation.
* `scroll_to`: Scroll element into view. Requires `index`. Use when you need to ensure an element is visible before interacting with it.

Scroll Action Tips:
- Can batch scroll with screenshot to see results mid-scroll (e.g., `[scroll, screenshot, scroll, screenshot]`)
- Useful for lazy-loading pages where content loads as you scroll

Screenshot Action:
* `screenshot`: Capture the current page state. Use this to inspect visual content mid-sequence (e.g., after scrolling to check what's visible before continuing). Optional: `file_path` to save to disk.

Screenshot Behavior:
- Automatically caches element mappings for subsequent actions
- Returns warning if page appears blank or not fully loaded
- Element cache is cleared when page is blank
- Coordinates become stale if page UI changes - take fresh screenshot if needed

Other Actions:
* `wait`: Wait for specified duration. Requires `duration` (milliseconds, 1-150000). Use when waiting for page animations, transitions, or async operations to complete.
* `left_click_drag`: Drag from one point to another. Two modes: (1) Index mode: `fromIndex` and `toIndex`, or (2) Coordinate mode: `fromX`, `fromY`, `toX`, `toY`. Optional: `duration` controls drag speed (default 500ms). Use coordinate mode for precise drag operations, index mode when dragging between known elements.

Parameters:
- `action`: "action"
- `actions`: Array of sub-action objects
- `tabId`: Tab ID to operate on (required)

Usage Guidelines:
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

* For click actions, prefer using screenshot coordinates. Use element refs only as a fallback when coordinates are difficult to determine or hard to find.
* When clicking an element, consult the screenshot to determine coordinates. Aim for the center of the element, not the edges.
* If a click doesn't register, try adjusting coordinates so the cursor tip is clearly on the target element.*: Page UI changes can make older screenshot coordinates stale — take a fresh screenshot if needed.

### 5. find (action="find")

Find all elements in the DOM (not just the visible viewport) that match a natural language query.

IMPORTANT LIMITATIONS:
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

When to use: This tool is ideal for immediately grabbing all textual content on a page without needing to scroll and take multiple screenshots. Great for articles, blog posts, documentation, or any text-heavy page where you need to read the full content quickly.

Limitations: This tool only extracts content currently loaded in the DOM. For pages with lazy-loading or infinite scroll, content at the bottom may not be loaded yet. In those cases, you'll need to use action sequences with scroll + screenshot to load and view additional content.

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

Create a new browser tab and optionally navigate to a URL.

Returns: The new tab's ID (e.g., "tab-2") which you MUST use for all subsequent operations on this tab.

Use Cases:
- Open multiple pages simultaneously
- Keep reference pages open while working in another tab
- Separate different workflows or accounts

Important:
- You must track and use the returned tab ID for all future operations on this tab
- Each tab has its own navigation history and DOM state
- All tabs share the same browser session (cookies, local storage, login state)

Parameters:
- `action`: "tab_create"
- `url`: (optional) URL to navigate to in new tab

Example:
```json
{"action": "tab_create", "url": "https://example.com"}
```

### 8. list_tabs (action="tab_list")

List all open browser tabs with their URLs and titles.

Use Cases:
- Track available tabs when managing multiple browser windows
- Find tab IDs for switching or closing operations
- Verify which tabs are currently open

Returns:
- List of all tabs with their IDs, URLs, and titles
- Indicates which tab is currently active
- Total tab count

Parameters:
- `action`: "tab_list"

Example:
```json
{"action": "tab_list"}
```

## Other Actions

The Browser tool also supports these individual actions for compatibility. All actions except `tab_create`, `tab_list`, and `close` require the `tabId` parameter.

Actions requiring `tabId`:
- `screenshot`: Capture page screenshot. Automatically caches element mappings. Returns warning if page is blank. Element cache cleared on blank pages.
- `click`: Click element by index
- `type`: Type text into element
- `right_click`, `double_click`, `triple_click`: Click variants
- `scroll`: Scroll page by direction and amount
- `key`: Press keyboard keys (e.g., "Enter", "cmd+a")
- `scroll_to`: Scroll element into view by index
- `drag`: Drag and drop operations (two modes: index or coordinate)
- `form_input`: Set form values directly (preferred for React apps)
- `wait`: Pause execution (use for animations, transitions, async operations)
- `tab_switch`: Switch to different tab. Important: Must take a screenshot after switching to interact with the tab
- `tab_close`: Close a tab. Clears element cache for that tab. Cannot close the last remaining tab

Actions NOT requiring `tabId`:
- `tab_create`: Create new tab (returns new tab ID)
- `tab_list`: List all tabs
- `close`: Close entire browser session. Clears all element caches. Cannot be recovered - session must be recreated

See individual action documentation in the full tool description for parameter details.

## Usage Patterns

1. For click actions, prefer using screenshot coordinates. Use element refs only as a fallback when coordinates are difficult to determine or hard to find."
2. When clicking an element, consult the screenshot to determine coordinates. Aim for the center of the element, not the edges."

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

## Choosing Between Approaches

When you have multiple ways to accomplish the same task, use these guidelines to choose the best approach:

### Click Actions: Coordinates vs Refs vs Index

Preference Hierarchy:
1. Screenshot coordinates (PRIMARY) - Most reliable, directly targets visual position
2. Element refs (FALLBACK) - Use when coordinates are difficult to determine
3. Element index (FALLBACK) - Use when neither coordinates nor refs are available

Decision Criteria:
- Use coordinates when: Element is clearly visible in screenshot, you can identify center point
- Use refs when: Element has stable ref from read_page, coordinate calculation is complex
- Use index when: Working with read_page results, element isn't visually identifiable

Tips:
- Always aim for the center of elements, not edges
- If clicks fail, take a fresh screenshot (coordinates may be stale)
- Verify cursor position is clearly on target element

### Text Input: form_input vs type

When to use `form_input`:
- React apps or modern frameworks (more reliable)
- Directly setting values (checkboxes, selects, inputs)
- Form fields that need specific values without typing simulation
- When you need immediate value setting without events

When to use `type`:
- Simple text input requiring typing simulation
- When onChange events must fire naturally
- Legacy forms without framework-specific handling

Why form_input is better for React:
- Directly sets DOM value, bypassing synthetic event issues
- More reliable for controlled components
- Avoids typing race conditions

### Scrolling: scroll vs scroll_to

When to use `scroll`:
- General page navigation (up/down/left/right)
- Loading more content on lazy-loading pages
- Moving viewport by specific pixel amounts
- Exploring page content incrementally

When to use `scroll_to`:
- Ensuring an element is visible before interaction
- Jumping directly to specific content
- Requires element index from read_page

Tip: Batch scroll with screenshot to see results: `[scroll, screenshot, scroll, screenshot]`

### Content Extraction: get_page_text vs Multiple Screenshots

When to use `get_page_text`:
- Articles, blog posts, documentation
- Text-heavy pages where you need full content
- Quick extraction without scrolling
- When text formatting doesn't matter

When to use multiple screenshots:
- Visual layout matters
- Need to see images, buttons, or UI elements
- Interactive elements are important
- Page has complex structure

Limitation: Both only extract content currently loaded in DOM - lazy-loaded content requires scrolling first

### Finding Elements: read_page vs find

When to use `read_page`:
- Understanding page structure
- Getting all visible elements
- Need element indices for immediate interaction
- Want to see everything in viewport

When to use `find`:
- Searching for specific elements by purpose or text
- Element might be outside viewport
- Natural language query works better than visual scanning
- Need to locate elements across entire loaded DOM

Important: Both only search currently loaded DOM - scroll to load more content first

## Framework-Specific Optimizations

### React Applications

Form Handling:
- Always prefer `form_input` over `type` for React form fields
- React's synthetic events can cause issues with keyboard typing
- `form_input` directly sets the value, bypassing event handling complexity
- Use boolean for checkboxes, text/value for selects, string/number for inputs

Why This Matters:
- Controlled components in React manage their own state
- Direct value setting is more reliable than simulated typing
- Avoids race conditions between typing and React state updates

### Single Page Applications (SPAs)

Navigation Caution:
- SPAs often use client-side routing (doesn't trigger page reload)
- Clicking navigation links may not clear DOM like traditional navigation
- Data persists in JavaScript state across "page" changes
- Element cache remains valid within same SPA view

When to Clear Cache:
- After navigate/go_back/go_forward (full page load)
- When page appears blank or unloaded
- After major UI state changes

### Lazy Loading & Infinite Scroll

Understanding DOM Limitations:
- `read_page`, `find`, and `get_page_text` only see currently loaded DOM
- Content below the fold may not be loaded yet
- Scrolling triggers lazy loading, expanding the DOM

Strategy:
- Use action sequences: `[scroll, screenshot, scroll, screenshot]` to progressively load content
- Take screenshots to verify content loaded before searching
- For infinite scroll: scroll in increments, checking for new content each time

### Form Handling Best Practices

Checkboxes:
- Use `form_input` with boolean value: `{"type": "form_input", "index": 5, "value": true}`
- Don't use click for toggling (unreliable state)

Dropdowns/Selects:
- Use `form_input` with option text or value: `{"type": "form_input", "index": 3, "value": "Option 2"}`
- More reliable than clicking to open dropdown

Text Inputs:
- React apps: Use `form_input` for direct value setting
- Legacy forms: Either approach works, `type` may be more natural

## Best Practices

### Efficiency & Performance

Batch Operations:
- Use action sequences to combine multiple operations in one call
- Reduces total number of tool calls and round-trip time
- Example: `[click, wait, type, key]` instead of 4 separate calls

Take Screenshots Strategically:
- After state changes to verify results
- After scrolling to see what's loaded
- Before and after critical actions (form submission, navigation)
- Can interleave screenshots in action sequences: `[scroll, screenshot, scroll, screenshot]`

Minimize Round Trips:
- Use `get_page_text` instead of scrolling + multiple screenshots for text extraction
- Use `find` to search entire loaded DOM instead of repeated read_page calls
- Batch multiple independent actions together

### State Management

Element Cache Behavior:
- Cached per tab: `sessionID_tabID`
- Cleared on: navigation, blank pages, tab close, session close
- Visual indices may become stale after DOM changes

When to Refresh Cache:
- After any page navigation (open, go_back, go_forward)
- When clicks/interactions fail (coordinates may be stale)
- After significant DOM changes (modals opening, content loading)
- When error message says "no element cache found"

Cache Staleness:
- Screenshot coordinates become stale when page UI changes
- Take fresh screenshot if interactions fail
- Element refs (fX_ref_Y) remain valid until page reload

### Error Handling & Recovery

Common Error Patterns:

1. "No element cache found for this tab"
   - Cause: No screenshot taken yet, or cache was cleared
   - Fix: Take a screenshot first

2. "Index N not found in cache"
   - Cause: Element indices changed (DOM updated)
   - Fix: Take fresh screenshot or use read_page again

3. "Invalid ref format"
   - Cause: Ref doesn't follow fX_ref_Y pattern
   - Fix: Use proper ref from read_page output

4. Click doesn't register
   - Cause: Stale coordinates, or clicking edge instead of center
   - Fix: Take fresh screenshot, aim for element center

Recovery Strategies:
- Always verify critical actions with screenshots
- For failed interactions: refresh page state with screenshot, retry
- For "element not found" errors: use read_page or find to relocate element
- For blank pages: take screenshot to verify, clear cache automatically happens

### Data Preservation

Before Navigation:
- Extract any data you need from current page (use get_page_text, read_page, or screenshots)
- Write important data to files
- Remember: navigate/go_back/go_forward WILL CLEAR DOM AND DATA

Workaround for Scrolling to Top:
- Don't use navigate - use `scroll` with direction "up" repeatedly
- Or use `scroll_to` with index 0 if element exists
- This preserves loaded data and DOM state

Multi-Tab Strategy:
- Keep reference material in separate tabs
- Switch tabs instead of navigating to preserve state
- Remember: all tabs share session cookies and login state

## Important Notes

- Session Isolation: Each session gets its own browser context
- Element Indices: Element indices from `read_page` are stable until the page changes
- Auto-Reconnection: Connection failures are handled transparently
- Permissions: Only `open` action requires user permission
- Shared Session State: All tabs share cookies, local storage, and login state
- Tab Isolation: Each tab has its own navigation history and DOM state
- Cache Management: Element caches are tab-specific and cleared on navigation
