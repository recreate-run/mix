#!/bin/bash

set -e

CONFIG_JSON="$1"  # JSON config as first argument
OUTPUT_NAME="$2"  # Output filename as second argument

if [[ -z "$CONFIG_JSON" || -z "$OUTPUT_NAME" ]]; then
    echo "Usage: $0 <config_json> <output_name>"
    echo "Example: $0 '{\"composition\":{...}}' 'my_video'"
    exit 1
fi

# Auto-detect workspace as parent directory (we're running from remotion_project/)
WORKSPACE_DIR="$(dirname "$PWD")"

TIMESTAMP=$(date +%s)
CONFIG_FILE="config_$TIMESTAMP.json"

# Simple directory structure - we're always in remotion_project
REMOTION_DIR="$PWD"
FINAL_OUTPUT_DIR="$WORKSPACE_DIR/output"

echo "=== Remotion Full Export ==="
echo "Config: $CONFIG_JSON"
echo "Output name: $OUTPUT_NAME"
echo "Workspace: $WORKSPACE_DIR"
echo "Remotion dir: $REMOTION_DIR"
echo ""

# Validate we have the export script
if [[ ! -f "scripts/export_video.sh" ]]; then
    echo "❌ Error: export_video.sh script not found in scripts/"
    exit 1
fi

# 1. Write config file (export_video.sh will handle the wrapping)
echo "📝 Writing config file..."
echo "$CONFIG_JSON" > "$CONFIG_FILE"

if [[ ! -f "$CONFIG_FILE" ]]; then
    echo "❌ Error: Failed to write config file"
    exit 1
fi

echo "✅ Config written to: $CONFIG_FILE"

# 2. Export video using existing script
echo ""
echo "🎬 Exporting video..."

./scripts/export_video.sh --config "$CONFIG_FILE" --output "$OUTPUT_NAME"

# 3. Check if video was created and move to final workspace location
VIDEO_FILE="output/$OUTPUT_NAME.mp4"
if [[ ! -f "$VIDEO_FILE" ]]; then
    echo "❌ Error: Video export failed - output file not found"
    echo "Expected: $VIDEO_FILE"
    echo "Contents: $(ls -la output/ 2>/dev/null || echo 'output directory not found')"
    exit 1
fi

echo "✅ Video exported successfully!"
echo "📁 Location: $VIDEO_FILE"

# 4. Move to final workspace location
echo ""
echo "📁 Organizing output..."
mkdir -p "$FINAL_OUTPUT_DIR"
TIMESTAMP_SUFFIX=$(date +%Y%m%d_%H%M%S)
FINAL_NAME="${OUTPUT_NAME}_${TIMESTAMP_SUFFIX}.mp4"
FINAL_PATH="$FINAL_OUTPUT_DIR/$FINAL_NAME"

cp "$VIDEO_FILE" "$FINAL_PATH"

if [[ ! -f "$FINAL_PATH" ]]; then
    echo "❌ Error: Failed to copy video to final location"
    exit 1
fi

echo "✅ Video export completed successfully!"
echo "📁 Final location: $FINAL_PATH"
echo "📊 File size: $(du -h "$FINAL_PATH" | cut -f1)"
echo ""
echo "FINAL_PATH:$FINAL_PATH"