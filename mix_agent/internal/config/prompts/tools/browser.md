# Browser Tool

Control a web browser for automated interactions. Provides session-isolated browser contexts with screenshot capture, element interaction, and navigation capabilities.

## Actions

### open
Navigate to a URL.

**Parameters:**
- `url` (string, required): The URL to navigate to (must be http:// or https://)

**Permission:** Requires user permission to navigate to new URLs.

**Example:**
```json
{"action": "open", "url": "https://example.com"}
```

### screenshot
Capture a screenshot of the current page, optionally with numbered element overlays for interaction.

**Parameters:**
- `withOverlay` (boolean, optional): Add numbered overlays to interactive elements (default: true)

**Returns:** Screenshot URL and list of interactive elements with indices.

**Example:**
```json
{"action": "screenshot", "withOverlay": true}
```

### read_page

Get an accessibility tree representation of currently visible page elements in the viewport.

**Parameters:**
- `interactiveOnly` (boolean, optional): Filter to show only interactive elements like buttons, links, inputs (default: false)
- `tabId` (string, optional): Specific tab ID (defaults to active tab)

**Returns:**
Accessibility tree with element references and coordinates. **Only shows content in the visible viewport** - content above/below the current scroll position is NOT included.

**Example:**
```json
{"action": "read_page", "interactiveOnly": true}
```

**Use Cases:**
- Get structured element data without screenshot image
- Find all interactive elements in current viewport
- Analyze page structure programmatically
- Lower bandwidth alternative to screenshot for element discovery
- Quick inspection of visible page content

**Key Differences from screenshot:**
- Returns JSON accessibility tree (not image)
- No visual representation or numbered overlays
- Only visible viewport elements (respects scroll position)
- Can filter to interactive-only elements
- More efficient for programmatic element analysis
- Lower bandwidth usage

**Important Notes:**
- This returns ONLY visible viewport elements
- Content above/below the current scroll position is excluded
- Use `scroll` action to reveal different page sections, then call `read_page` again
- Element indices are sequential (0, 1, 2...) for visible elements only
- Use these indices with `click`, `type`, etc. just like with screenshot

**Comparison with Other Actions:**
- Use `read_page` for programmatic element discovery
- Use `screenshot` for visual inspection and debugging
- Use `get_text` for extracting text content only
- Use `find` for searching specific elements across entire page (not just viewport)

### click
Click an interactive element by its index (from screenshot).

**Parameters:**
- `index` (integer, required): Element index from the most recent screenshot

**Example:**
```json
{"action": "click", "index": 0}
```

### type
Type text into an input element.

**Parameters:**
- `index` (integer, required): Element index from the most recent screenshot
- `text` (string, required): Text to type into the element

**Example:**
```json
{"action": "type", "index": 1, "text": "search query"}
```

**Note:** For React/Vue apps where keyboard events don't trigger state updates, use `form_input` instead.

### right_click
Right-click an interactive element by its index.

**Parameters:**
- `index` (integer, required): Element index from the most recent screenshot

**Example:**
```json
{"action": "right_click", "index": 3}
```

**Use Cases:**
- Opening context menus
- Triggering right-click-specific actions
- Testing right-click handlers

**Important:** This performs the right-click action only. Context menu items are not accessible via CDP and cannot be retrieved.

### double_click
Double-click an interactive element by its index.

**Parameters:**
- `index` (integer, required): Element index from the most recent screenshot

**Example:**
```json
{"action": "double_click", "index": 5}
```

**Use Cases:**
- Selecting text by double-clicking
- Opening items in file managers or lists
- Triggering double-click handlers

### form_input
Set form input value directly via JavaScript (for React/Vue apps).

**Parameters:**
- `index` (integer, required): Element index from the most recent screenshot (must be textbox, searchbox, or combobox)
- `value` (string, required): Value to set in the input field

**Example:**
```json
{"action": "form_input", "index": 2, "value": "user@example.com"}
```

**Use Cases:**
- Filling forms in React/Vue apps where `type` action doesn't trigger state updates
- Setting input values when keyboard events are intercepted
- Direct value manipulation for controlled components

**Important:** This action dispatches `input` and `change` events after setting the value. Only works with input elements (textbox, searchbox, combobox). For regular typing with keyboard simulation, use `type` action instead.

### go_back
Navigate backward in browser history.

**Example:**
```json
{"action": "go_back"}
```

**Returns:** New URL after navigation.

**Use Cases:**
- Going back to previous page in multi-step workflows
- Undoing navigation actions
- Exploring site navigation history

**Important:** Clears the element cache after navigation. Take a new screenshot to see interactive elements on the previous page.

### go_forward
Navigate forward in browser history.

**Example:**
```json
{"action": "go_forward"}
```

**Returns:** New URL after navigation.

**Use Cases:**
- Going forward after going back
- Navigating through previously visited pages
- Testing browser history navigation

**Important:** Clears the element cache after navigation. Take a new screenshot to see interactive elements.

### scroll
Scroll the page in a direction.

**Parameters:**
- `direction` (string, required): Scroll direction (up/down/left/right)
- `amount` (integer, optional): Number of pixels to scroll (default: 100)

**Example:**
```json
{"action": "scroll", "direction": "down", "amount": 500}
```

### upload
Upload files to a file input element.

**Parameters:**
- `index` (integer, required): Element index from the most recent screenshot (must be a file input)
- `filePath` (string, required): Path to the file to upload (can be absolute or session-relative)

**Path Resolution:**
- Absolute paths are used as-is (e.g., "/tmp/document.pdf")
- Relative paths are searched in session storage first, then the uploads directory
- Paths must be within session storage or the uploads directory for security

**Example:**
```json
{"action": "upload", "index": 2, "filePath": "document.pdf"}
```

### get_text
Extract text content from the page.

**Parameters:**
- `strategy` (string, optional): Extraction strategy - auto, article, main, body (default: auto)

**Strategies:**
- `auto`: Try article → main → body (recommended)
- `article`: Extract from <article> element only
- `main`: Extract from <main> or [role="main"] only
- `body`: Extract from entire <body>

**Returns:** Extracted text content (up to 1MB) with character count and source element.

**Example:**
```json
{"action": "get_text", "strategy": "auto"}
```

**Use Cases:**
- Extracting article content for analysis
- Reading documentation pages
- Scraping visible text from web pages

### find
Search for elements matching a keyword query across the entire DOM (not just visible elements).

**Parameters:**
- `query` (string, required): Keyword query to search for (e.g., "button Submit", "link github", "input email")

**Query Format:**
- First word can be an element role: button, link, input, textbox, checkbox, etc.
- Remaining words are matched against element text content (case-insensitive)
- Example: "button Submit" finds buttons containing "Submit"
- Example: "link" finds all links

**Returns:** List of matching elements with indices, roles, names, and coordinates.

**Example:**
```json
{"action": "find", "query": "button Submit"}
```

**Use Cases:**
- Finding elements outside the visible viewport
- Searching for specific elements without taking screenshots
- Locating elements by text content rather than visual position

### close
Close the browser and cleanup the session connection.

**Example:**
```json
{"action": "close"}
```

### tab_create
Create a new browser tab in the current session.

**Returns:** New tab ID, URL, and title.

**Example:**
```json
{"action": "tab_create"}
```

**Use Cases:**
- Comparing information across multiple pages side-by-side
- Managing multiple login sessions simultaneously
- Loading content in background while working in another tab
- Separating workflows across different tabs

### tab_list
List all open tabs in the current session.

**Returns:** List of all tabs with their IDs, URLs, titles, and active status.

**Example:**
```json
{"action": "tab_list"}
```

**Use Cases:**
- Viewing all open tabs before switching
- Checking which tab is currently active
- Getting tab IDs for tab switching or closing

### tab_switch
Switch to a different tab by its ID.

**Parameters:**
- `tabId` (string, required): The ID of the tab to switch to (from `tab_list` or `tab_create`)

**Example:**
```json
{"action": "tab_switch", "tabId": "tab-2"}
```

**Use Cases:**
- Switching between tabs to interact with different pages
- Moving focus to a specific tab for operations
- Managing multi-tab workflows

**Important:** After switching tabs, the active tab changes. All subsequent actions (screenshot, click, etc.) without an explicit `tabId` parameter will operate on the newly active tab.

### tab_close
Close a specific tab by its ID.

**Parameters:**
- `tabId` (string, required): The ID of the tab to close

**Example:**
```json
{"action": "tab_close", "tabId": "tab-2"}
```

**Use Cases:**
- Cleaning up tabs after completing a workflow
- Managing browser resources
- Closing unnecessary tabs

**Important:**
- Cannot close the last remaining tab (will return an error)
- If you close the active tab, the first remaining tab (alphabetically by ID) becomes active

## Tab-Specific Operations

All browser actions (open, screenshot, click, type, etc.) accept an optional `tabId` parameter to operate on a specific tab. If `tabId` is not provided, the action operates on the currently active tab.

**Example - Screenshot specific tab:**
```json
{"action": "screenshot", "tabId": "tab-2", "withOverlay": true}
```

**Example - Navigate in specific tab:**
```json
{"action": "open", "url": "https://example.com", "tabId": "tab-3"}
```

**Example - Click in active tab (no tabId needed):**
```json
{"action": "click", "index": 0}
```

## Usage Patterns

### Navigate and Explore
1. Open a URL with `open` action
2. Take a screenshot with `screenshot` to see interactive elements
3. Click links or interact with elements using their indices

### Form Filling
1. Navigate to a page with a form
2. Take a screenshot to identify input fields
3. Use `click` to focus fields and `type` to enter text
4. Click the submit button

### Form Filling in React/Vue Apps
1. Navigate to a SPA with a form
2. Take a screenshot to identify input fields
3. Use `form_input` to directly set values (bypasses keyboard event handling)
4. Submit the form with `click`

### Form Filling with File Upload
1. Navigate to a page with a file upload form
2. Use `screenshot` to identify the file input element
3. Use `upload` action with the file input's index and file path
4. Fill other form fields with `type` as needed
5. Submit the form with `click`

### Content Analysis
1. Navigate to a page (blog post, documentation, article)
2. Use `get_text` with strategy "auto" to extract main content
3. Process or analyze the extracted text
4. No need for multiple screenshots - get all text at once

### Finding Off-Screen Elements
1. Navigate to a page
2. Use `find` to search for elements by keyword (e.g., "button Login")
3. Use returned indices to interact with found elements
4. Useful for finding elements without scrolling through entire pages

### Data Extraction
1. Navigate to the target page
2. Use `get_text` to extract visible content
3. Use `scroll` to load more content if needed
4. Use `find` to locate specific elements for interaction

### Multi-Page Exploration
1. Navigate to a starting page
2. Take screenshots and explore links
3. Use `go_back` to return to previous pages
4. Use `go_forward` to revisit pages after going back
5. Take new screenshots after each navigation to see current page elements

### Context Menu Interactions
1. Navigate to a page with right-click functionality
2. Take a screenshot to identify target elements
3. Use `right_click` on the desired element
4. Note: Context menu items cannot be inspected via CDP

### Multi-Tab Workflows
**Scenario: Comparing prices across e-commerce sites**
1. Open first store: `{"action": "open", "url": "https://store1.com/product"}`
2. Create second tab: `{"action": "tab_create"}` → Returns `tab-2`
3. Navigate second tab: `{"action": "open", "url": "https://store2.com/product", "tabId": "tab-2"}`
4. Screenshot first tab: `{"action": "tab_switch", "tabId": "tab-1"}` then `{"action": "screenshot"}`
5. Screenshot second tab: `{"action": "tab_switch", "tabId": "tab-2"}` then `{"action": "screenshot"}`
6. Compare prices and features from both screenshots

**Scenario: Multi-account testing**
1. Open login page in tab-1: `{"action": "open", "url": "https://app.example.com/login"}`
2. Login with first account in tab-1
3. Create new tab for second account: `{"action": "tab_create"}` → Returns `tab-2`
4. Login with second account in tab-2: `{"action": "open", "url": "https://app.example.com/login", "tabId": "tab-2"}`
5. Test both accounts simultaneously by switching between tabs

**Scenario: Background loading**
1. Navigate to article: `{"action": "open", "url": "https://blog.com/article1"}`
2. Create tab for second article: `{"action": "tab_create"}` → Returns `tab-2`
3. Start loading second article: `{"action": "open", "url": "https://blog.com/article2", "tabId": "tab-2"}`
4. Read and extract text from first article while second loads
5. Switch to second tab when ready: `{"action": "tab_switch", "tabId": "tab-2"}`

## Important Notes

- **Session Isolation:** Each session gets its own browser context
- **Element Indices:** Element indices from screenshots are stable until the page changes
- **Auto-Reconnection:** Connection failures are handled transparently with automatic reconnection
- **Screenshot Storage:** Screenshots are saved to session storage and accessible via returned URLs
- **Permissions:** Only the `open` action requires explicit user permission; other actions are allowed once a URL is opened

## Error Handling

- **Navigation Timeout:** If a page takes too long to load, a timeout error will be returned
- **Element Not Found:** If an element index is invalid, an error will be returned
- **Connection Failure:** If the browser service is unavailable, a connection error will be returned
- **Invalid URL:** Only http:// and https:// URLs are accepted