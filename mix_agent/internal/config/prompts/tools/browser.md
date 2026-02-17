# Browser Tool

Control a web browser for automated interactions. Provides session-isolated browser contexts with navigation, element interaction, and content extraction capabilities.

## IMPORTANT: Action Sequences

The Browser tool supports executing multiple operations in a single call using the `sequence` action with an `actions` array. This is more efficient than making separate tool calls.

## Operations

### 1. navigate (action="open" | "go_back" | "go_forward")

Navigate to a URL, or go forward/back in browser history.

CRITICAL WARNINGS - DATA LOSS:

- You should NEVER use this tool for scrolling to the top of the page
- Navigate WILL RESET DATA ON THE PAGE - any form data, JavaScript state, or loaded content will be lost
- Navigating WILL ALSO CLEAR THE DOM OF WHATEVER PAGE YOU WERE PREVIOUSLY ON
- Workaround: If you need to preserve data that's been loaded on the page, use `scroll` within a sequence instead
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

Filter Options (optional):

- No filter: All visible elements in the viewport — use to understand full page structure
- `"interactive"`: Buttons, links, and inputs with coordinates — use to find clickable elements
- `"links"`: Only anchor/link elements — use for link extraction tasks
- `"buttons"`: Only button elements — use to find actions on a page
- `"text"`: Only text-content elements (paragraphs, headings, labels) — use for full-page text extraction
- `"headings"`: Only heading elements (h1–h6) — use for page structure overview

IMPORTANT Limitations:

- Only searches elements currently loaded in the DOM (visible viewport + any lazy-loaded content so far)
- Does NOT include content that hasn't been loaded yet (below fold, lazy-loaded sections)
- For pages with lazy-loading, you must scroll to load more content before it appears in read_page results

Coordinates shown are element centers in screenshot space (ready for clicking, no offset calculation needed).

Elements are referenced by their ref ID (e.g., f0_ref_1, f1_ref_2). The 'f' prefix indicates the frame:

- f0_ref_* = main frame
- f1_ref_*, f2_ref_* = iframes (numbered by depth-first traversal order)

Attributes:

Elements include HTML attributes when available: href (link URLs), id, type, placeholder, name, aria-label. Example: `- link "Product" [ref=f0_ref_25] (x=400,y=200) href="/dp/B08ABC123"` shows a link with its URL in the href attribute.

Parameters:

- `action`: "read_page"
- `filter`: (optional) "interactive" | "links" | "buttons" | "text" | "headings" — default: all elements
- `tabId`: Tab ID to operate on (required)

Examples:

```json
{"action": "read_page", "tabId": "tab-1"}
{"action": "read_page", "filter": "interactive", "tabId": "tab-1"}
{"action": "read_page", "filter": "links", "tabId": "tab-1"}
{"action": "read_page", "filter": "text", "tabId": "tab-1"}
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

### 4. sequence (action="sequence")

Execute a sequence of browser operations one by one.

Each operation in the `actions` array can be one of the following:

Click Actions (use `coordinate` [x, y] or `ref` parameter):

- `left_click`: Click the left mouse button. Optional: `duration` for click-and-hold, `repeat` for multiple clicks.
- `right_click`: Right-click to open context menus. Optional: `duration` for click-and-hold.
- `double_click`: Double-click the left mouse button.
- `triple_click`: Triple-click the left mouse button (select paragraph).

Keyboard Actions:

- `type`: Type text and press keys. Requires `text` (string). Use `{key}` syntax for special keys: `"text{Enter}"`, `"{cmd+a}"`, `"{Backspace}{Delete}"`. Escape literal braces with `{{` or `}}`.

  Supported keys: Navigation (Enter, Tab, Escape, Space, Arrows), Editing (Backspace, Delete, Insert), Navigation (Home, End, PageUp, PageDown), Function (F1-F12), Modifiers (cmd+a, ctrl+c, shift+Tab, alt+F4).

  Usage modes:
  • With index: Clicks element at index, then processes keyboard input (use when element not focused)
  • Without index: Processes keyboard input into currently focused element (use after clicking the element)

  Examples: `"user@example.com{Tab}password{Enter}"` (login form), `"{cmd+a}{Delete}new text"` (replace all), `"code: {{example}}"` (literal braces).

Form Actions:

- `form_input`: Set form element value directly (more reliable for React apps than typing). Requires `index` and `value`. For checkboxes use boolean, for selects use option text/value, for inputs use string/number. Preferred over `type` for React apps because it directly sets the value in the DOM, bypassing synthetic event handling issues.

Scroll Actions:

- `scroll`: Scroll page in a direction. Requires `direction` ("up"/"down"/"left"/"right"). Optional: `scroll_amount` (pixels, default 100). Use for general page navigation.
- `scroll_to`: Scroll element into view. Requires `index`. Use when you need to ensure an element is visible before interacting with it.

Scroll Action Tips:

- Useful for lazy-loading pages where content loads as you scroll
- Use standalone analyze_screenshot after scrolling to verify loaded content

Other Actions:

- `wait`: Wait for specified duration. Requires `duration` (milliseconds, 1-150000). Use when waiting for page animations, transitions, or async operations to complete.
- `left_click_drag`: Drag from one point to another. Two modes: (1) Index mode: `fromIndex` and `toIndex`, or (2) Coordinate mode: `fromX`, `fromY`, `toX`, `toY`. Optional: `duration` controls drag speed (default 500ms). Use coordinate mode for precise drag operations, index mode when dragging between known elements.

Parameters:

- `action`: "sequence"
- `actions`: Array of operations
- `tabId`: Tab ID to operate on (required)

Usage Guidelines:

- IMPORTANT: Verify operations were executed correctly before performing irreversible actions (e.g. check form fields before submitting, review email content before sending).
- It's okay to chain multiple operations together if they are safe, there will automatically be a short delay between them while executing.
- To check page state mid-workflow, use standalone analyze_screenshot between sequences.

Example:

```json
{
  "action": "sequence",
  "tabId": "tab-1",
  "actions": [
    {"action": "left_click", "coordinate": [450, 300]},
    {"action": "wait", "duration": 500},
    {"action": "type", "text": "search query{Enter}"}
  ]
}
```

### Standalone Usage

All operations documented above can be called **standalone** OR **sequenced**:
- Standalone: `{"action": "left_click", "coordinate": [x, y], "tabId": "tab-1"}`
- Sequenced: `{"action": "sequence", "actions": [{"action": "left_click", "coordinate": [x, y]}], "tabId": "tab-1"}`

When calling standalone:

- Operations requiring element targeting (`form_input`, `scroll_to`) need `tabId` + targeting parameter
- `type` needs `tabId` + optional `index` (omit index to type into focused element)
- Operations with `coordinate` or `ref` parameters (`left_click`, `right_click`, etc.) need `tabId`
- Operations like `scroll`, `wait` need `tabId`
- `tab_create`, `tab_list`, and `close` do NOT require `tabId`

Prefer sequences for multiple operations.

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

### 6. create_tab (action="tab_create")

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

### 7. list_tabs (action="tab_list")

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

### 8. analyze_screenshot (action="analyze_screenshot")

Captures the current page state and performs image analysis or element location detection.

Two Modes:

1. Text Analysis Mode (default):
   - Returns natural language description of page content
   - Use for understanding page layout, verifying page state, extracting information
   - Example prompts: "What's on this page?", "Describe the navigation menu"

2. Bounding Box Mode (auto-detected):
   - Gives Coordinates of the element centers computed from the bounding box
   - Triggered by keywords: "bounding box", "coordinates", "box_2d" in prompt
   - Use for precise element location and visual targeting
   - Example prompts: "Give me bounding box coordinates for the submit button"

Parameters:

- `action`: "analyze_screenshot"
- `prompt`: Analysis instruction for Gemini (required)
- `tabId`: Tab ID to analyze (required)

Examples:

Text Analysis:

```json
{
  "action": "analyze_screenshot",
  "prompt": "Describe the main navigation menu and its options",
  "tabId": "tab-1"
}
```

Bounding Box Detection:

```json
{
  "action": "analyze_screenshot",
  "prompt": "Give me the bounding box coordinates for the 'Sign In' button",
  "tabId": "tab-1"
}
```

Output Formats:

Text Mode (default):

```
The page shows a login form with two input fields (email and password) and a blue submit button below...
```

Bounding Box Mode (read_page style):

```
- button (x=408,y=70)
```

## Usage Patterns

### Navigate and Explore

1. Create a tab: `{"action": "tab_create", "url": "..."}` → Returns `{"id": "tab-1", ...}`
2. Open a URL: `{"action": "open", "url": "...", "tabId": "tab-1"}`
3. Use `read_page` to discover elements: `{"action": "read_page", "filter": "interactive", "tabId": "tab-1"}`
4. Interact with elements using coordinates or refs from `read_page`

### Form Filling with Sequences

```json
{
  "action": "sequence",
  "tabId": "tab-1",
  "actions": [
    {"action": "left_click", "coordinate": [380, 250]},
    {"action": "type", "text": "user@example.com{Tab}"},
    {"action": "type", "text": "password123{Enter}"}
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

### Click Actions: Three-Tier Approach

Preference Hierarchy:

1. analyze_screenshot (PRIMARY) - Use bounding box prompt to get element centers in read_page format, directly usable
2. read_page coordinates (FIRST FALLBACK) - Get element centers via filter: "interactive", use directly without adjustment
3. Element refs (SECOND FALLBACK) - Use when both coordinate approaches fail or element has persistent ref

Decision Flow:

- Start with analyze_screenshot for visual targeting
- Fall back to read_page if bounding box detection fails or is ambiguous
- Use refs only when coordinates are unreliable or element reference needed across actions

Tips:

- If clicks fail, refresh with analyze_screenshot (coordinates stale after page changes)
- Both analyze_screenshot and read_page return center coordinates directly usable with coordinate parameter
- Element refs remain valid until page reload

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

### Content Extraction: read_page filter="text" vs Multiple Screenshots

When to use `read_page` with `filter: "text"`:

- Articles, blog posts, documentation
- Text-heavy pages where you need full content quickly
- When text formatting doesn't matter
- Extracting all headings, paragraphs, and labels in viewport

When to use multiple screenshots:

- Visual layout matters
- Need to see images, buttons, or UI elements
- Interactive elements are important
- Page has complex structure

Limitation: Both only extract content currently loaded in DOM - lazy-loaded content requires scrolling first

### Finding Elements: read_page vs find

When to use `read_page`:

- Understanding page structure
- Getting all visible elements with coordinates and refs
- Need element coordinates for immediate interaction
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

- `read_page` and `find` only see currently loaded DOM
- Content below the fold may not be loaded yet
- Scrolling triggers lazy loading, expanding the DOM

Strategy:

- Use scroll sequences, then analyze_screenshot separately to verify loaded content
- For infinite scroll: scroll in increments, use analyze_screenshot to check for new content each time

### Form Handling Best Practices

Checkboxes:

- Use `form_input` with boolean value: `{"action": "form_input", "index": 5, "value": true}`
- Don't use click for toggling (unreliable state)

Dropdowns/Selects:

- Use `form_input` with option text or value: `{"action": "form_input", "index": 3, "value": "Option 2"}`
- More reliable than clicking to open dropdown

Text Inputs:

- React apps: Use `form_input` for direct value setting
- Legacy forms: Either approach works, `type` may be more natural

## Best Practices

### Efficiency & Performance

Sequences:

- Use sequence action to combine multiple operations in one call
- Reduces total number of tool calls and round-trip time
- Example: `[click, wait, type]` instead of 3 separate calls

Use analyze_screenshot Strategically:

- After state changes to verify results
- After scrolling to see what's loaded
- Before and after critical actions (form submission, navigation)

Minimize Round Trips:

- Use `read_page` with `filter: "text"` instead of scrolling + multiple screenshots for text extraction
- Use `find` to search entire loaded DOM instead of repeated read_page calls
- Sequence multiple independent operations together

### State Management

Element Cache Behavior:

- Cached per tab: `sessionID_tabID`
- Cleared on: navigation, blank pages, tab close, session close
- Coordinates may become stale after DOM changes

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

2. "Invalid ref format"
   - Cause: Ref doesn't follow fX_ref_Y pattern
   - Fix: Use proper ref from read_page output

3. Click doesn't register
   - Cause: Stale coordinates, or clicking edge instead of center
   - Fix: Take fresh screenshot, aim for element center

Recovery Strategies:

- Always verify critical actions with screenshots
- For failed interactions: refresh page state with screenshot, retry
- For "element not found" errors: use read_page or find to relocate element
- For blank pages: take screenshot to verify, clear cache automatically happens

### Data Preservation

Before Navigation:

- Extract any data you need from current page (use read_page with filter, or screenshots)
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
- Element References: Element refs and coordinates from `read_page` are stable until the page changes
- Auto-Reconnection: Connection failures are handled transparently
- Permissions: Only `open` action requires user permission
- Shared Session State: All tabs share cookies, local storage, and login state
- Tab Isolation: Each tab has its own navigation history and DOM state
- Cache Management: Element caches are tab-specific and cleared on navigation
