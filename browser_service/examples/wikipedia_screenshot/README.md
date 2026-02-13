# Wikipedia Screenshot Example

This example demonstrates how to use the browser service to:
1. Navigate to a webpage (Wikipedia's Cat article)
2. Capture a screenshot
3. Save the screenshot to disk as a PNG file

## Prerequisites

The browser service must be running:

```bash
cd ../..
task dev
```

## Running the Example

```bash
go run main.go
```

This will:
- Connect to the browser service at `ws://localhost:8081/ws`
- Navigate to `https://en.wikipedia.org/wiki/Cat`
- Take a viewport screenshot
- Save it as `wikipedia_cats.png` in the current directory

## Output

The example will create a file `wikipedia_cats.png` containing a screenshot of the Wikipedia Cats page.

To view the screenshot:

```bash
open wikipedia_cats.png
```

## Code Walkthrough

The example demonstrates the core browser automation workflow:

1. **Connect**: Create a client connection to the browser service WebSocket
2. **Navigate**: Use `client.Navigate()` to load a URL
3. **Screenshot**: Use `client.Screenshot()` to capture the page
4. **Decode**: Base64-decode the image data from the response
5. **Save**: Write the binary PNG data to a file

All operations use context with timeout for proper resource management.
