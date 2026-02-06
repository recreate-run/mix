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

### scroll
Scroll the page in a direction.

**Parameters:**
- `direction` (string, required): Scroll direction (up/down/left/right)
- `amount` (integer, optional): Number of pixels to scroll (default: 100)

**Example:**
```json
{"action": "scroll", "direction": "down", "amount": 500}
```

### close
Close the browser and cleanup the session connection.

**Example:**
```json
{"action": "close"}
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

### Data Extraction
1. Navigate to the target page
2. Take screenshots and read element text
3. Use `scroll` to view more content
4. Take additional screenshots as needed

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