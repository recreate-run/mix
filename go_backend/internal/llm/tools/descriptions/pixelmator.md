# Image editing tool

This tool provides programmatic access to Pixelmator Pro functionality for image editing workflows.

## Instructions

Use `uv run python -c "import sys; sys.path.append('$<launchdir>/tools/pixelmator'); from image_editing import *; operation_name(args)"` to execute Pixelmator Pro operations.

## Available Operations

### Document Operations

**open_document**

- Opens an image file in Pixelmator Pro
- Args: `{"filepath": "/path/to/image.jpg"}`
- Returns: Document information with id, width, height, name, resolution, color_profile

**get_document_info**  

- Returns information about the currently active document
- Args: None
- Returns: Document properties including dimensions and metadata

**close_document**

- Closes the current document
- Args: `{"save": false}` (optional)
- Returns: Boolean success status

### Image Editing Operations

**crop_document**

- Crops the current document to specified bounds
- Args: `{"bounds": [x, y, width, height]}`
- Returns: Updated document info after cropping

**resize_document**

- Resizes the current document to specified dimensions  
- Args: `{"width": 1920, "height": 1080, "algorithm": "LANCZOS"}` (algorithm optional)
- Valid algorithms: LANCZOS, BILINEAR, NEAREST
- Returns: Updated document info after resizing

### Layer Operations

**get_layers**

- Returns all layers in the current document
- Args: None
- Returns: List of layer objects with name, type, visible, opacity, blend_mode

**create_layer**

- Creates a new layer in the current document
- Args: `{"layer_type": "text", "name": "my_layer", ...}` (name optional)
- Layer types: text, shape, color
- Additional args for text: text, font_size
- Additional args for color: color (RGB array)
- Additional args for shape: shape_type
- Returns: Created layer info

**duplicate_layer**

- Duplicates an existing layer
- Args: `{"layer_name": "Layer 1"}`
- Returns: New layer info

**delete_layer**

- Deletes a layer from the current document
- Args: `{"layer_name": "Layer 1"}`
- Returns: Boolean success status

### Export Operations

**get_screenshot**

- Exports the current document to a JPEG file
- Args: `{"output_path": "/path/to/output.jpg"}`
- Returns: Export info with output_path, format, file_size, success

## Usage Examples

<examples>
// Open an image
{"operation": "open_document", "args": {"filepath": "/Users/user/image.jpg"}}

// Get document information
{"operation": "get_document_info"}

// Crop to focus area
{"operation": "crop_document", "args": {"bounds": [100, 100, 800, 600]}}

// Resize image
{"operation": "resize_document", "args": {"width": 1920, "height": 1080, "algorithm": "LANCZOS"}}

// Create a text layer
{"operation": "create_layer", "args": {"layer_type": "text", "name": "title", "text": "Hello World", "font_size": 64}}

// Export as JPEG
{"operation": "get_screenshot", "args": {"output_path": "/Users/user/output.jpg"}}

// Close document
{"operation": "close_document", "args": {"save": false}}
</examples>

## Typical Workflow

1. **Open** → Load an image file into Pixelmator Pro
2. **Inspect** → Get document info and layers
3. **Edit** → Crop, resize, or modify layers as needed
4. **Export** → Save result to desired format
5. **Cleanup** → Close document when finished

## Important Notes

- All operations require an active Pixelmator Pro application
- Most operations require a document to be open (except open_document)
- File paths must be absolute paths
- Follows fail-fast error handling - exceptions propagate immediately
- Operations are atomic - each completes fully or fails entirely
- Layer names are case-sensitive and must match exactly
- Export operations automatically create output directories if needed

## Error Handling

Common error patterns:

- "No document is currently open" - No active document in Pixelmator Pro
- "File not found" - Input file doesn't exist  
- "Layer not found" - Referenced layer doesn't exist
- "Invalid bounds" - Crop bounds exceed document dimensions
- "Export failed" - Output file wasn't created successfully

All errors include descriptive messages to help with debugging and resolution.
