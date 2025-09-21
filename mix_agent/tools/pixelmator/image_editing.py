"""
Pixelmator Pro client for image editing automation via AppleScript.
"""

import subprocess
import os
from typing import Dict, Any, List, Optional, Tuple


def _run_applescript(script: str) -> str:
    """
    Execute AppleScript command and return result.

    Args:
        script: AppleScript code to execute

    Returns:
        str: Output from AppleScript execution

    Raises:
        RuntimeError: If AppleScript execution fails
    """
    try:
        result = subprocess.run(
            ['osascript', '-e', script],
            capture_output=True,
            text=True,
            check=True
        )
        return result.stdout.strip()
    except subprocess.CalledProcessError as e:
        raise RuntimeError(f"AppleScript failed: {e.stderr.strip()}") from e


def open_document(filepath: str) -> Dict[str, Any]:
    """
    Open an image file in Pixelmator Pro.

    Args:
        filepath: Full path to the image file

    Returns:
        Dict[str, Any]: Document info with id, width, height, name

    Raises:
        FileNotFoundError: If file doesn't exist
        RuntimeError: If Pixelmator Pro fails to open the file
    """
    if not os.path.exists(filepath):
        raise FileNotFoundError(f"File not found: {filepath}")

    script = f'tell application "Pixelmator Pro" to open POSIX file "{filepath}"'
    doc_id = _run_applescript(script)

    # Get document info after opening
    return get_document_info()


def get_document_info() -> Dict[str, Any]:
    """
    Get information about the currently active document.

    Returns:
        Dict[str, Any]: Document properties including id, name, width, height, resolution, color_profile

    Raises:
        RuntimeError: If no document is open
    """
    try:
        script = 'tell application "Pixelmator Pro" to tell front document to get {width, height, name, id, resolution, color profile}'
        result = _run_applescript(script)
        
        # Parse comma-separated list: "1536.0, 1024.0, Name, ID, 72.0, None"
        parts = [part.strip() for part in result.split(',')]
        
        width = float(parts[0])
        height = float(parts[1])
        name = parts[2]
        doc_id = parts[3]
        resolution = float(parts[4]) if parts[4] != "None" else 72.0
        color_profile = parts[5] if parts[5] != "None" else "sRGB"

        return {
            "id": doc_id,
            "name": name,
            "width": int(width),
            "height": int(height),
            "resolution": resolution,
            "color_profile": color_profile
        }

    except RuntimeError as e:
        if "front document" in str(e):
            raise RuntimeError("No document is currently open in Pixelmator Pro") from e
        raise


def get_layers() -> List[Dict[str, Any]]:
    """
    Get all layers in the current document.

    Returns:
        List[Dict[str, Any]]: List of layer dictionaries with name, type, visible, opacity, blend_mode

    Raises:
        RuntimeError: If no document is open
    """
    try:
        script = '''
        tell application "Pixelmator Pro"
            tell front document
                set layerData to {}
                repeat with i from 1 to count of layers
                    set currentLayer to layer i
                    try
                        set layerInfo to (name of currentLayer) & "|" & (visible of currentLayer) & "|" & (opacity of currentLayer) & "|" & (class of currentLayer) & "|" & (blend mode of currentLayer)
                    on error
                        try
                            set layerInfo to (name of currentLayer) & "|" & (visible of currentLayer) & "|" & (opacity of currentLayer) & "|" & (class of currentLayer) & "|normal"
                        on error
                            set layerInfo to (name of currentLayer) & "|true|100|unknown|normal"
                        end try
                    end try
                    set end of layerData to layerInfo
                end repeat
                set AppleScript's text item delimiters to "\\n"
                set layerDataString to layerData as string
                set AppleScript's text item delimiters to ""
                return layerDataString
            end tell
        end tell
        '''
        
        result = _run_applescript(script)
        
        if not result or result.strip() == "":
            return []

        layers = []
        # AppleScript returns each layer info on a separate line
        for layer_info in result.strip().split('\n'):
            if not layer_info.strip():
                continue
                
            parts = layer_info.split('|')
            if len(parts) >= 5:
                name = parts[0]
                visible = parts[1].lower() == 'true'
                opacity = float(parts[2])
                layer_class = parts[3]
                blend_mode = parts[4]
                
                layer_type = _parse_layer_type(layer_class)
                
                layers.append({
                    "name": name,
                    "type": layer_type,
                    "visible": visible,
                    "opacity": opacity,
                    "blend_mode": blend_mode
                })

        return layers

    except RuntimeError as e:
        if "front document" in str(e):
            raise RuntimeError("No document is currently open in Pixelmator Pro") from e
        raise


def _parse_layer_type(layer_class: str) -> str:
    """
    Parse Pixelmator Pro layer class to simplified type.

    Args:
        layer_class: Raw layer class from AppleScript

    Returns:
        str: Simplified layer type
    """
    class_mappings = {
        "image layer": "image",
        "text layer": "text", 
        "shape layer": "shape",
        "group layer": "group",
        "color adjustments layer": "adjustment",
        "effects layer": "effects"
    }
    
    return class_mappings.get(layer_class.lower(), "unknown")


def crop_document(bounds: Tuple[int, int, int, int]) -> Dict[str, Any]:
    """
    Crop the current document to specified bounds.

    Args:
        bounds: Crop rectangle as (x, y, width, height)

    Returns:
        Dict[str, Any]: Updated document info after cropping

    Raises:
        ValueError: If bounds are invalid
        RuntimeError: If crop fails or no document is open
    """
    x, y, width, height = bounds

    if width <= 0 or height <= 0:
        raise ValueError(f"Width and height must be positive, got: {width}x{height}")
    if x < 0 or y < 0:
        raise ValueError(f"X and Y coordinates must be non-negative, got: ({x}, {y})")

    # Get current document dimensions to validate bounds
    doc_info = get_document_info()
    if x + width > doc_info["width"] or y + height > doc_info["height"]:
        raise ValueError(f"Crop bounds {bounds} exceed document dimensions {doc_info['width']}x{doc_info['height']}")

    script = f'tell application "Pixelmator Pro" to tell front document to crop bounds {{{x}, {y}, {x + width}, {y + height}}}'
    _run_applescript(script)

    # Return updated document info
    return get_document_info()


def resize_document(width: int, height: int, algorithm: str = 'LANCZOS') -> Dict[str, Any]:
    """
    Resize the current document to specified dimensions.

    Args:
        width: New width in pixels
        height: New height in pixels
        algorithm: Resize algorithm ('LANCZOS', 'BILINEAR', 'NEAREST')

    Returns:
        Dict[str, Any]: Updated document info after resizing

    Raises:
        ValueError: If dimensions are invalid
        RuntimeError: If resize fails or no document is open
    """
    if width <= 0 or height <= 0:
        raise ValueError(f"Width and height must be positive, got: {width}x{height}")

    valid_algorithms = {'LANCZOS', 'BILINEAR', 'NEAREST'}
    if algorithm not in valid_algorithms:
        raise ValueError(f"Invalid algorithm: {algorithm}. Must be one of {valid_algorithms}")

    # Pixelmator Pro uses different algorithm names
    algorithm_mapping = {
        'LANCZOS': 'Lanczos',
        'BILINEAR': 'bilinear',
        'NEAREST': 'nearest neighbor'
    }
    
    pixelmator_algorithm = algorithm_mapping[algorithm]
    script = f'tell application "Pixelmator Pro" to tell front document to resize to dimensions {{{width}, {height}}} algorithm "{pixelmator_algorithm}"'
    
    _run_applescript(script)

    # Return updated document info
    return get_document_info()


def get_screenshot(output_path: str) -> Dict[str, Any]:
    """
    Export the current document to a JPEG file.

    Args:
        output_path: Path for the exported JPEG file

    Returns:
        Dict[str, Any]: Export info with output_path, format, file_size, success

    Raises:
        RuntimeError: If export fails or no document is open
    """
    # Ensure output directory exists
    output_dir = os.path.dirname(output_path)
    if output_dir and not os.path.exists(output_dir):
        os.makedirs(output_dir)

    # Export as JPEG with default quality (80%)
    script = f'tell application "Pixelmator Pro" to export front document to POSIX file "{output_path}" as JPEG with compression factor 0.8'

    _run_applescript(script)

    # Check if file was created and get size
    if not os.path.exists(output_path):
        raise RuntimeError(f"Export failed - file was not created: {output_path}")

    file_size = os.path.getsize(output_path)

    return {
        "output_path": output_path,
        "format": "JPEG",
        "file_size": file_size,
        "success": True
    }


def close_document(save: bool = False) -> bool:
    """
    Close the current document.

    Args:
        save: Whether to save before closing

    Returns:
        bool: True if closed successfully

    Raises:
        RuntimeError: If no document is open or close fails
    """
    try:
        if save:
            script = '''
            tell application "Pixelmator Pro"
                tell front document
                    save
                    close
                end tell
            end tell
            '''
        else:
            script = 'tell application "Pixelmator Pro" to close front document'
        
        _run_applescript(script)
        return True

    except RuntimeError as e:
        if "front document" in str(e):
            raise RuntimeError("No document is currently open in Pixelmator Pro") from e
        raise


def create_layer(layer_type: str, name: Optional[str] = None, **kwargs) -> Dict[str, Any]:
    """
    Create a new layer in the current document.

    Args:
        layer_type: Type of layer ('text', 'shape', 'color')
        name: Layer name (auto-generated if None)
        **kwargs: Layer-specific properties

    Returns:
        Dict[str, Any]: Layer info dictionary

    Raises:
        ValueError: If layer_type is invalid
        RuntimeError: If creation fails or no document is open
    """
    valid_types = {'text', 'shape', 'color'}
    if layer_type not in valid_types:
        raise ValueError(f"Invalid layer_type: {layer_type}. Must be one of {valid_types}")

    if name is None:
        name = f"{layer_type}_layer"

    if layer_type == 'text':
        text = kwargs.get('text', 'Sample Text')
        font_size = kwargs.get('font_size', 48)
        script = f'tell application "Pixelmator Pro" to tell front document to make new text layer with properties {{name:"{name}", text:"{text}", font size:{font_size}}}'
    
    elif layer_type == 'color':
        color = kwargs.get('color', (1.0, 1.0, 1.0, 1.0))  # White by default
        r, g, b, a = color
        script = f'tell application "Pixelmator Pro" to tell front document to make new color layer with properties {{name:"{name}", color:{{{r}, {g}, {b}}}}}'
    
    elif layer_type == 'shape':
        shape_type = kwargs.get('shape_type', 'rectangle')
        script = f'tell application "Pixelmator Pro" to tell front document to make new {shape_type} shape layer with properties {{name:"{name}"}}'

    _run_applescript(script)

    # Return info about the created layer
    layers = get_layers()
    for layer in layers:
        if layer['name'] == name:
            return layer
    
    raise RuntimeError(f"Failed to create layer: {name}")


def duplicate_layer(layer_name: str) -> Dict[str, Any]:
    """
    Duplicate an existing layer.

    Args:
        layer_name: Name of the layer to duplicate

    Returns:
        Dict[str, Any]: New layer info dictionary with 'index' field for unique identification

    Raises:
        ValueError: If layer doesn't exist
        RuntimeError: If duplication fails
    """
    # Verify layer exists and duplicate in single operation
    script = f'''
    tell application "Pixelmator Pro"
        tell front document
            if not (exists layer "{layer_name}") then
                error "Layer not found"
            end if
            set layerCountBefore to count of layers
            duplicate layer "{layer_name}"
            set layerCountAfter to count of layers
            if layerCountAfter <= layerCountBefore then
                error "Duplication failed"
            end if
            -- The duplicated layer is typically inserted at the top (index 1)
            set duplicatedLayer to layer 1
            try
                return (name of duplicatedLayer) & "|" & (visible of duplicatedLayer) & "|" & (opacity of duplicatedLayer) & "|" & (class of duplicatedLayer) & "|" & (blend mode of duplicatedLayer) & "|1"
            on error
                return (name of duplicatedLayer) & "|true|100|unknown|normal|1"
            end try
        end tell
    end tell
    '''
    
    try:
        result = _run_applescript(script)
        parts = result.split('|')
        
        if len(parts) >= 6:
            return {
                "name": parts[0],
                "type": _parse_layer_type(parts[3]),
                "visible": parts[1].lower() == 'true',
                "opacity": float(parts[2]),
                "blend_mode": parts[4],
                "index": int(parts[5])
            }
        else:
            raise RuntimeError(f"Failed to get duplicated layer info: {result}")
            
    except RuntimeError as e:
        if "Layer not found" in str(e):
            raise ValueError(f"Layer '{layer_name}' not found") from e
        raise RuntimeError(f"Failed to duplicate layer: {layer_name}") from e


def delete_layer(layer_name: str, layer_index: Optional[int] = None) -> bool:
    """
    Delete a layer from the current document.

    Args:
        layer_name: Name of the layer to delete
        layer_index: Optional index of the layer (1-based, for layers with duplicate names)

    Returns:
        bool: True if deleted successfully

    Raises:
        ValueError: If layer doesn't exist
        RuntimeError: If deletion fails
    """
    if layer_index is not None:
        # Delete by index (more reliable for duplicate names)
        script = f'''
        tell application "Pixelmator Pro"
            tell front document
                if {layer_index} > count of layers then
                    error "Layer index out of range"
                end if
                set targetLayer to layer {layer_index}
                if name of targetLayer is not "{layer_name}" then
                    error "Layer name mismatch at index"
                end if
                delete layer {layer_index}
                return "deleted"
            end tell
        end tell
        '''
    else:
        # Delete by name (original behavior)
        script = f'''
        tell application "Pixelmator Pro"
            tell front document
                if not (exists layer "{layer_name}") then
                    error "Layer not found"
                end if
                delete layer "{layer_name}"
                return "deleted"
            end tell
        end tell
        '''
    
    try:
        result = _run_applescript(script)
        return result.strip() == "deleted"
        
    except RuntimeError as e:
        if "Layer not found" in str(e) or "Layer index out of range" in str(e) or "Layer name mismatch" in str(e):
            raise ValueError(f"Layer '{layer_name}' not found") from e
        raise RuntimeError(f"Failed to delete layer: {layer_name}") from e


    """
    Export the current viewport view to a file.

    Args:
        output_path: Path for the exported file
        format: Export format ('PNG', 'JPEG', 'TIFF')

    Returns:
        Dict[str, Any]: Export info with output_path, format, file_size, success

    Raises:
        ValueError: If format is unsupported
        RuntimeError: If export fails or no document is open
    """
    valid_formats = {'PNG', 'JPEG', 'TIFF'}
    if format not in valid_formats:
        raise ValueError(f"Invalid format: {format}. Must be one of {valid_formats}")

    # Ensure output directory exists
    output_dir = os.path.dirname(output_path)
    if output_dir and not os.path.exists(output_dir):
        os.makedirs(output_dir)

    script = f'tell application "Pixelmator Pro" to export current view to POSIX file "{output_path}" as {format}'
    _run_applescript(script)

    # Check if file was created and get size
    if not os.path.exists(output_path):
        raise RuntimeError(f"Export failed - file was not created: {output_path}")

    file_size = os.path.getsize(output_path)

    return {
        "output_path": output_path,
        "format": format,
        "file_size": file_size,
        "success": True
    }