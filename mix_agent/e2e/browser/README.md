# Browser E2E Tests

This directory contains end-to-end tests for browser automation capabilities. Each test sends natural language prompts to the AI agent and verifies that browser actions are executed correctly.

## Test Files Overview

### action_sequence_test.go

Tests multi-step action sequences with observable state changes.

**Mock Page:** `action_sequence.html` - Contains counter, input field, toggle button, and status display.

#### TestBrowserE2EActionSequenceSuccess
**Prompt:** "Open {url} and click the 'Increment Counter' button once, then type 'test' into the text input field, and take a screenshot."

Tests complete action workflow: navigation, clicking, typing, and screenshot capture. Verifies browser tool is called and actions complete successfully.

#### TestBrowserE2EActionSequenceFailFast
**Prompt:** "Open {url} and click the 'Increment Counter' button, then try to click a button with id 'nonexistent-button'."

Tests fail-fast behavior when encountering errors mid-sequence. Verifies that the agent detects and reports errors for nonexistent elements without crashing.

#### TestBrowserE2EActionSequenceWithScreenshots
**Prompt:** "Open {url}. Then: 1) Take a screenshot of the initial state, 2) Click the 'Toggle Element' button, 3) Take another screenshot after clicking. Tell me if you can see any differences."

Tests screenshot capture between actions to verify intermediate states. Verifies multiple screenshots are captured and stored, and state changes are observed.

---

### browser_e2e_test.go

General browser workflow and feature tests.

#### TestBrowserE2EFullWorkflow
**Prompt:** "Open google.com in the browser and take a screenshot"

**Mock Page:** google.com (external)

Tests complete user workflow: session creation, message sending, browser tool usage, and screenshot storage. Basic smoke test for browser integration.

#### TestBrowserE2ESessionIsolation
**Prompt:** "Go to the Wikipedia page on cats" (both sessions)

**Mock Page:** Wikipedia (external)

Tests that browser sessions are isolated. Each session should have separate browser contexts and file storage even when navigating to the same URL. Verifies sessions don't interfere with each other.

#### TestBrowserE2ETextExtraction
**Prompt:** "Open {url} and extract text using the {strategy} strategy"

**Mock Page:** `text_extraction.html` (referenced but not in testdata)

Tests text extraction feature with different strategies: auto, article, main, body. Verifies extracted text appears in assistant responses.

#### TestBrowserE2EDOMSearch
**Prompt:** "Open {url} and find all elements with the word 'search'"

**Mock Page:** `dom_search.html` (referenced but not in testdata)

Tests DOM search capability for finding elements by text content. Verifies search results are reported in assistant responses.

#### TestBrowserE2EFileUpload
**Prompt:** "Open {url}, then take a screenshot showing the file upload form. The file upload feature allows uploading files to file input elements."

**Mock Page:** `file_upload.html` (referenced but not in testdata)

Tests file upload feature integration. Verifies the agent can access and interact with file upload forms. Note: actual file upload not tested, only form accessibility.

#### TestBrowserE2EScreenshotURL
**Prompt:** "Go to the Wikipedia page on cats and take a screenshot"

**Mock Page:** Wikipedia (external)

Tests that screenshots are served via HTTP and accessible via URLs. Verifies screenshot URLs are returned in tool results, fetching works, content-type is image/png, and cache headers are set.

---

### drag_drop_test.go

Tests drag-and-drop operations using both index and coordinate modes.

**Mock Page:** `sortable_list.html` - Contains draggable cards and a slider with mouse event tracking.

#### TestBrowserE2EDragByIndex
**Prompt:** "Navigate to {url}. Then use the browser tool with action 'left_click_drag', fromIndex 0, and toIndex 1 to test the index-based drag functionality."

Tests index-based drag operation. Verifies drag action is executed with fromIndex/toIndex parameters and success message contains "Successfully dragged".

#### TestBrowserE2EDragByCoordinates
**Prompt:** "Navigate to {url}. Use the browser tool with action 'left_click_drag' to drag from coordinates (100, 300) to (200, 300). This will test the drag functionality."

Tests coordinate-based drag operation. Verifies drag action uses specific x,y coordinates and completes successfully.

#### TestBrowserE2EDragFailure
**Prompt:** "Navigate to {url}. Then use the browser tool with action 'left_click_drag' with ONLY fromIndex set to 0, but do NOT provide toIndex or any coordinates. This should cause a validation error."

Tests error handling for invalid drag parameters. Verifies that incomplete drag parameters (missing toIndex) produce validation errors and are properly reported.

---

### element_selection_test.go

Tests different element selection strategies: index, ref, and coordinates.

**Mock Page:** `element_selection.html` - Contains 4 buttons, text input, and dropdown with status display.

#### TestBrowserE2EElementSelectionByIndex
**Prompt:** "Open {url} and click the first button you see. Tell me what the status message says after clicking."

Tests index-based element selection (clicking first interactive element). Verifies click succeeds and status message updates are detected.

#### TestBrowserE2EElementSelectionByRef
**Prompt:** "Open {url} and read the page to see what buttons are available. Then click Button 2 specifically."

Tests ref-based element selection using element references from page reading. Verifies agent can obtain element refs and use them for targeted interactions.

#### TestBrowserE2EElementSelectionByCoordinate
**Prompt:** "Open {url}, take a screenshot, and click on Button 3 based on its visual location. Report the status after clicking."

Tests coordinate-based selection using visual guidance from screenshots. Verifies screenshot is taken and specific button is clicked based on visual location.

#### TestBrowserE2EInvalidRefHandling
**Prompt:** "Open {url} and try to click an element with id 'nonexistent-element-12345'. Report what happens."

Tests error handling for invalid/nonexistent element references. Verifies graceful error reporting when trying to interact with elements that don't exist.

#### TestBrowserE2EStaleRefAfterNavigation
**Prompt:** "Open {url}" (page A), then "Now navigate to {url2} (a different page). After loading, try to interact with elements from the previous page if you still have references to them."

**Mock Pages:** `element_selection.html`, then `action_sequence.html`

Tests handling of stale element references after page navigation. Verifies agent handles stale refs gracefully without crashes when navigating between pages.

---

### form_filling_test.go

Tests form interaction workflows including filling and submission.

**Mock Pages:** `login_form.html` (username, password, role dropdown), `success.html` (confirmation page)

#### TestBrowserE2EFormFilling
**Prompt:** "Open {url} and fill out the login form with username 'testuser', password 'testpass123', role 'Admin', then submit it. After submission, confirm you can see the success message."

Tests complete form workflow: typing into text fields, selecting dropdown options, form submission, and navigation to success page. Verifies all form actions complete and success page is reached.

#### TestBrowserE2EFormSequentialActions
**Prompt:** "Open {url} and fill just the username field with 'alice', then take a screenshot to show the field is filled."

Tests sequential form field filling without full submission. Verifies individual form fields can be filled and state is captured via screenshot.

---

### keyboard_e2e_test.go

Tests keyboard input operations including special keys and modifiers.

**Mock Page:** `keyboard_test.html` - Contains multiple input fields and textarea with keyboard event logging.

#### TestBrowserE2EKeyboardBasicKeys
**Prompts:**
- "Open {url} and press Tab key to navigate between fields"
- "Press Enter key"
- "Press Backspace key"

Tests basic keyboard operations (Tab, Enter, Backspace). Verifies unified type action is called and keyboard input results show "Successfully processed keyboard input".

#### TestBrowserE2EKeyboardModifiers
**Prompt:** "Open {url} and press Shift+Enter to create a new line"

Tests modifier key combinations (Shift+Enter). Verifies unified type action handles modifier combinations correctly using {key} syntax.

---

### mode_e2e_test.go

Tests browser automation across different browser modes.

**Mock Page:** `mode_test.html` - Simple page with click button and text input.

#### TestBrowserE2EModeCompatibility
**Prompts:** "Open {url} and take a screenshot", "Click the 'Click Me' button on the page", "Type 'test input' into the search input field"

Tests browser operations across modes: local-browser-service, remote-cdp-websocket, electron-embedded-browser. Verifies navigation, clicking, typing, and screenshots work in all supported modes.

#### TestBrowserE2EModeSwitching
**Prompt:** "Open {url} and verify the title"

Tests creating multiple sessions with same browser mode. Verifies sessions have isolated browser contexts and file storage, ensuring no cross-session interference.

#### TestBrowserE2EModeValidation
**No user prompts** (API validation tests)

Tests browser mode validation at session creation: rejects invalid modes, rejects remote mode without cdpUrl, rejects remote mode with non-WebSocket URL.

---

### tab_management_test.go

Tests multi-tab browser workflows.

**Mock Page:** `mode_test.html`

#### TestBrowserE2ETabManagement
**Prompts:**
1. "Use the Browser tool with action='tab_create' to create two new browser tabs. Report the tab IDs created."
2. "Use the Browser tool with action='open' to navigate tab-2 to {url}. Then use action='open' to navigate tab-3 to {url} as well."
3. "Use the Browser tool with action='tab_switch' and tabId='tab-2' to switch to tab-2"
4. "Use the Browser tool with action='tab_switch' and tabId='tab-3' to switch to tab-3"
5. "Use the Browser tool with action='tab_list' to list all open tabs and report which tabs exist."
6. "Use the Browser tool with action='tab_close' and tabId='tab-3' to close tab-3. Then use action='tab_list' to list all remaining tabs to verify it's closed."
7. "Use the Browser tool with action='tab_switch' and tabId='tab-3' to try switching to tab-3 (this should fail since it's closed). Report the error."

Tests complete tab lifecycle: creation, navigation, switching, listing, closing, and error handling for closed tabs. Verifies all tab management actions work correctly.

---

### timeout_recovery_test.go

Tests browser resilience with slow pages, network failures, and delays.

**Mock Pages:** `delayed.html` (with ?delay=5000 parameter), `element_selection.html`, `action_sequence.html`

#### TestBrowserE2ESlowPageLoad
**Prompt:** "Open {url} and tell me what you see after it finishes loading. Wait for the content to appear."

**Mock Page:** `delayed.html?delay=5000` - Page with 5-second delay before content appears.

Tests handling of slow-loading pages. Verifies agent waits for page load without hanging, completes processing successfully, and session remains usable afterward.

#### TestBrowserE2ENetworkFailure
**Prompts:** "Open {validUrl}", then "Try to open http://localhost:9999/nonexistent and report what happens", then "Navigate back to {validUrl}"

Tests network failure handling when navigating to unreachable URLs. Verifies errors are reported gracefully, no crashes occur, and session remains usable after network failures.

#### TestBrowserE2ESlowActionSequence
**Prompt:** "Open {url}. Then perform these steps slowly with screenshots: 1) Take initial screenshot, 2) Click Increment Counter button, 3) Take screenshot, 4) Click Toggle Element button, 5) Take final screenshot. Report on the state changes you observed."

Tests multi-step sequences with multiple screenshot captures. Verifies all actions complete despite processing delays and multiple screenshots are captured successfully.

---

### spa_search_test.go

Tests dynamic single-page application (SPA) interactions with search functionality, simulating Amazon-like behavior.

**Mock Page:** `spa_search.html` - SPA with search input, dynamic DOM updates, and product results.

#### TestBrowserE2EDynamicSPASearch
**Prompt:** "Open {url} and type 'laptops' into the search box, then press Enter to search. Tell me what search results you see."

Tests complete SPA search workflow: typing into input field, pressing Enter key to trigger search, and verifying dynamic DOM updates show search results. Validates that cache synchronization fixes work correctly on SPAs that modify the DOM after user input (similar to Amazon search behavior).

Verifies: Type action executes, Enter key press triggers search, search results appear dynamically in DOM, and browser tool completes successfully without stale element errors.

---

### spa_form_test.go

Tests dynamic single-page application (SPA) form submission with JavaScript-driven processing, simulating modern React/Vue form behavior.

**Mock Page:** `spa_registration_form.html` - Registration form with multiple input types, validation, and in-page success display.

#### TestBrowserE2EDynamicSPAFormSubmission
**Prompt:** "Open {url} and fill out the registration form with these details: name 'Alice Smith', email 'alice@test.com', password 'Pass123!', country 'USA', check the terms checkbox, and submit the form. Tell me what success message you see."

Tests complete SPA form workflow: filling text inputs, selecting dropdown options, checking checkboxes, submitting form with JavaScript, and verifying success message appears without page navigation. Validates cache synchronization on forms that dynamically update DOM after submission.

Verifies: Type actions for all text fields, dropdown selection for country, checkbox interaction for terms, form submission without navigation, success message with user data appears dynamically in DOM. Tests multiple input types (text, email, password, select, checkbox) in a single workflow.

---

## Test HTML Pages

### action_sequence.html
Interactive page with counter (increment/reset buttons), text input, toggle element (show/hide), and status display. Tests observable state changes across multiple actions.

### delayed.html
Page with configurable delay (?delay=ms parameter) that shows loading state, then reveals content after timeout. Tests handling of slow-loading pages.

### element_selection.html
Contains 4 labeled buttons, text input, dropdown select, and status area. Tests different element selection strategies (index, ref, coordinate).

### keyboard_test.html
Multiple text fields and textarea with keyboard event logging. Tracks Tab navigation, key presses, modifiers, and focus changes.

### login_form.html
Standard login form with username (text), password (password), and role (select) fields. Submits to success.html on form submission.

### mode_test.html
Simple test page with clickable button and text input. Used for testing across different browser modes.

### sortable_list.html
Drag-and-drop fixture with draggable cards, drop zone, and slider. Includes mouse event logging for drag operations.

### spa_search.html
Dynamic single-page application simulating Amazon-like product search. Features search input with Enter key handler, JavaScript-driven DOM updates, loading states, and mock product database with laptops, headphones, and keyboards. Results display dynamically with product cards showing title, price, rating, and description. Tests cache synchronization and dynamic DOM interaction.

### spa_registration_form.html
Modern SPA registration form with gradient design, multiple input types (text, email, password, select, checkbox), real-time validation, and JavaScript-driven form submission. Prevents default form submission and processes registration client-side with 600ms simulated API delay. Shows loading spinner, then displays success message with submitted user data in-page without navigation. Includes mock email validation (duplicate detection) and inline error messages. Tests cache synchronization on dynamic form processing.

### success.html
Login confirmation page displaying submitted username and role from URL parameters. Used to verify form submission.

## Running Tests

These tests require:
- Browser service running on localhost:8081
- Mix backend server running
- E2E build tag: `go test -tags=e2e ./e2e/browser/...`

Tests create sessions, send messages, wait for processing, and verify results through message history and file storage.

### Tests Using Real Websites

While most tests use local mock HTML pages for predictable, isolated testing, **3 tests use real hosted websites** to validate production readiness:

1. **TestBrowserE2EFullWorkflow** - Uses google.com for basic smoke testing
2. **TestBrowserE2ESessionIsolation** - Uses Wikipedia's cats page to test session isolation
3. **TestBrowserE2EScreenshotURL** - Uses Wikipedia's cats page to test screenshot serving

Real websites provide essential confidence that the browser automation works beyond controlled test fixtures, though they may occasionally fail due to network issues or site changes.
